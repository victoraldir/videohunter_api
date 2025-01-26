package usecase

// import (
// 	"net/http"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/victoraldir/myvideohunterbsky/repository/dynamodb"
// 	dynamodb_mock "github.com/victoraldir/myvideohunterbsky/repository/dynamodb/mocks"
// 	"github.com/victoraldir/myvideohunterbsky/services/bsky"
// 	"go.uber.org/mock/gomock"
// )

// var dynamodDBClientMock *dynamodb_mock.MockDynamodDBClient

// func setup(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	dynamodDBClientMock = dynamodb_mock.NewMockDynamodDBClient(ctrl)
// }

// func TestFetchPost_Execute(t *testing.T) {

// 	setup(t)

// 	t.Run("should return error when getting last scan from dynamodb", func(t *testing.T) {
// 		// Arrange
// 		httpClient := http.Client{}
// 		userName := "username"
// 		password := "password"
// 		bskyService := bsky.NewBskyService(&httpClient, userName, password)

// 		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()

// 		dynamodbRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
// 		fetchPost := NewFetchPost(bskyService, dynamodbRepository)

// 		// Act
// 		err := fetchPost.Execute(FetchPostRequest{
// 			BotName: "@myvideohunter.com",
// 		})

// 		// Assert
// 		assert.NotNil(t, err)
// 	})

// 	t.Run("should run successfully", func(t *testing.T) {

// 		// Arrange
// 		httpClient := http.Client{}
// 		userName := "bla"
// 		password := "bla"
// 		bskyService := bsky.NewBskyService(&httpClient, userName, password)

// 		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()

// 		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
// 		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

// 		dynamodbRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
// 		fetchPost := NewFetchPost(bskyService, dynamodbRepository)

// 		// Act
// 		err := fetchPost.Execute(FetchPostRequest{
// 			BotName: "@myvideohunter.com",
// 		})

// 		// Assert
// 		assert.Nil(t, err)

// 	})
// }
