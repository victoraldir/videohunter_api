package dynamodb

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/repositories"
)

// fakeDB is a tiny in-memory stand-in for DynamoDB that implements only what
// the two repositories below actually call. It is deliberately literal: the
// tests use it to prove which keys and which conditions the repositories send,
// which is the part that cannot be checked any other way without credentials.
type fakeDB struct {
	mu    sync.Mutex
	items map[string]map[string]*dynamodb.AttributeValue
}

func newFakeDB() *fakeDB {
	return &fakeDB{items: map[string]map[string]*dynamodb.AttributeValue{}}
}

func (f *fakeDB) key(item map[string]*dynamodb.AttributeValue) string {
	return stringValue(item["pk"]) + "|" + stringValue(item["sk"])
}

func (f *fakeDB) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := f.key(input.Item)

	if input.ConditionExpression != nil && strings.Contains(*input.ConditionExpression, "attribute_not_exists(sk)") {
		if _, exists := f.items[key]; exists {
			return nil, conditionFailed()
		}
	}

	f.items[key] = input.Item

	return &dynamodb.PutItemOutput{}, nil
}

func (f *fakeDB) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	item, exists := f.items[f.key(input.Key)]
	if !exists {
		return &dynamodb.GetItemOutput{}, nil
	}

	return &dynamodb.GetItemOutput{Item: item}, nil
}

func (f *fakeDB) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	partition := stringValue(input.ExpressionAttributeValues[":pk"])
	prefix := ""

	if value, ok := input.ExpressionAttributeValues[":prefix"]; ok {
		prefix = stringValue(value)
	}

	keys := []string{}

	for key := range f.items {
		parts := strings.SplitN(key, "|", 2)
		if parts[0] != partition {
			continue
		}
		if prefix != "" && !strings.HasPrefix(parts[1], prefix) {
			continue
		}
		keys = append(keys, key)
	}

	// The fake table is a map, so order is imposed here the way DynamoDB
	// imposes it: by sort key, ascending unless asked otherwise.
	sortStrings(keys)

	if input.ScanIndexForward != nil && !*input.ScanIndexForward {
		reverse(keys)
	}

	if input.Limit != nil && int64(len(keys)) > *input.Limit {
		keys = keys[:*input.Limit]
	}

	items := []map[string]*dynamodb.AttributeValue{}

	for _, key := range keys {
		items = append(items, f.items[key])
	}

	return &dynamodb.QueryOutput{Items: items}, nil
}

func (f *fakeDB) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := f.key(input.Key)
	item, exists := f.items[key]

	if input.ConditionExpression != nil {
		condition := *input.ConditionExpression

		if condition == "attribute_exists(sk)" && !exists {
			return nil, conditionFailed()
		}

		// The rate limit condition, spelled out by the repository.
		if strings.Contains(condition, "#count < :max") {
			current := 0
			if exists && item["count"] != nil {
				current = intValue(item["count"])
			}

			if current >= intValue(input.ExpressionAttributeValues[":max"]) {
				return nil, conditionFailed()
			}
		}
	}

	if !exists {
		item = map[string]*dynamodb.AttributeValue{
			"pk": input.Key["pk"],
			"sk": input.Key["sk"],
		}
		f.items[key] = item
	}

	if name, ok := input.ExpressionAttributeNames["#name"]; ok {
		item[*name] = input.ExpressionAttributeValues[":name"]
	}

	if strings.Contains(aws.StringValue(input.UpdateExpression), "ADD #count") {
		current := 0
		if item["count"] != nil {
			current = intValue(item["count"])
		}
		item["count"] = &dynamodb.AttributeValue{N: aws.String(fmt.Sprint(current + 1))}
	}

	if _, ok := input.ExpressionAttributeValues[":ttl"]; ok && item["expiresAt"] == nil {
		item["expiresAt"] = input.ExpressionAttributeValues[":ttl"]
	}

	return &dynamodb.UpdateItemOutput{}, nil
}

func (f *fakeDB) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := f.key(input.Key)
	item, exists := f.items[key]

	if input.ConditionExpression != nil && strings.Contains(*input.ConditionExpression, "#userId = :userId") {
		if !exists || stringValue(item["userId"]) != stringValue(input.ExpressionAttributeValues[":userId"]) {
			return nil, conditionFailed()
		}
	}

	delete(f.items, key)

	return &dynamodb.DeleteItemOutput{}, nil
}

func (f *fakeDB) BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, requests := range input.RequestItems {
		for _, request := range requests {
			if request.DeleteRequest == nil {
				continue
			}
			delete(f.items, f.key(request.DeleteRequest.Key))
		}
	}

	return &dynamodb.BatchWriteItemOutput{}, nil
}

func (f *fakeDB) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.items)
}

func conditionFailed() error {
	return awserr.New(dynamodb.ErrCodeConditionalCheckFailedException, "condition failed", nil)
}

