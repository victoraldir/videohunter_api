package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/victoraldir/myvideohunterapi/adapters/apigateway"
	"github.com/victoraldir/myvideohunterapi/auth"
	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/repositories"
	"github.com/victoraldir/myvideohunterapi/utils"
)

const (
	// maxChatMessageLength keeps a message inside one bubble and well below
	// the websocket frame limit.
	maxChatMessageLength = 500

	// The room is loaded with the last messages, so a page that has been open
	// all day is not flooded with a month of history.
	recentMessageLimit = 50

	// A signed in user can write ten messages a minute, which is enough for a
	// conversation and useless for spam.
	maxMessagesPerMinute = 10

	// guestNotice is what a connection without an account is told when it
	// tries to do anything other than read.
	guestNotice = "Log in to join the conversation."
)

// ConnectionManagerFactory builds the client that posts back to a websocket.
// It is injected so tests never touch API Gateway.
type ConnectionManagerFactory func(domainName, stage string) apigateway.ConnectionManager

// chatEnvelope is what the browser sends over the socket. Every action carries
// the video id, which is how the handler finds the connection's room without a
// second index.
type chatEnvelope struct {
	Action    string `json:"action"`
	VideoId   string `json:"video_id"`
	Text      string `json:"text"`
	MessageId string `json:"message_id"`
}

// ChatHandler serves the room attached to a video page.
//
// Reading is open to everyone: a room is public to the video page it sits on,
// which is why a connection may exist without an account. Writing is not: the
// token is checked when the socket connects, and every message is attributed
// from the stored connection rather than from anything the browser claims, so
// nobody can post as someone else.
type ChatHandler struct {
	Verifier    TokenVerifier
	Rooms       repositories.ChatRepository
	Profiles    repositories.UserDataRepository
	Connections ConnectionManagerFactory
}

func NewChatHandler(
	verifier TokenVerifier,
	rooms repositories.ChatRepository,
	profiles repositories.UserDataRepository,
	connections ConnectionManagerFactory,
) *ChatHandler {
	return &ChatHandler{
		Verifier:    verifier,
		Rooms:       rooms,
		Profiles:    profiles,
		Connections: connections,
	}
}

