package handlers

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigHandler_PublishesTheLoginAndChatEndpoints(t *testing.T) {

	handler := NewConfigHandler("videohunter-auth.auth.us-east-1.amazoncognito.com", "client-1", "wss://chat.example.com/prod")

	response, err := handler.Handle(events.APIGatewayProxyRequest{})

	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "public, max-age=300", response.Headers["Cache-Control"])

	body := response.Body
	assert.Contains(t, body, `"enabled":true`)
	assert.Contains(t, body, `"client_id":"client-1"`)
	assert.Contains(t, body, `"authorize_url":"https://videohunter-auth.auth.us-east-1.amazoncognito.com/oauth2/authorize"`)
	assert.Contains(t, body, `"token_url":"https://videohunter-auth.auth.us-east-1.amazoncognito.com/oauth2/token"`)
	assert.Contains(t, body, `"logout_url":"https://videohunter-auth.auth.us-east-1.amazoncognito.com/logout"`)
	assert.Contains(t, body, `"chat_ws_url":"wss://chat.example.com/prod"`)
}

func TestConfigHandler_ReportsLoginAsDisabledWhenCognitoIsNotConfigured(t *testing.T) {

	handler := NewConfigHandler("", "", "")

	response, err := handler.Handle(events.APIGatewayProxyRequest{})

	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	// The site reads this and hides the login rather than offering a broken
	// one, which is what makes a partial deploy safe.
	assert.Contains(t, response.Body, `"enabled":false`)
	assert.Contains(t, response.Body, `"authorize_url":""`)
}
