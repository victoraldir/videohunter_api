package dynamodb

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/repositories"
)

//go:generate mockgen -destination=../dynamodb/mocks/mockChatDataDBClient.go -package=dynamodb github.com/victoraldir/myvideohunterapi/adapters/dynamodb ChatDataDBClient
type ChatDataDBClient interface {
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
	UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
	BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error)
}

// The chat table holds everything that is short lived, keyed so that a room
// needs one query and a rate limit needs one update:
//
//	ROOM#<video id>   CONN#<connection id>          an open websocket
//	ROOM#<video id>   MSG#<time>#<id>               a message
//	ROOM#<video id>   REPORT#<message id>#<user id> a report for review
//	USER#<user id>    RATELIMIT#<window>            messages in this minute
//
// Every row carries expiresAt so DynamoDB removes it on its own: connections
// after two hours, messages after thirty days, reports after ninety.
const (
	roomPrefix    = "ROOM#"
	userPrefix    = "USER#"
	connectionKey = "CONN#"
	messageKey    = "MSG#"
	reportKey     = "REPORT#"
	rateLimitKey  = "RATELIMIT#"

	rateLimitWindow = time.Minute
	connectionTTL   = 2 * time.Hour
	messageTTL      = 30 * 24 * time.Hour
	reportTTL       = 90 * 24 * time.Hour

	// byUserIndex finds everything one user left in any room. It is what makes
	// "delete my account" possible: a room is keyed by video, so without it a
	// user's messages could not be found across rooms.
	byUserIndex = "byUser"
)

type chatDataRepository struct {
	client    ChatDataDBClient
	tableName string
}

func NewChatDataRepository(client ChatDataDBClient, tableName string) repositories.ChatRepository {
	return &chatDataRepository{client: client, tableName: tableName}
}

func roomPartition(videoId string) string {
	return roomPrefix + videoId
}

func ratePartition(userId string) string {
	return userPrefix + userId
}

func connectionSortKey(connectionId string) string {
	return connectionKey + connectionId
}

func messageSortKey(messageId string) string {
	return messageKey + messageId
}

func (d *chatDataRepository) SaveConnection(videoId string, connection domain.ChatConnection) error {

	_, err := d.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"pk":        {S: aws.String(roomPartition(videoId))},
			"sk":        {S: aws.String(connectionSortKey(connection.ConnectionId))},
			"userId":    {S: aws.String(connection.UserId)},
			"author":    {S: aws.String(connection.Author)},
			"expiresAt": {N: aws.String(fmt.Sprint(time.Now().Add(connectionTTL).Unix()))},
		},
	})

	return err
}

func (d *chatDataRepository) GetConnection(videoId, connectionId string) (*domain.ChatConnection, error) {

	output, err := d.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"pk": {S: aws.String(roomPartition(videoId))},
			"sk": {S: aws.String(connectionSortKey(connectionId))},
		},
	})
	if err != nil {
		return nil, err
	}

	if output == nil || output.Item == nil {
		return nil, nil
	}

	return &domain.ChatConnection{
		ConnectionId: connectionId,
		UserId:       stringValue(output.Item["userId"]),
		Author:       stringValue(output.Item["author"]),
	}, nil
}

func (d *chatDataRepository) RoomConnections(videoId string) ([]domain.ChatConnection, error) {

	output, err := d.client.Query(&dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :prefix)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk":     {S: aws.String(roomPartition(videoId))},
			":prefix": {S: aws.String(connectionKey)},
		},
	})
	if err != nil {
		return nil, err
	}

	connections := make([]domain.ChatConnection, 0, len(output.Items))

	for _, item := range output.Items {
		connections = append(connections, domain.ChatConnection{
			ConnectionId: strings.TrimPrefix(stringValue(item["sk"]), connectionKey),
			UserId:       stringValue(item["userId"]),
			Author:       stringValue(item["author"]),
		})
	}

	return connections, nil
}

func (d *chatDataRepository) PruneConnections(videoId string, connectionIds []string) error {

	const batchSize = 25

	for start := 0; start < len(connectionIds); start += batchSize {
		end := start + batchSize
		if end > len(connectionIds) {
			end = len(connectionIds)
		}

		requests := []*dynamodb.WriteRequest{}

		for _, connectionId := range connectionIds[start:end] {
			requests = append(requests, &dynamodb.WriteRequest{
				DeleteRequest: &dynamodb.DeleteRequest{
					Key: map[string]*dynamodb.AttributeValue{
						"pk": {S: aws.String(roomPartition(videoId))},
						"sk": {S: aws.String(connectionSortKey(connectionId))},
					},
				},
			})
		}

		if _, err := d.client.BatchWriteItem(&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{d.tableName: requests},
		}); err != nil {
			return err
		}
	}

	return nil
}

func (d *chatDataRepository) SaveMessage(videoId string, message *domain.ChatMessage) error {

	_, err := d.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"pk":        {S: aws.String(roomPartition(videoId))},
			"sk":        {S: aws.String(messageSortKey(message.Id))},
			"userId":    {S: aws.String(message.UserId)},
			"author":    {S: aws.String(message.Author)},
			"text":      {S: aws.String(message.Text)},
			"createdAt": {S: aws.String(message.CreatedAt)},
			"expiresAt": {N: aws.String(fmt.Sprint(time.Now().Add(messageTTL).Unix()))},
		},
	})

	return err
}