func (h *ChatHandler) Handle(request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {

	switch request.RequestContext.RouteKey {
	case "$connect":
		return h.connect(request), nil
	case "$disconnect":
		// Nothing to clean up here: the connection row expires on its own, and
		// a broadcast that reaches a closed socket prunes it immediately.
		return websocketResponse(http.StatusOK), nil
	default:
		return h.message(request), nil
	}
}

func (h *ChatHandler) connect(request events.APIGatewayWebsocketProxyRequest) events.APIGatewayProxyResponse {

	videoId := strings.TrimSpace(request.QueryStringParameters["videoId"])
	if videoId == "" {
		slog.Info("Rejecting a chat connection without a video id")
		return websocketResponse(http.StatusBadRequest)
	}

	connection := domain.ChatConnection{ConnectionId: request.RequestContext.ConnectionID}

	// A connection without a token is a guest. It may watch the room; every
	// action that writes checks SignedIn before touching anything.
	if token := strings.TrimSpace(request.QueryStringParameters["Authorization"]); token != "" {
		claims, err := h.Verifier.Verify(auth.BearerToken(token))
		if err != nil {
			// A token that is present but unusable is an error rather than a
			// guest: the browser should log in again, not silently lose the
			// ability to post.
			slog.Info("Rejecting a chat connection with an unusable token", "error", err)
			return websocketResponse(http.StatusUnauthorized)
		}

		profile, err := h.Profiles.EnsureProfile(claims.Sub)
		if err != nil {
			slog.Error("Could not load the profile of a chat connection", "error", err)
			return websocketResponse(http.StatusInternalServerError)
		}

		connection.UserId = claims.Sub
		connection.Author = profile.Nickname
	}

	if err := h.Rooms.SaveConnection(videoId, connection); err != nil {
		slog.Error("Could not record a chat connection", "error", err)
		return websocketResponse(http.StatusInternalServerError)
	}

	return websocketResponse(http.StatusOK)
}

func (h *ChatHandler) message(request events.APIGatewayWebsocketProxyRequest) events.APIGatewayProxyResponse {

	var envelope chatEnvelope

	if err := json.Unmarshal([]byte(request.Body), &envelope); err != nil {
		slog.Info("Rejecting an unreadable chat message", "error", err)
		return websocketResponse(http.StatusOK)
	}

	envelope.VideoId = strings.TrimSpace(envelope.VideoId)
	connectionId := request.RequestContext.ConnectionID
	manager := h.Connections(request.RequestContext.DomainName, request.RequestContext.Stage)

	if envelope.VideoId == "" {
		h.sendTo(manager, connectionId, errorFrame("This message is missing its room."))
		return websocketResponse(http.StatusOK)
	}

	// The room is found from the video id and the connection id. The user id
	// comes from this row, never from the payload.
	connection, err := h.Rooms.GetConnection(envelope.VideoId, connectionId)
	if err != nil {
		slog.Error("Could not look up a chat connection", "error", err)
		h.sendTo(manager, connectionId, errorFrame("Something went wrong. Please try again."))
		return websocketResponse(http.StatusOK)
	}

	if connection == nil {
		// The connection is not part of this room: it expired, or it was
		// never registered. Stay quiet; the browser reconnects on its own.
		slog.Warn("Ignoring a message from an unknown chat connection", "videoId", envelope.VideoId)
		return websocketResponse(http.StatusOK)
	}

	switch envelope.Action {
	case "recent":
		// Reading is open to guests.
		h.recent(manager, connectionId, envelope.VideoId, connection)

	case "send":
		if !h.requireAccount(manager, connection, connectionId) {
			break
		}
		h.send(manager, connection, envelope.VideoId, envelope.Text)

	case "delete":
		if !h.requireAccount(manager, connection, connectionId) {
			break
		}
		h.deleteMessage(manager, connection, envelope.VideoId, envelope.MessageId)

	case "report":
		if !h.requireAccount(manager, connection, connectionId) {
			break
		}
		h.reportMessage(manager, connectionId, connection, envelope.VideoId, envelope.MessageId)

	default:
		h.sendTo(manager, connectionId, errorFrame("Unknown action."))
	}

	return websocketResponse(http.StatusOK)
}

// requireAccount refuses an action that writes when the connection is a guest,
// answering with what the browser should do about it. It returns false when
// the caller must not continue.
func (h *ChatHandler) requireAccount(manager apigateway.ConnectionManager, connection *domain.ChatConnection, connectionId string) bool {

	if connection.SignedIn() {
		return true
	}

	h.sendTo(manager, connectionId, errorFrame(guestNotice))

	return false
}

func (h *ChatHandler) recent(manager apigateway.ConnectionManager, connectionId, videoId string, connection *domain.ChatConnection) {

	messages, err := h.Rooms.RecentMessages(videoId, recentMessageLimit)
	if err != nil {
		slog.Error("Could not load chat history", "error", err)
		h.sendTo(manager, connectionId, errorFrame("We could not load the chat. Please try again."))
		return
	}

	h.sendTo(manager, connectionId, map[string]interface{}{
		"action":   "recent",
		"messages": messages,
		"user_id":  connection.UserId,
		// The name this connection writes under, so the browser can say who
		// it is without a second request. Empty for a guest.
		"nickname": connection.Author,
	})
}

func (h *ChatHandler) send(manager apigateway.ConnectionManager, connection *domain.ChatConnection, videoId, rawText string) {

	text := sanitizeText(rawText)
	if text == "" || len([]rune(text)) > maxChatMessageLength {
		h.sendTo(manager, connection.ConnectionId, errorFrame("Messages are 1 to 500 characters long."))
		return
	}

	allowed, err := h.Rooms.AllowMessage(connection.UserId, maxMessagesPerMinute)
	if err != nil {
		slog.Error("Could not check the chat rate limit", "error", err)
		h.sendTo(manager, connection.ConnectionId, errorFrame("Something went wrong. Please try again."))
		return
	}

	if !allowed {
		h.sendTo(manager, connection.ConnectionId, errorFrame("You are sending messages too quickly. Wait a moment."))
		return
	}

	now := time.Now().UTC()

	message := &domain.ChatMessage{
		Id:        utils.NewSortableId(now),
		UserId:    connection.UserId,
		Author:    connection.Author,
		Text:      text,
		CreatedAt: now.Format(time.RFC3339),
	}

	if err := h.Rooms.SaveMessage(videoId, message); err != nil {
		slog.Error("Could not save a chat message", "error", err)
		h.sendTo(manager, connection.ConnectionId, errorFrame("Your message was not sent. Please try again."))
		return
	}

	h.broadcast(manager, videoId, map[string]interface{}{
		"action":  "message",
		"message": message,
	})
}

func (h *ChatHandler) deleteMessage(manager apigateway.ConnectionManager, connection *domain.ChatConnection, videoId, messageId string) {

	messageId = strings.TrimSpace(messageId)
	if messageId == "" {
		h.sendTo(manager, connection.ConnectionId, errorFrame("Which message should be deleted?"))
		return
	}

	deleted, err := h.Rooms.DeleteMessage(videoId, messageId, connection.UserId)
	if err != nil {
		slog.Error("Could not delete a chat message", "error", err)
		h.sendTo(manager, connection.ConnectionId, errorFrame("Could not delete that message. Please try again."))
		return
	}

	if !deleted {
		h.sendTo(manager, connection.ConnectionId, errorFrame("You can only delete your own messages."))
		return
	}

	h.broadcast(manager, videoId, map[string]interface{}{
		"action":     "deleted",
		"message_id": messageId,
	})
}

func (h *ChatHandler) reportMessage(manager apigateway.ConnectionManager, connectionId string, connection *domain.ChatConnection, videoId, messageId string) {

	messageId = strings.TrimSpace(messageId)
	if messageId == "" {
		h.sendTo(manager, connectionId, errorFrame("Which message should be reported?"))
		return
	}

	if err := h.Rooms.ReportMessage(videoId, messageId, connection.UserId); err != nil {
		slog.Error("Could not report a chat message", "error", err)
		h.sendTo(manager, connectionId, errorFrame("Could not report that message. Please try again."))
		return
	}

	h.sendTo(manager, connectionId, map[string]interface{}{
		"action":     "reported",
		"message_id": messageId,
	})
}

// broadcast posts a payload to every connection in a room, dropping the
// connections that are no longer there.
func (h *ChatHandler) broadcast(manager apigateway.ConnectionManager, videoId string, payload interface{}) {

	encoded, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Could not encode a chat payload", "error", err)
		return
	}

	connections, err := h.Rooms.RoomConnections(videoId)
	if err != nil {
		slog.Error("Could not list the chat connections", "error", err)
		return
	}

	gone := []string{}

	for _, connection := range connections {
		err := manager.PostToConnection(connection.ConnectionId, encoded)

		if err == nil {
			continue
		}

		if errors.Is(err, apigateway.ErrConnectionGone) {
			gone = append(gone, connection.ConnectionId)
			continue
		}

		slog.Warn("Could not post to a chat connection", "error", err)
	}

	if len(gone) > 0 {
		if err := h.Rooms.PruneConnections(videoId, gone); err != nil {
			slog.Warn("Could not prune the closed chat connections", "error", err)
		}
	}
}

func (h *ChatHandler) sendTo(manager apigateway.ConnectionManager, connectionId string, payload interface{}) {

	encoded, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Could not encode a chat payload", "error", err)
		return
	}

	if err := manager.PostToConnection(connectionId, encoded); err != nil && !errors.Is(err, apigateway.ErrConnectionGone) {
		slog.Warn("Could not post to a chat connection", "error", err)
	}
}

func errorFrame(message string) map[string]string {
	return map[string]string{"action": "error", "message": message}
}

func websocketResponse(statusCode int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: statusCode}
}
