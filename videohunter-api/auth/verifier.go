// Package auth verifies the Cognito ID tokens that the browser sends with the
// requests of a logged in user.
//
// The tokens are verified inside the Lambda instead of by an API Gateway
// authorizer on purpose. The API Gateway resource is shared with every
// download and every video page, and it must not change: a mistake there takes
// the whole service down. New protected routes simply call Verify.
package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrUnauthorized wraps every rejected token. Handlers map it to a 401 so the
// browser starts a new login instead of retrying.
var ErrUnauthorized = errors.New("unauthorized")

const (
	// The signing keys rotate rarely, so caching them for a few hours keeps
	// the JWKS endpoint off the hot path without delaying a rotation by more
	// than half a day.
	jwksCacheTTL = 12 * time.Hour

	// Cognito and the Lambda clocks can differ by a few seconds.
	tokenLeeway = time.Minute

	issuerFormat = "https://cognito-idp.%s.amazonaws.com/%s"
)

// Claims is what the API reads out of a Cognito ID token. Sub is the stable
// user id and the only identifier that reaches DynamoDB; the name and email
// are used to label chat messages.
type Claims struct {
	Sub   string
	Email string
	Name  string
}

// DisplayName is what a chat room shows next to a message. It never falls back
// to the user id, so the identifier is not exposed in public rooms.
func (c Claims) DisplayName() string {
	if name := strings.TrimSpace(c.Name); name != "" {
		return name
	}

	if email := strings.TrimSpace(c.Email); email != "" {
		if at := strings.Index(email, "@"); at > 0 {
			return email[:at]
		}
		return email
	}

	return "Anonymous"
}

// Verifier validates Cognito ID tokens against the user pool's JWKS.
type Verifier struct {
	issuer     string
	clientID   string
	keysURL    string
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	keysUntil time.Time
}

// NewVerifier builds a verifier for one Cognito user pool and app client.
func NewVerifier(region, userPoolID, clientID string) *Verifier {
	issuer := fmt.Sprintf(issuerFormat, region, userPoolID)
	return newVerifier(issuer, clientID, &http.Client{Timeout: 5 * time.Second})
}

// newVerifier takes the issuer directly so tests can point it at a local JWKS.
func newVerifier(issuer, clientID string, httpClient *http.Client) *Verifier {
	return &Verifier{
		issuer:     issuer,
		clientID:   clientID,
		keysURL:    issuer + "/.well-known/jwks.json",
		httpClient: httpClient,
	}
}

// Verify checks the signature, issuer, audience and validity window of a
// Cognito ID token and returns the claims the API cares about.
func (v *Verifier) Verify(rawToken string) (*Claims, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, fmt.Errorf("%w: empty token", ErrUnauthorized)
	}

	parsed, err := jwt.Parse(rawToken, v.keyFunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.clientID),
		jwt.WithLeeway(tokenLeeway),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("%w: unexpected claims type", ErrUnauthorized)
	}

	// Only an ID token carries the audience and the profile claims this API
	// uses. An access token would otherwise pass the audience check above.
	if tokenUse, _ := claims["token_use"].(string); tokenUse != "id" {
		return nil, fmt.Errorf("%w: not an ID token", ErrUnauthorized)
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("%w: token has no subject", ErrUnauthorized)
	}

	return &Claims{
		Sub:   sub,
		Email: stringClaim(claims, "email"),
		Name:  stringClaim(claims, "name"),
	}, nil
}

// keyFunc resolves the token's key id against the cached JWKS, refreshing the
// cache once when a key id is not known yet (a rotation).
func (v *Verifier) keyFunc(token *jwt.Token) (interface{}, error) {
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, fmt.Errorf("%w: token has no key id", ErrUnauthorized)
	}

	if key := v.cachedKey(kid); key != nil {
		return key, nil
	}

	if err := v.refreshKeys(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	if key := v.cachedKey(kid); key != nil {
		return key, nil
	}

	return nil, fmt.Errorf("%w: unknown signing key", ErrUnauthorized)
}

func (v *Verifier) cachedKey(kid string) *rsa.PublicKey {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if time.Now().After(v.keysUntil) {
		return nil
	}

	return v.keys[kid]
}

func (v *Verifier) refreshKeys() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	response, err := v.httpClient.Get(v.keysURL)
	if err != nil {
		return fmt.Errorf("fetching JWKS: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching JWKS: status %d", response.StatusCode)
	}

	var document struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		return fmt.Errorf("decoding JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(document.Keys))

	for _, key := range document.Keys {
		if key.Kty != "RSA" || key.Kid == "" {
			continue
		}

		publicKey, err := rsaPublicKey(key.N, key.E)
		if err != nil {
			return fmt.Errorf("decoding JWKS key %q: %w", key.Kid, err)
		}

		keys[key.Kid] = publicKey
	}

	if len(keys) == 0 {
		return errors.New("JWKS has no usable RSA keys")
	}

	v.keys = keys
	v.keysUntil = time.Now().Add(jwksCacheTTL)

	return nil
}

// rsaPublicKey builds an RSA public key from the base64url modulus and
// exponent of a JWKS entry.
func rsaPublicKey(modulus, exponent string) (*rsa.PublicKey, error) {
	n, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}

	e, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil {
		return nil, err
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(n),
		E: int(new(big.Int).SetBytes(e).Int64()),
	}, nil
}

// BearerToken extracts the token from an Authorization header, accepting the
// documented "Bearer " prefix and a bare token.
func BearerToken(header string) string {
	header = strings.TrimSpace(header)

	if len(header) >= 7 && strings.EqualFold(header[:7], "bearer ") {
		return strings.TrimSpace(header[7:])
	}

	// A header that is nothing but the scheme carries no token.
	if strings.EqualFold(header, "bearer") {
		return ""
	}

	return header
}

func stringClaim(claims jwt.MapClaims, name string) string {
	value, _ := claims[name].(string)
	return value
}