// intValue reads a number attribute. The production helper only reads string
// attributes, this one is for the counters the rate limit stores.
func intValue(attribute *dynamodb.AttributeValue) int {
	if attribute == nil {
		return 0
	}

	value := 0
	_, _ = fmt.Sscanf(aws.StringValue(attribute.N), "%d", &value)

	return value
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func reverse(values []string) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}

func TestUserDataRepository_GroupsSavedVideosIntoTheirFolders(t *testing.T) {

	db := newFakeDB()
	library := NewUserDataRepository(db, "user_data")

	first, err := library.CreateFolder("user-1", "Music")
	require.NoError(t, err)

	second, err := library.CreateFolder("user-1", "Sport")
	require.NoError(t, err)

	// Someone else's library must never show up.
	_, err = library.CreateFolder("user-2", "Private")
	require.NoError(t, err)

	require.NoError(t, library.SaveVideoToFolder("user-1", first.Id, "video-a"))
	require.NoError(t, library.SaveVideoToFolder("user-1", first.Id, "video-b"))
	require.NoError(t, library.SaveVideoToFolder("user-1", second.Id, "video-c"))

	// Removing a video only affects the folder it was in.
	require.NoError(t, library.DeleteVideoFromFolder("user-1", first.Id, "video-b"))

	folders, err := library.ListFolders("user-1")
	require.NoError(t, err)

	require.Len(t, folders, 2)
	assert.Equal(t, "Music", folders[0].Name)
	assert.Equal(t, []string{"video-a"}, videoIds(folders[0]))
	assert.Equal(t, []string{"video-c"}, videoIds(folders[1]))

	// The other user's folder is untouched and invisible.
	other, err := library.ListFolders("user-2")
	require.NoError(t, err)
	require.Len(t, other, 1)
	assert.Equal(t, "Private", other[0].Name)
}

func TestUserDataRepository_RefusesToSaveIntoAMissingFolder(t *testing.T) {

	db := newFakeDB()
	library := NewUserDataRepository(db, "user_data")

	err := library.SaveVideoToFolder("user-1", "not-a-folder", "video-a")

	assert.ErrorIs(t, err, repositories.ErrFolderNotFound)
	assert.Equal(t, 0, db.count())
}

func TestUserDataRepository_RenamesAndDeletesFolders(t *testing.T) {

	db := newFakeDB()
	library := NewUserDataRepository(db, "user_data")

	folder, err := library.CreateFolder("user-1", "Old name")
	require.NoError(t, err)

	require.NoError(t, library.RenameFolder("user-1", folder.Id, "New name"))
	require.NoError(t, library.SaveVideoToFolder("user-1", folder.Id, "video-a"))

	assert.ErrorIs(t, library.RenameFolder("user-1", "gone", "x"), repositories.ErrFolderNotFound)

	// Deleting a folder takes its saved videos with it and leaves nothing
	// behind.
	require.NoError(t, library.DeleteFolder("user-1", folder.Id))
	assert.ErrorIs(t, library.DeleteFolder("user-1", folder.Id), repositories.ErrFolderNotFound)

	folders, err := library.ListFolders("user-1")
	require.NoError(t, err)
	assert.Empty(t, folders)
	assert.Equal(t, 0, db.count())

	renamed, err := library.CreateFolder("user-1", "Second")
	require.NoError(t, err)
	require.NoError(t, library.RenameFolder("user-1", renamed.Id, "Third"))
	require.NoError(t, library.SaveVideoToFolder("user-1", renamed.Id, "video-a"))

	folders, err = library.ListFolders("user-1")
	require.NoError(t, err)
	require.Len(t, folders, 1)
	assert.Equal(t, "Third", folders[0].Name)
	assert.Equal(t, []string{"video-a"}, videoIds(folders[0]))
}

