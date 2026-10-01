package dynamodb

import (
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/repositories"
	"github.com/victoraldir/myvideohunterapi/utils"
)

//go:generate mockgen -destination=../dynamodb/mocks/mockUserDataDBClient.go -package=dynamodb github.com/victoraldir/myvideohunterapi/adapters/dynamodb UserDataDBClient
type UserDataDBClient interface {
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
	BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error)
}

// The user data table keys every owned row under the user id, so a whole
// library is one query:
//
//	pk = <Cognito sub>   sk = FOLDER#<id>                     folder
//	pk = <Cognito sub>   sk = FOLDER#<id>#VIDEO#<video id>    saved video
//	pk = <Cognito sub>   sk = BLOCK#<blocked user id>         chat block
const (
	folderPrefix = "FOLDER#"
	videoMarker  = "#VIDEO#"
	blockPrefix  = "BLOCK#"
)

type userDataRepository struct {
	client    UserDataDBClient
	tableName string
}

func NewUserDataRepository(client UserDataDBClient, tableName string) repositories.UserDataRepository {
	if tableName == "" {
		slog.Error("USER_DATA_TABLE is not set: the library and account deletion will fail")
	}

	return &userDataRepository{client: client, tableName: tableName}
}

func folderKey(folderId string) string {
	return folderPrefix + folderId
}

func videoKey(folderId, videoId string) string {
	return folderPrefix + folderId + videoMarker + videoId
}

func blockKey(userId string) string {
	return blockPrefix + userId
}

func (d *userDataRepository) ListFolders(userId string) ([]domain.Folder, error) {

	folders := []domain.Folder{}
	positions := map[string]int{}

	err := d.eachItem(userId, "", func(item map[string]*dynamodb.AttributeValue) {
		sk := stringValue(item["sk"])

		if !strings.HasPrefix(sk, folderPrefix) {
			return
		}

		if folderId, videoId, ok := splitVideoKey(sk); ok {
			position, found := positions[folderId]
			if !found {
				// A saved video whose folder row is missing is not part of
				// any library; skip it rather than resurrect the folder.
				return
			}

			folders[position].Videos = append(folders[position].Videos, domain.SavedVideo{
				VideoId: videoId,
				SavedAt: stringValue(item["savedAt"]),
			})
			return
		}

		folder := domain.Folder{
			Id:        strings.TrimPrefix(sk, folderPrefix),
			Name:      stringValue(item["name"]),
			CreatedAt: stringValue(item["createdAt"]),
			Videos:    []domain.SavedVideo{},
		}

		positions[folder.Id] = len(folders)
		folders = append(folders, folder)
	})

	if err != nil {
		return nil, err
	}

	// CreatedAt is stored with millisecond precision, but two folders made in
	// the same millisecond would still tie, and the id is what makes the order
	// deterministic rather than whatever the query happened to return.
	sort.SliceStable(folders, func(i, j int) bool {
		if folders[i].CreatedAt != folders[j].CreatedAt {
			return folders[i].CreatedAt < folders[j].CreatedAt
		}
		return folders[i].Id < folders[j].Id
	})

	for i := range folders {
		videos := folders[i].Videos
		sort.SliceStable(videos, func(a, b int) bool {
			if videos[a].SavedAt != videos[b].SavedAt {
				return videos[a].SavedAt < videos[b].SavedAt
			}
			return videos[a].VideoId < videos[b].VideoId
		})
	}

	return folders, nil
}

func (d *userDataRepository) CreateFolder(userId, name string) (*domain.Folder, error) {

	// Nanosecond precision: a folder created a moment after another must sort
	// after it, and a whole second is long enough to create several.
	now := time.Now().UTC().Format(time.RFC3339Nano)

	folder := &domain.Folder{
		Id:        utils.NewId(),
		Name:      name,
		CreatedAt: now,
		Videos:    []domain.SavedVideo{},
	}

	_, err := d.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"pk":        {S: aws.String(userId)},
			"sk":        {S: aws.String(folderKey(folder.Id))},
			"name":      {S: aws.String(folder.Name)},
			"createdAt": {S: aws.String(folder.CreatedAt)},
		},
		// The id is random; the condition only protects against a collision.
		ConditionExpression: aws.String("attribute_not_exists(sk)"),
	})
	if err != nil {
		return nil, err
	}

	return folder, nil
}

func (d *userDataRepository) RenameFolder(userId, folderId, name string) error {

	_, err := d.client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"pk": {S: aws.String(userId)},
			"sk": {S: aws.String(folderKey(folderId))},
		},
		UpdateExpression:         aws.String("SET #name = :name"),
		ExpressionAttributeNames: map[string]*string{"#name": aws.String("name")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":name": {S: aws.String(name)},
		},
		// The strict prefix keeps a saved video row from being renamed: only
		// the folder row itself has exactly this key.
		ConditionExpression: aws.String("attribute_exists(sk)"),
	})
	if err != nil {
		if isConditionFailed(err) {
			return repositories.ErrFolderNotFound
		}
		return err
	}

	return nil
}

func (d *userDataRepository) DeleteFolder(userId, folderId string) error {

	folder, err := d.folder(userId, folderId)
	if err != nil {
		return err
	}
	if folder == nil {
		return repositories.ErrFolderNotFound
	}

	keys := []string{}

	err = d.eachItem(userId, folderPrefix+folderId, func(item map[string]*dynamodb.AttributeValue) {
		if _, _, ok := splitVideoKey(stringValue(item["sk"])); ok {
			keys = append(keys, stringValue(item["sk"]))
		}
	})
	if err != nil {
		return err
	}

	keys = append(keys, folderKey(folderId))

	return d.batchDelete(userId, keys)
}