func (d *chatDataRepository) RecentMessages(videoId string, limit int64) ([]domain.ChatMessage, error) {

	output, err := d.client.Query(&dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :prefix)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk":     {S: aws.String(roomPartition(videoId))},
			":prefix": {S: aws.String(messageKey)},
		},
		// Newest first, then flipped back to reading order below.
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int64(limit),
	})
	if err != nil {
		return nil, err
	}

	messages := make([]domain.ChatMessage, 0, len(output.Items))

	for _, item := range output.Items {
		messages = append(messages, domain.ChatMessage{
			Id:        strings.TrimPrefix(stringValue(item["sk"]), messageKey),
			UserId:    stringValue(item["userId"]),
			Author:    stringValue(item["author"]),
			Text:      stringValue(item["text"]),
			CreatedAt: stringValue(item["createdAt"]),
		})
	}

	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].Id < messages[j].Id
	})

	return messages, nil
}

func (d *chatDataRepository) DeleteMessage(videoId, messageId, userId string) (bool, error) {

	_, err := d.client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"pk": {S: aws.String(roomPartition(videoId))},
			"sk": {S: aws.String(messageSortKey(messageId))},
		},
		// Only the author can delete their own message. The check is on the
		// user id, not on the display name, which two people can share.
		ConditionExpression: aws.String("#userId = :userId"),
		ExpressionAttributeNames: map[string]*string{
			"#userId": aws.String("userId"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":userId": {S: aws.String(userId)},
		},
	})
	if err != nil {
		if isConditionFailed(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (d *chatDataRepository) ReportMessage(videoId, messageId, userId string) error {

	_, err := d.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"pk":        {S: aws.String(roomPartition(videoId))},
			"sk":        {S: aws.String(reportKey + messageId + "#" + userId)},
			"messageId": {S: aws.String(messageId)},
			// Also carried as userId so the byUser index finds the report when
			// the reporter deletes their account.
			"userId":    {S: aws.String(userId)},
			"createdAt": {S: aws.String(time.Now().UTC().Format(time.RFC3339))},
			"expiresAt": {N: aws.String(fmt.Sprint(time.Now().Add(reportTTL).Unix()))},
		},
	})

	return err
}

// DeleteAll removes what the user wrote in chat, wherever it is: their
// messages, their connections and the messages they reported, found through
// the byUser index, plus the rate limit rows under their own partition.
func (d *chatDataRepository) DeleteAll(userId string) error {

	keys, err := d.byUser(userId)
	if err != nil {
		return err
	}

	rateLimits, err := d.rateLimitRows(userId)
	if err != nil {
		return err
	}

	return deleteKeys(d.client, d.tableName, append(keys, rateLimits...))
}

func (d *chatDataRepository) byUser(userId string) ([]tableKey, error) {

	input := &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		IndexName:              aws.String(byUserIndex),
		KeyConditionExpression: aws.String("userId = :userId"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":userId": {S: aws.String(userId)},
		},
	}

	return d.eachKey(input)
}

func (d *chatDataRepository) rateLimitRows(userId string) ([]tableKey, error) {

	input := &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("pk = :pk"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {S: aws.String(ratePartition(userId))},
		},
	}

	return d.eachKey(input)
}

// eachKey runs a query to the end and returns the primary keys of every row.
func (d *chatDataRepository) eachKey(input *dynamodb.QueryInput) ([]tableKey, error) {

	keys := []tableKey{}

	for {
		output, err := d.client.Query(input)
		if err != nil {
			return nil, err
		}

		for _, item := range output.Items {
			keys = append(keys, tableKey{pk: stringValue(item["pk"]), sk: stringValue(item["sk"])})
		}

		if output.LastEvaluatedKey == nil {
			return keys, nil
		}

		input.ExclusiveStartKey = output.LastEvaluatedKey
	}
}

// tableKey is the primary key of a row, which is all a delete needs.
type tableKey struct {
	pk string
	sk string
}

type batchWriteClient interface {
	BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error)
}

// deleteKeys removes rows 25 at a time, which is DynamoDB's batch write limit.
func deleteKeys(client batchWriteClient, tableName string, keys []tableKey) error {

	const batchSize = 25

	for start := 0; start < len(keys); start += batchSize {
		end := start + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		requests := []*dynamodb.WriteRequest{}

		for _, key := range keys[start:end] {
			requests = append(requests, &dynamodb.WriteRequest{
				DeleteRequest: &dynamodb.DeleteRequest{
					Key: map[string]*dynamodb.AttributeValue{
						"pk": {S: aws.String(key.pk)},
						"sk": {S: aws.String(key.sk)},
					},
				},
			})
		}

		if _, err := client.BatchWriteItem(&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{tableName: requests},
		}); err != nil {
			return err
		}
	}

	return nil
}

func (d *chatDataRepository) AllowMessage(userId string, limit int64) (bool, error) {

	window := time.Now().UTC().Truncate(rateLimitWindow).Unix()

	_, err := d.client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"pk": {S: aws.String(ratePartition(userId))},
			"sk": {S: aws.String(rateLimitKey + fmt.Sprint(window))},
		},
		// "count" is a reserved word, so it is always aliased.
		UpdateExpression: aws.String("SET expiresAt = if_not_exists(expiresAt, :ttl) ADD #count :one"),
		ExpressionAttributeNames: map[string]*string{
			"#count": aws.String("count"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":one": {N: aws.String("1")},
			":ttl": {N: aws.String(fmt.Sprint(time.Now().Add(2 * rateLimitWindow).Unix()))},
			":max": {N: aws.String(fmt.Sprint(limit))},
		},
		// The whole point of the condition is to fail once the user has spent
		// their budget for this minute.
		ConditionExpression: aws.String("attribute_not_exists(#count) OR #count < :max"),
	})
	if err != nil {
		if isConditionFailed(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