func TestUserDataRepository_BlockList(t *testing.T) {

	db := newFakeDB()
	library := NewUserDataRepository(db, "user_data")

	require.NoError(t, library.BlockUser("user-1", "user-2"))
	require.NoError(t, library.BlockUser("user-1", "user-3"))
	require.NoError(t, library.BlockUser("user-2", "user-1"))

	blocked, err := library.ListBlockedUsers("user-1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"user-2", "user-3"}, blocked)

	require.NoError(t, library.UnblockUser("user-1", "user-2"))

	blocked, err = library.ListBlockedUsers("user-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"user-3"}, blocked)

	// Blocks do not leak into the library listing.
	folders, err := library.ListFolders("user-1")
	require.NoError(t, err)
	assert.Empty(t, folders)
}

func TestChatDataRepository_ConnectionsAreKeptPerRoom(t *testing.T) {

	db := newFakeDB()
	chat := NewChatDataRepository(db, "chat_data")

	require.NoError(t, chat.SaveConnection("video-1", connection("conn-1", "user-1", "Victor")))
	require.NoError(t, chat.SaveConnection("video-1", connection("conn-2", "user-2", "Ana")))
	require.NoError(t, chat.SaveConnection("video-2", connection("conn-3", "user-1", "Victor")))

	connection, err := chat.GetConnection("video-1", "conn-1")
	require.NoError(t, err)
	require.NotNil(t, connection)
	assert.Equal(t, "user-1", connection.UserId)
	assert.Equal(t, "Victor", connection.Author)

	// A connection id from another room is not found, which is what stops a
	// browser from writing into a room it never joined.
	wrongRoom, err := chat.GetConnection("video-2", "conn-1")
	require.NoError(t, err)
	assert.Nil(t, wrongRoom)

	room, err := chat.RoomConnections("video-1")
	require.NoError(t, err)
	assert.Len(t, room, 2)

	require.NoError(t, chat.PruneConnections("video-1", []string{"conn-2"}))

	room, err = chat.RoomConnections("video-1")
	require.NoError(t, err)
	require.Len(t, room, 1)
	assert.Equal(t, "conn-1", room[0].ConnectionId)
}

func TestChatDataRepository_MessagesComeBackInReadingOrder(t *testing.T) {

	db := newFakeDB()
	chat := NewChatDataRepository(db, "chat_data")

	// Written out of order on purpose: the sort key is what orders a room.
	require.NoError(t, chat.SaveMessage("video-1", message("0000000000002#b", "user-2", "Ana", "second")))
	require.NoError(t, chat.SaveMessage("video-1", message("0000000000001#a", "user-1", "Victor", "first")))
	require.NoError(t, chat.SaveMessage("video-1", message("0000000000003#c", "user-1", "Victor", "third")))
	require.NoError(t, chat.SaveMessage("video-2", message("0000000000001#z", "user-1", "Victor", "other room")))

	messages, err := chat.RecentMessages("video-1", 50)
	require.NoError(t, err)

	require.Len(t, messages, 3)
	assert.Equal(t, "first", messages[0].Text)
	assert.Equal(t, "second", messages[1].Text)
	assert.Equal(t, "third", messages[2].Text)

	// Only the newest fit when the room is limited: the limit keeps the most
	// recent messages, not the oldest ones.
	messages, err = chat.RecentMessages("video-1", 2)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, "second", messages[0].Text)
	assert.Equal(t, "third", messages[1].Text)
	assert.NotContains(t, messageTexts(messages), "first")
}

func TestChatDataRepository_OnlyTheAuthorCanDeleteAMessage(t *testing.T) {

	db := newFakeDB()
	chat := NewChatDataRepository(db, "chat_data")

	require.NoError(t, chat.SaveMessage("video-1", message("0000000000001#a", "user-1", "Victor", "mine")))

	deleted, err := chat.DeleteMessage("video-1", "0000000000001#a", "user-2")
	require.NoError(t, err)
	assert.False(t, deleted)

	deleted, err = chat.DeleteMessage("video-1", "0000000000001#a", "user-1")
	require.NoError(t, err)
	assert.True(t, deleted)

	messages, err := chat.RecentMessages("video-1", 50)
	require.NoError(t, err)
	assert.Empty(t, messages)

	// Deleting a message that never existed is not an error and not a success.
	deleted, err = chat.DeleteMessage("video-1", "0000000000009#z", "user-1")
	require.NoError(t, err)
	assert.False(t, deleted)
}

func TestChatDataRepository_RateLimitsPerUserAndPerMinute(t *testing.T) {

	db := newFakeDB()
	chat := NewChatDataRepository(db, "chat_data")

	for attempt := 0; attempt < 3; attempt++ {
		allowed, err := chat.AllowMessage("user-1", 3)
		require.NoError(t, err)
		assert.True(t, allowed, "attempt %d should be allowed", attempt)
	}

	allowed, err := chat.AllowMessage("user-1", 3)
	require.NoError(t, err)
	assert.False(t, allowed)

	// The budget is per user.
	allowed, err = chat.AllowMessage("user-2", 3)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestChatDataRepository_ReportStoresARowForReview(t *testing.T) {

	db := newFakeDB()
	chat := NewChatDataRepository(db, "chat_data")

	require.NoError(t, chat.ReportMessage("video-1", "0000000000001#a", "user-1"))
	require.NoError(t, chat.ReportMessage("video-1", "0000000000001#a", "user-2"))

	// Reports are stored, never mixed into the message list.
	messages, err := chat.RecentMessages("video-1", 50)
	require.NoError(t, err)
	assert.Empty(t, messages)
	assert.Equal(t, 2, db.count())
}

func connection(connectionId, userId, author string) domain.ChatConnection {
	return domain.ChatConnection{ConnectionId: connectionId, UserId: userId, Author: author}
}

func message(id, userId, author, text string) *domain.ChatMessage {
	return &domain.ChatMessage{
		Id:        id,
		UserId:    userId,
		Author:    author,
		Text:      text,
		CreatedAt: "2026-10-01T10:00:00Z",
	}
}

func videoIds(folder domain.Folder) []string {
	ids := []string{}
	for _, video := range folder.Videos {
		ids = append(ids, video.VideoId)
	}
	return ids
}

func messageTexts(messages []domain.ChatMessage) []string {
	texts := []string{}
	for _, message := range messages {
		texts = append(texts, message.Text)
	}
	return texts
}
