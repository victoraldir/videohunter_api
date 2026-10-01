package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer   = "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_test"
	testClientID = "testclientid"
	testKid      = "test-key-1"
)

// testKeys is a throwaway RSA key pair used to sign the tokens in this file.
// Nothing here talks to Cognito.
type testKeys struct {
	private *rsa.PrivateKey
	server  *httptest.Server

	mu        sync.Mutex
	served    *rsa.PublicKey
	servedKid string
}

func newTestKeys(t *testing.T) *testKeys {
	t.Helper()

	private, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keys := &testKeys{private: private, served: &private.PublicKey, servedKid: testKid}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		keys.mu.Lock()
		served, kid := keys.served, keys.servedKid
		keys.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwksDocument(kid, served))
	}))

	t.Cleanup(server.Close)

	keys.server = server

	return keys
}

// publish swaps the key the test JWKS endpoint hands out, to stand in for a
// Cognito key rotation.
func (k *testKeys) publish(kid string, key *rsa.PublicKey) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.servedKid = kid
	k.served = key
}

// withJWKS points the verifier's JWKS URL at the local test server.
func (k *testKeys) withJWKS(verifier *Verifier) *Verifier {
	verifier.keysURL = k.server.URL
	return verifier
}

func (k *testKeys) verifier() *Verifier {
	return newVerifier(testIssuer, testClientID, &http.Client{Timeout: 5 * time.Second})
}

func jwksDocument(kid string, key *rsa.PublicKey) map[string]interface{} {
	return map[string]interface{}{
		"keys": []map[string]string{{
			"kid": kid,
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}},
	}
}

type tokenOptions struct {
	issuer    string
	clientID  string
	tokenUse  string
	sub       string
	email     string
	name      string
	kid       string
	expiresAt time.Time
	signKey   *rsa.PrivateKey
	omitSub   bool
}

func (k *testKeys) sign(t *testing.T, options tokenOptions) string {
	t.Helper()

	if options.issuer == "" {
		options.issuer = testIssuer
	}
	if options.clientID == "" {
		options.clientID = testClientID
	}
	if options.tokenUse == "" {
		options.tokenUse = "id"
	}
	if options.sub == "" {
		options.sub = "user-sub-1"
	}
	if options.kid == "" {
		options.kid = testKid
	}
	if options.expiresAt.IsZero() {
		options.expiresAt = time.Now().Add(time.Hour)
	}
	if options.signKey == nil {
		options.signKey = k.private
	}

	claims := jwt.MapClaims{
		"iss":       options.issuer,
		"aud":       options.clientID,
		"token_use": options.tokenUse,
		"exp":       options.expiresAt.Unix(),
		"iat":       time.Now().Add(-time.Minute).Unix(),
	}

	if !options.omitSub {
		claims["sub"] = options.sub
	}
	if options.email != "" {
		claims["email"] = options.email
	}
	if options.name != "" {
		claims["name"] = options.name
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = options.kid

	signed, err := token.SignedString(options.signKey)
	require.NoError(t, err)

	return signed
}

func TestVerify_AcceptsAValidIdToken(t *testing.T) {
	keys := newTestKeys(t)
	verifier := keys.withJWKS(keys.verifier())

	token := keys.sign(t, tokenOptions{email: "victor@example.com", name: "Victor"})

	claims, err := verifier.Verify(token)

	require.NoError(t, err)
	assert.Equal(t, "user-sub-1", claims.Sub)
	assert.Equal(t, "victor@example.com", claims.Email)
	assert.Equal(t, "Victor", claims.Name)
}

func TestVerify_RejectsBadTokens(t *testing.T) {
	keys := newTestKeys(t)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tampered := keys.sign(t, tokenOptions{})
	tampered = tampered[:len(tampered)-3] + "aaa"

	cases := map[string]string{
		"empty":           "",
		"garbage":         "not-a-jwt",
		"tampered":        tampered,
		"expired":         keys.sign(t, tokenOptions{expiresAt: time.Now().Add(-time.Hour)}),
		"wrong issuer":    keys.sign(t, tokenOptions{issuer: "https://evil.example.com/pool"}),
		"wrong audience":  keys.sign(t, tokenOptions{clientID: "someone-elses-app"}),
		"access token":    keys.sign(t, tokenOptions{tokenUse: "access"}),
		"no subject":      keys.sign(t, tokenOptions{omitSub: true}),
		"foreign signing": keys.sign(t, tokenOptions{signKey: otherKey}),
		"unknown key id":  keys.sign(t, tokenOptions{kid: "rotated-away"}),
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			verifier := keys.withJWKS(keys.verifier())

			claims, err := verifier.Verify(token)

			assert.Nil(t, claims)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrUnauthorized), "expected ErrUnauthorized, got %v", err)
		})
	}
}

func TestVerify_RefreshesTheKeyCacheWhenAKeyRotates(t *testing.T) {
	keys := newTestKeys(t)
	verifier := keys.withJWKS(keys.verifier())

	// Warm the cache with the original key.
	_, err := verifier.Verify(keys.sign(t, tokenOptions{}))
	require.NoError(t, err)

	// Cognito rotates: a new key id is published and the old one is gone.
	rotated, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keys.publish("rotated", &rotated.PublicKey)

	claims, err := verifier.Verify(keys.sign(t, tokenOptions{kid: "rotated", signKey: rotated}))

	require.NoError(t, err)
	assert.Equal(t, "user-sub-1", claims.Sub)
}

func TestDisplayName(t *testing.T) {
	cases := map[string]struct {
		claims Claims
		want   string
	}{
		"prefers the profile name": {claims: Claims{Name: "Victor", Email: "victor@example.com"}, want: "Victor"},
		"falls back to the email":  {claims: Claims{Email: "victor@example.com"}, want: "victor"},
		"blank name yields email":  {claims: Claims{Name: "  ", Email: "victor@example.com"}, want: "victor"},
		"never exposes the sub":    {claims: Claims{Sub: "secret-sub"}, want: "Anonymous"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.want, testCase.claims.DisplayName())
		})
	}
}

func TestBearerToken(t *testing.T) {
	assert.Equal(t, "abc", BearerToken("Bearer abc"))
	assert.Equal(t, "abc", BearerToken("bearer abc"))
	assert.Equal(t, "abc", BearerToken("  Bearer   abc  "))
	assert.Equal(t, "abc", BearerToken("abc"))
	assert.Equal(t, "", BearerToken(""))
	assert.Equal(t, "", BearerToken("Bearer   "))
}
