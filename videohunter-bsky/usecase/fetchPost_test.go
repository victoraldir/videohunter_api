package usecase

import (
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	dynamodb_aws "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/victoraldir/myvideohunterbsky/repository/dynamodb"
	dynamodb_mock "github.com/victoraldir/myvideohunterbsky/repository/dynamodb/mocks"
	"github.com/victoraldir/myvideohuntershared/domain"
	"github.com/victoraldir/myvideohuntershared/services/bsky"
	"go.uber.org/mock/gomock"
)

var dynamodDBClientMock *dynamodb_mock.MockDynamodDBClient

func setup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dynamodDBClientMock = dynamodb_mock.NewMockDynamodDBClient(ctrl)
}

func TestFetchPost_Execute(t *testing.T) {

	setup(t)

	t.Run("should return error when getting last scan from dynamodb", func(t *testing.T) {
		// Arrange
		httpClient := http.Client{}
		userName := "username"
		password := "password"
		bskyService := bsky.NewBskyService(&httpClient, userName, password)

		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()

		dynamodbRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		fetchPost := NewFetchPost(bskyService, dynamodbRepository)

		// Act
		err := fetchPost.Execute(FetchPostRequest{
			BotName: "@myvideohunter.com",
		})

		// Assert
		assert.NotNil(t, err)
	})

	t.Run("should run successfully", func(t *testing.T) {

		// Arrange
		httpClient := http.Client{}
		userName := "myvideohunter.com"
		password := "s3cr3t"
		bskyService := bsky.NewBskyService(&httpClient, userName, password)

		tableStr := "settings"
		bskyAccessTokenKey := domain.BskyAccessToken
		bskyLastScanKey := domain.BskyLastExecutionTime
		lastScanTimeStr := "2025-01-26T16:07:14Z"
		lastScanTimeOutput := &dynamodb_aws.GetItemOutput{
			Item: map[string]*dynamodb_aws.AttributeValue{
				"key": {
					S: aws.String(string(bskyAccessTokenKey)),
				},
				"value": {
					S: &lastScanTimeStr,
				},
			},
		}

		lastScanTimeInput := &dynamodb_aws.GetItemInput{
			TableName: &tableStr,
			Key: map[string]*dynamodb_aws.AttributeValue{
				"key": {
					S: aws.String(string(bskyLastScanKey)),
				},
			},
		}

		dynamodDBClientMock.EXPECT().GetItem(lastScanTimeInput).Return(lastScanTimeOutput, nil)
		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil)
		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil)
		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

		dynamodbRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		fetchPost := NewFetchPost(bskyService, dynamodbRepository)

		// Act
		err := fetchPost.Execute(FetchPostRequest{
			BotName: "@myvideohunter.com",
		})

		// Assert
		assert.Nil(t, err)

	})
}
