package repositories

import (
	"errors"

	"github.com/victoraldir/myvideohunterapi/domain"
)

// ErrFolderNotFound is returned when a folder id does not belong to the user
// making the request. Handlers turn it into a 404.
var ErrFolderNotFound = errors.New("folder not found")

// UserDataRepository stores what a signed in user owns: the folders of saved
// videos, the users they have blocked in chat, and their public profile.
type UserDataRepository interface {
	// EnsureProfile returns the user's profile, creating it with a generated
	// nickname the first time it is asked for, so no account is ever without
	// one and no separate sign up step can be missed.
	EnsureProfile(userId string) (*domain.Profile, error)

	// SetNickname replaces the nickname of an existing profile.
	SetNickname(userId, nickname string) error

	ListFolders(userId string) ([]domain.Folder, error)
	CreateFolder(userId, name string) (*domain.Folder, error)
	RenameFolder(userId, folderId, name string) error
	DeleteFolder(userId, folderId string) error
	SaveVideoToFolder(userId, folderId, videoId string) error
	DeleteVideoFromFolder(userId, folderId, videoId string) error

	ListBlockedUsers(userId string) ([]string, error)
	BlockUser(userId, blockedUserId string) error
	UnblockUser(userId, blockedUserId string) error

	// DeleteAll removes every row the user owns. It backs the "delete my
	// account" request, so it has to be exhaustive rather than convenient.
	DeleteAll(userId string) error
}

// ChatRepository stores the short lived chat state of a video page: the open
// websocket connections, the messages, and a per user rate limit.
type ChatRepository interface {
	SaveConnection(videoId string, connection domain.ChatConnection) error
	GetConnection(videoId, connectionId string) (*domain.ChatConnection, error)
	RoomConnections(videoId string) ([]domain.ChatConnection, error)
	PruneConnections(videoId string, connectionIds []string) error

	SaveMessage(videoId string, message *domain.ChatMessage) error
	RecentMessages(videoId string, limit int64) ([]domain.ChatMessage, error)
	DeleteMessage(videoId, messageId, userId string) (bool, error)
	ReportMessage(videoId, messageId, userId string) error

	// DeleteAll removes everything the user left behind in chat: their
	// messages, their connections and the messages they reported, wherever
	// those live. It backs the "delete my account" request.
	DeleteAll(userId string) error

	// AllowMessage reports whether the user is still under the message rate
	// limit, and counts the attempt when they are.
	AllowMessage(userId string, limit int64) (bool, error)
}
