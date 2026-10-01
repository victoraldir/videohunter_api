package handlers

import (
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/victoraldir/myvideohunterapi/adapters/apigateway"
	"github.com/victoraldir/myvideohunterapi/auth"
	"github.com/victoraldir/myvideohunterapi/domain"
)

type fakeConnections struct {
	mu    sync.Mutex
	posts map[string][][]byte
	gone  map[string]bool
}

func newFakeConnections() *fakeConnections {
	return &fakeConnections{posts: map[string][][]byte{}, gone: map[string]bool{}}
}

func (f *fakeConnections) PostToConnection(connectionId string, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.gone[connectionId] {
		return apigateway.ErrConnectionGone
	}

	copied := make([]byte, len(payload))
	copy(copied, payload)
	f.posts[connectionId] = append(f.posts[connectionId], copied)

	return nil
}

// frames returns the payloads delivered to one connection, oldest first.
func (f *fakeConnections) frames(t *testing.T, connectionId string) []map[string]interface{} {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()

	frames := []map[string]interface{}{}

	for _, payload := range f.posts[connectionId] {
		frame := map[string]interface{}{}
		require.NoError(t, json.Unmarshal(payload, &frame))
		frames = append(frames, frame)
	}

	return frames
}

func (f *fakeConnections) recipients() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	names := make([]string, 0, len(f.posts))
	for connectionId := range f.posts {
		names = append(names, connectionId)
	}
	sort.Strings(names)

	return names
}

type stubChat struct {
	connection   *domain.ChatConnection
	getErr       error
	room         []domain.ChatConnection
	roomErr      error
	pruned       []string
	saved        []*domain.ChatMessage
	saveErr      error
	recent       []domain.ChatMessage
	deleted      bool
	deleteErr    error
	reported     []string
	reportErr    error
	allow        bool
	allowErr     error
	rateChecked  []string
	savedConn    *domain.ChatConnection
	savedVideoId string
	saveConnErr  error
	deletedAll   []string
	deleteAllErr error
}

func (s *stubChat) SaveConnection(videoId string, connection domain.ChatConnection) error {
	s.savedVideoId = videoId
	s.savedConn = &connection
	return s.saveConnErr
}

func (s *stubChat) GetConnection(string, string) (*domain.ChatConnection, error) {
	return s.connection, s.getErr
}

func (s *stubChat) RoomConnections(string) ([]domain.ChatConnection, error) { return s.room, s.roomErr }

func (s *stubChat) PruneConnections(_ string, connectionIds []string) error {
	s.pruned = append(s.pruned, connectionIds...)
	return nil
}

func (s *stubChat) SaveMessage(_ string, message *domain.ChatMessage) error {
	s.saved = append(s.saved, message)
	return s.saveErr
}

func (s *stubChat) RecentMessages(string, int64) ([]domain.ChatMessage, error) { return s.recent, nil }

func (s *stubChat) DeleteMessage(string, string, string) (bool, error) { return s.deleted, s.deleteErr }

func (s *stubChat) ReportMessage(_ string, messageId, userId string) error {
	s.reported = append(s.reported, messageId+"/"+userId)
	return s.reportErr
}

func (s *stubChat) AllowMessage(userId string, _ int64) (bool, error) {
	s.rateChecked = append(s.rateChecked, userId)
	return s.allow, s.allowErr
}

func (s *stubChat) DeleteAll(userId string) error {
	s.deletedAll = append(s.deletedAll, userId)
	return s.deleteAllErr
}

func websocketRequest(routeKey, body string, query map[string]string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		Body:                  body,
		QueryStringParameters: query,
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			RouteKey:     routeKey,
			ConnectionID: "conn-1",
			DomainName:   "api.example.com",
			Stage:        "prod",
		},
	}
}

func chatHandler(verifier TokenVerifier, chat *stubChat, connections *fakeConnections) *ChatHandler {
	return chatHandlerWithProfile(verifier, chat, &stubUserData{}, connections)
}

