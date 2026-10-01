package domain

// ChatMessage is one message in the room attached to a video page.
//
// Author is the display name captured when the message was written, so a room
// can show who said what without another lookup. UserId is the Cognito subject:
// it lets an author delete their own message and lets a reader hide the
// messages of someone they blocked, and it is never rendered.
type ChatMessage struct {
	Id        string `json:"id"`
	UserId    string `json:"user_id"`
	Author    string `json:"author"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

// ChatConnection is one browser attached to a room's websocket.
//
// A connection with no UserId is a guest: anyone may watch a room, so the
// socket is allowed to exist without an account, but only a connection that
// carries a user id can write, delete or report.
type ChatConnection struct {
	ConnectionId string
	UserId       string
	Author       string
}

// SignedIn reports whether the connection belongs to an account.
func (c ChatConnection) SignedIn() bool {
	return c.UserId != ""
}
