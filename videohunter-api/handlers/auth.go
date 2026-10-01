package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/victoraldir/myvideohunterapi/auth"
)

// TokenVerifier is the part of the auth package the account and chat handlers
// use. It is an interface so tests can answer without signing a token.
type TokenVerifier interface {
	Verify(rawToken string) (*auth.Claims, error)
}

// headerValue reads a request header without depending on its casing: API
// Gateway passes header names through as the client sent them.
func headerValue(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}

	return ""
}

func unauthorizedResponse() events.APIGatewayProxyResponse {
	return jsonResponse(http.StatusUnauthorized, "Please log in to continue.")
}

// jsonValue renders a successful response body.
func jsonValue(statusCode int, value interface{}) events.APIGatewayProxyResponse {
	body, err := json.Marshal(value)
	if err != nil {
		// Only reachable if a response type stops being marshalable, which is
		// a bug rather than a request problem.
		return jsonResponse(http.StatusInternalServerError, "Something went wrong. Please try again.")
	}

	return jsonResponseWithBody(statusCode, string(body))
}

func noContentResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusNoContent,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
	}
}

// decodeBody parses a JSON request body, treating an empty body as invalid
// input rather than as an empty object.
func decodeBody(body string, target interface{}) error {
	if strings.TrimSpace(body) == "" {
		return errors.New("empty body")
	}

	return json.Unmarshal([]byte(body), target)
}