func chatHandlerWithProfile(verifier TokenVerifier, chat *stubChat, profiles *stubUserData, connections *fakeConnections) *ChatHandler {
	return NewChatHandler(verifier, chat, profiles, func(string, string) apigateway.ConnectionManager {
		return connections
	})
}

func TestChatHandler_ConnectRequiresAVideoAndAcceptsGuests(t *testing.T) {

	chat := &stubChat{}
	connections := newFakeConnections()

	// No room.
	verifier := &stubVerifier{claims: &auth.Claims{Sub: "user-1", Name: "Victor"}}
	response, err := chatHandler(verifier, chat, connections).Handle(
		websocketRequest("$connect", "", map[string]string{"Authorization": "a-token"}))
	require.NoError(t, err)
	assert.Equal(t, 400, response.StatusCode)
	assert.Nil(t, chat.savedConn)

	// Room but no token at all: a guest, allowed to watch.
	response, err = chatHandler(verifier, chat, connections).Handle(
		websocketRequest("$connect", "", map[string]string{"videoId": "video-1"}))
	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	require.NotNil(t, chat.savedConn)
	assert.Equal(t, "video-1", chat.savedVideoId)
	assert.Equal(t, "", chat.savedConn.UserId)
	assert.Equal(t, "", chat.savedConn.Author)
	assert.False(t, chat.savedConn.SignedIn())

	// A token that is present but unusable is still an error, not a guest.
	rejecting := &stubVerifier{err: errors.New("expired")}
	response, err = chatHandler(rejecting, chat, connections).Handle(
		websocketRequest("$connect", "", map[string]string{"videoId": "video-1", "Authorization": "bad"}))
	require.NoError(t, err)
	assert.Equal(t, 401, response.StatusCode)

	// Both, and the room name comes from the stored profile rather than from
	// anything the token carries.
	response, err = chatHandler(verifier, chat, connections).Handle(
		websocketRequest("$connect", "", map[string]string{"videoId": "video-1", "Authorization": "a-token"}))
	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	require.NotNil(t, chat.savedConn)
	assert.Equal(t, "user-1", chat.savedConn.UserId)
	assert.Equal(t, "Tester", chat.savedConn.Author)
	assert.Equal(t, "conn-1", chat.savedConn.ConnectionId)
}

func TestChatHandler_LetsAGuestReadButNotWrite(t *testing.T) {

	connections := newFakeConnections()

	// A guest connection, the shape the connect handler stores for an
	// unauthenticated socket.
	guest := &stubChat{
		connection: &domain.ChatConnection{ConnectionId: "conn-1"},
		recent: []domain.ChatMessage{
			{Id: "message-1", UserId: "user-2", Author: "Ana", Text: "hello", CreatedAt: "2026-01-01T00:00:00Z"},
		},
	}

	// Reading works.
	response, err := chatHandler(&stubVerifier{}, guest, connections).Handle(
		websocketRequest("$default", `{"action":"recent","video_id":"video-1"}`, nil))
	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	frames := connections.frames(t, "conn-1")
	require.Len(t, frames, 1)
	assert.Equal(t, "recent", frames[0]["action"])
	assert.Equal(t, "", frames[0]["user_id"])

	// Writing does not, and nothing is stored.
	response, err = chatHandler(&stubVerifier{}, guest, connections).Handle(
		websocketRequest("$default", `{"action":"send","video_id":"video-1","text":"spam"}`, nil))
	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Empty(t, guest.saved)
	assert.Empty(t, guest.reported)
	assert.Empty(t, guest.rateChecked)

	frames = connections.frames(t, "conn-1")
	require.Len(t, frames, 2)
	assert.Equal(t, "error", frames[1]["action"])
	assert.Equal(t, guestNotice, frames[1]["message"])

	// Deleting and reporting are refused the same way.
	for _, action := range []string{"delete", "report"} {
		response, err = chatHandler(&stubVerifier{}, guest, connections).Handle(
			websocketRequest("$default", `{"action":"`+action+`","video_id":"video-1","message_id":"message-1"}`, nil))
		require.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
	}

	assert.False(t, guest.deleted)
	assert.Empty(t, guest.reported)
}

