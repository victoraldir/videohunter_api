package dynamodb

import (
	"log/slog"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/victoraldir/myvideohunterapi/domain"
)

type dynamodbSettingsRepository struct {
	client             DynamodDBClient
	setttingsTableName string
}

func NewDynamoSettingsRepository(client DynamodDBClient, setttingsTableName string) dynamodbSettingsRepository {
	return dynamodbSettingsRepository{client, setttingsTableName}
}

func (d dynamodbSettingsRepository) SaveSetting(setting *domain.Settings) (*domain.Settings, error) {

	output, err := d.client.PutItem(&dynamodb.PutItemInput{
		TableName: &d.setttingsTableName,
		Item: map[string]*dynamodb.AttributeValue{
			"key": {
				S: &setting.KeySetting,
			},
			"value": {
				S: &setting.Value,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	slog.Debug("DynamoDB SaveSetting output: ", "output", output)

	return setting, nil
}

func (d dynamodbSettingsRepository) GetSetting(key domain.KeySetting) (*domain.Settings, error) {

	slog.Debug("DynamoDB GetSetting key: ", "key", key, "TableName", d.setttingsTableName)

	keyStr := string(key)
	keyPtr := &keyStr

	output, err := d.client.GetItem(&dynamodb.GetItemInput{
		TableName: &d.setttingsTableName,
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: keyPtr,
			},
		},
	})

	if err != nil {
		slog.Error("DynamoDB GetSetting error: ", "error", err)
		return nil, err
	}

	if output == nil {
		slog.Debug("DynamoDB GetSetting output is nil")
		return nil, nil
	}

	if output.Item == nil {
		slog.Debug("DynamoDB GetSetting output.Item is nil")
		return nil, nil
	}

	setting := domain.Settings{
		KeySetting: keyStr,
		Value:      *output.Item["value"].S,
	}

	return &setting, nil
}
