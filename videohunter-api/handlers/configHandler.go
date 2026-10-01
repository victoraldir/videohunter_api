package handlers

import (
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// ConfigHandler serves the public configuration the browser needs before it can
// log in or open a chat room: where the Cognito hosted UI lives, which app
// client to use, and the websocket endpoint of the chat.
//
// None of this is secret, and serving it from the deployed stack means the
// static site has no build time configuration to drift from the infrastructure.
type ConfigHandler struct {
	domain    string
	clientID  string
	chatWsURL string
}

func NewConfigHandler(domain, clientID, chatWsURL string) *ConfigHandler {
	return &ConfigHandler{domain: domain, clientID: clientID, chatWsURL: chatWsURL}
}

type cognitoConfig struct {
	Enabled      bool   `json:"enabled"`
	ClientId     string `json:"client_id"`
	AuthorizeUrl string `json:"authorize_url"`
	TokenUrl     string `json:"token_url"`
	LogoutUrl    string `json:"logout_url"`
}

type publicConfig struct {
	Cognito   cognitoConfig `json:"cognito"`
	ChatWsUrl string        `json:"chat_ws_url"`
}

func (h *ConfigHandler) Handle(events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	enabled := h.domain != "" && h.clientID != ""

	config := publicConfig{
		Cognito: cognitoConfig{
			Enabled:  enabled,
			ClientId: h.clientID,
		},
		ChatWsUrl: h.chatWsURL,
	}

	if enabled {
		base := "https://" + h.domain

		config.Cognito.AuthorizeUrl = base + "/oauth2/authorize"
		config.Cognito.TokenUrl = base + "/oauth2/token"
		config.Cognito.LogoutUrl = base + "/logout"
	} else {
		slog.Warn("Cognito is not configured, so the site will not offer a login")
	}

	response := jsonValue(http.StatusOK, config)
	// The token endpoint and the client id change only on a redeploy, and a
	// short cache keeps the browser from asking on every page view.
	response.Headers["Cache-Control"] = "public, max-age=300"

	return response, nil
}