func TestChatHandler_IgnoresMessagesFromUnknownConnections(t *testing.T) {

	chat := &stubChat{connection: nil}
	connections := newFakeConnections()

	response, err := chatHandler(signedIn(), chat, connections).Handle(
		websocketRequest("$default", `{"action":"send","video_id":"video-1","text":"hi"}`, nil))

	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Empty(t, chat.saved)
	assert.Empty(t, connections.recipients())
}

func TestChatHandler_BroadcastsAMessageToTheRoom(t *testing.T) {

	chat := &stubChat{
		connection: &domain.ChatConnection{ConnectionId: "conn-1", UserId: "user-1", Author: "Victor"},
		room: []domain.ChatConnection{
			{ConnectionId: "conn-1", UserId: "user-1", Author: "Victor"},
			{ConnectionId: "conn-2", UserId: "user-2", Author: "Ana"},
		},
		allow: true,
	}
	connections := newFakeConnections()

	response, err := chatHandler(signedIn(), chat, connections).Handle(
		websocketRequest("$default", `{"action":"send","video_id":"video-1","text":"  hello  "}`, nil))

	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	require.Len(t, chat.saved, 1)
	// The author is taken from the stored connection, not from the payload.
	assert.Equal(t, "user-1", chat.saved[0].UserId)
	assert.Equal(t, "Victor", chat.saved[0].Author)
	assert.Equal(t, "hello", chat.saved[0].Text)
	assert.NotEmpty(t, chat.saved[0].Id)

	// Everyone in the room, including the sender, gets the message.
	assert.Equal(t, []string{"conn-1", "conn-2"}, connections.recipients())

	for _, connectionId := range []string{"conn-1", "conn-2"} {
		frames := connections.frames(t, connectionId)
		require.Len(t, frames, 1)
		assert.Equal(t, "message", frames[0]["action"])
	}
}

func TestChatHandler_RejectsLongMessagesAndRateLimitedUsers(t *testing.T) {

	long := make([]rune, maxChatMessageLength+1)
	for index := range long {
		long[index] = 'a'
	}

	for name, chat := range map[string]*stubChat{
		"too long": {
			connection: &domain.ChatConnection{ConnectionId: "conn-1", UserId: "user-1", Author: "Victor"},
			allow:      true,
		},
		"empty": {
			connection: &domain.ChatConnection{ConnectionId: "conn-1", UserId: "user-1", Author: "Victor"},
			allow:      true,
		},
		"rate limited": {
			connection: &domain.ChatConnection{ConnectionId: "conn-1", UserId: "user-1", Author: "Victor"},
			allow:      false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			connections := newFakeConnections()
			text := "   "

			if name == "too long" {
				text = string(long)
			}

			body, err := json.Marshal(map[string]string{"action": "send", "video_id": "video-1", "text": text})
			require.NoError(t, err)

			response, err := chatHandler(signedIn(), chat, connections).Handle(
				websocketRequest("$default", string(body), nil))
			require.NoError(t, err)
			assert.Equal(t, 200, response.StatusCode)
			assert.Empty(t, chat.saved)

			frames := connections.frames(t, "conn-1")
			require.Len(t, frames, 1)
			assert.Equal(t, "error", frames[0]["action"])
		})
	}
}