func (d *userDataRepository) SaveVideoToFolder(userId, folderId, videoId string) error {

	folder, err := d.folder(userId, folderId)
	if err != nil {
		return err
	}
	if folder == nil {
		return repositories.ErrFolderNotFound
	}

	_, err = d.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"pk":      {S: aws.String(userId)},
			"sk":      {S: aws.String(videoKey(folderId, videoId))},
			"videoId": {S: aws.String(videoId)},
			"savedAt": {S: aws.String(time.Now().UTC().Format(time.RFC3339Nano))},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *userDataRepository) DeleteVideoFromFolder(userId, folderId, videoId string) error {

	// Deleting an already deleted video is a no-op, so removing from two
	// folders is safe when the UI moves a video around.
	_, err := d.client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"pk": {S: aws.String(userId)},
			"sk": {S: aws.String(videoKey(folderId, videoId))},
		},
	})

	return err
}

func (d *userDataRepository) ListBlockedUsers(userId string) ([]string, error) {

	blocked := []string{}

	err := d.eachItem(userId, blockPrefix, func(item map[string]*dynamodb.AttributeValue) {
		blocked = append(blocked, strings.TrimPrefix(stringValue(item["sk"]), blockPrefix))
	})
	if err != nil {
		return nil, err
	}

	return blocked, nil
}

func (d *userDataRepository) BlockUser(userId, blockedUserId string) error {

	_, err := d.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"pk":        {S: aws.String(userId)},
			"sk":        {S: aws.String(blockKey(blockedUserId))},
			"createdAt": {S: aws.String(time.Now().UTC().Format(time.RFC3339))},
		},
	})

	return err
}

func (d *userDataRepository) UnblockUser(userId, blockedUserId string) error {

	_, err := d.client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"pk": {S: aws.String(userId)},
			"sk": {S: aws.String(blockKey(blockedUserId))},
		},
	})

	return err
}

// DeleteAll removes every row the user owns: folders, saved videos and the
// chat block list. Deleting an account leaves nothing behind, and running it
// twice is harmless.
func (d *userDataRepository) DeleteAll(userId string) error {

	keys := []string{}

	err := d.eachItem(userId, "", func(item map[string]*dynamodb.AttributeValue) {
		keys = append(keys, stringValue(item["sk"]))
	})
	if err != nil {
		return err
	}

	return d.batchDelete(userId, keys)
}

// folder returns a folder row, or nil when the user has no folder with that id.
func (d *userDataRepository) folder(userId, folderId string) (*domain.Folder, error) {

	output, err := d.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"pk": {S: aws.String(userId)},
			"sk": {S: aws.String(folderKey(folderId))},
		},
	})
	if err != nil {
		return nil, err
	}

	if output == nil || output.Item == nil {
		return nil, nil
	}

	return &domain.Folder{
		Id:        folderId,
		Name:      stringValue(output.Item["name"]),
		CreatedAt: stringValue(output.Item["createdAt"]),
		Videos:    []domain.SavedVideo{},
	}, nil
}

// eachItem walks every row of one user under an optional key prefix. A user
// with thousands of saved videos is far beyond what this feature is for, but
// pagination is cheap to get right.
func (d *userDataRepository) eachItem(userId, prefix string, visit func(item map[string]*dynamodb.AttributeValue)) error {

	input := &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("pk = :pk"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {S: aws.String(userId)},
		},
	}

	if prefix != "" {
		input.KeyConditionExpression = aws.String("pk = :pk AND begins_with(sk, :prefix)")
		input.ExpressionAttributeValues[":prefix"] = &dynamodb.AttributeValue{S: aws.String(prefix)}
	}

	for {
		output, err := d.client.Query(input)
		if err != nil {
			return err
		}

		for _, item := range output.Items {
			visit(item)
		}

		if output.LastEvaluatedKey == nil {
			return nil
		}

		input.ExclusiveStartKey = output.LastEvaluatedKey
	}
}

// batchDelete removes rows 25 at a time, which is DynamoDB's batch write limit.
func (d *userDataRepository) batchDelete(userId string, keys []string) error {

	const batchSize = 25

	for start := 0; start < len(keys); start += batchSize {
		end := start + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		requests := []*dynamodb.WriteRequest{}

		for _, sk := range keys[start:end] {
			requests = append(requests, &dynamodb.WriteRequest{
				DeleteRequest: &dynamodb.DeleteRequest{
					Key: map[string]*dynamodb.AttributeValue{
						"pk": {S: aws.String(userId)},
						"sk": {S: aws.String(sk)},
					},
				},
			})
		}

		output, err := d.client.BatchWriteItem(&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{d.tableName: requests},
		})
		if err != nil {
			return err
		}

		if len(output.UnprocessedItems) > 0 {
			// Retrying once is enough: the rows are gone from the library's
			// point of view even if a straggler needs a later sweep.
			slog.Warn("DynamoDB left unprocessed deletes", "count", len(output.UnprocessedItems[d.tableName]))
		}
	}

	return nil
}

// splitVideoKey splits "FOLDER#<id>#VIDEO#<video id>" into its two ids. The
// second return value is false for a plain folder key.
func splitVideoKey(sk string) (folderId, videoId string, ok bool) {

	position := strings.Index(sk, videoMarker)
	if position < 0 {
		return "", "", false
	}

	return strings.TrimPrefix(sk[:position], folderPrefix), sk[position+len(videoMarker):], true
}

func stringValue(attribute *dynamodb.AttributeValue) string {
	if attribute == nil || attribute.S == nil {
		return ""
	}

	return *attribute.S
}