func TestChatHandler_DeletesOnlyOwnMessages(t *testing.T) {

	own := &stubChat{
		connection: &domain.ChatConnection{ConnectionId: "conn-1", UserId: "user-1", Author: "Victor"},
		room:       []domain.ChatConnection{{ConnectionId: "conn-1"}, {ConnectionId: "conn-2"}},
		deleted:    true,
	}
	connections := newFakeConnections()

	response, err := chatHandler(signedIn(), own, connections).Handle(
		websocketRequest("$default", `{"action":"delete","video_id":"video-1","message_id":"123#abc"}`, nil))
	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	frames := connections.frames(t, "conn-2")
	require.Len(t, frames, 1)
	assert.Equal(t, "deleted", frames[0]["action"])
	assert.Equal(t, "123#abc", frames[0]["message_id"])

	// The database refuses when the message is someone else's.
	foreign := &stubChat{
		connection: &domain.ChatConnection{ConnectionId: "conn-1", UserId: "user-1", Author: "Victor"},
		deleted:    false,
	}
	connections = newFakeConnections()

	response, err = chatHandler(signedIn(), foreign, connections).Handle(
		websocketRequest("$default", `{"action":"delete","video_id":"video-1","message_id":"123#abc"}`, nil))
	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "error", connections.frames(t, "conn-1")[0]["action"])
}

func TestChatHandler_SendsHistoryAndReportsToTheRequesterOnly(t *testing.T) {

	chat := &stubChat{
		connection: &domain.ChatConnection{ConnectionId: "conn-1", UserId: "user-1", Author: "Victor"},
		room:       []domain.ChatConnection{{ConnectionId: "conn-1"}, {ConnectionId: "conn-2"}},
		recent:     []domain.ChatMessage{{Id: "1#a", Author: "Ana", Text: "hola"}},
	}
	connections := newFakeConnections()

	_, err := chatHandler(signedIn(), chat, connections).Handle(
		websocketRequest("$default", `{"action":"recent","video_id":"video-1"}`, nil))
	require.NoError(t, err)

	frames := connections.frames(t, "conn-1")
	require.Len(t, frames, 1)
	assert.Equal(t, "recent", frames[0]["action"])
	assert.Equal(t, "user-1", frames[0]["user_id"])
	// Nobody else sees the private reply.
	assert.Empty(t, connections.frames(t, "conn-2"))

	connections = newFakeConnections()

	_, err = chatHandler(signedIn(), chat, connections).Handle(
		websocketRequest("$default", `{"action":"report","video_id":"video-1","message_id":"1#a"}`, nil))
	require.NoError(t, err)
	assert.Equal(t, []string{"1#a/user-1"}, chat.reported)
	assert.Equal(t, "reported", connections.frames(t, "conn-1")[0]["action"])

	// An unknown action is answered, not ignored, so the browser can show it.
	connections = newFakeConnections()

	_, err = chatHandler(signedIn(), chat, connections).Handle(
		websocketRequest("$default", `{"action":"dance","video_id":"video-1"}`, nil))
	require.NoError(t, err)
	frames = connections.frames(t, "conn-1")
	require.Len(t, frames, 1)
	assert.Equal(t, "error", frames[0]["action"])
}

func TestChatHandler_DropsConnectionsThatAreGone(t *testing.T) {

	chat := &stubChat{
		connection: &domain.ChatConnection{ConnectionId: "conn-1", UserId: "user-1", Author: "Victor"},
		room: []domain.ChatConnection{
			{ConnectionId: "conn-1"},
			{ConnectionId: "conn-closed"},
			{ConnectionId: "conn-2"},
		},
		allow: true,
	}
	connections := newFakeConnections()
	connections.gone["conn-closed"] = true

	_, err := chatHandler(signedIn(), chat, connections).Handle(
		websocketRequest("$default", `{"action":"send","video_id":"video-1","text":"hi"}`, nil))
	require.NoError(t, err)

	assert.Equal(t, []string{"conn-1", "conn-2"}, connections.recipients())
	assert.Equal(t, []string{"conn-closed"}, chat.pruned)
}

func TestChatHandler_DisconnectIsANoOp(t *testing.T) {

	response, err := chatHandler(signedIn(), &stubChat{}, newFakeConnections()).Handle(
		websocketRequest("$disconnect", "", nil))

	require.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
}
