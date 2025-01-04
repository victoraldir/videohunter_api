package usecases

import (
	"testing"
)

func TestCreateUrlBatchUseCase_Execute(t *testing.T) {
	// t.Run("Should create a batch of urls", func(t *testing.T) {
	// 	// Arrange
	// 	urls := []string{"at://did:plc:aca4rpd2skm56qugeb6o4fua/app.bsky.feed.post/3l5nhkzz62d2k"}

	// 	httpClient := http.Client{}
	// 	localDynamo := utils.CreateLocalDynamodbClient(config_api.Configuration{
	// 		Environment:        config_api.Local,
	// 		LocalDynamodbAddr:  "http://localhost:8000",
	// 		AwsSecretAccessKey: "dummysecret",
	// 		AwsApiKey:          "dummykey",
	// 	})

	// 	socialNetworkRepository := bsky.NewBskyService(&httpClient, "", "")
	// 	videoRepository := dynamodb.NewDynamodbVideoRepository(localDynamo, "video")

	// 	createUrlBatchUseCase := NewCreateUrlBatchUseCase(socialNetworkRepository, videoRepository)

	// 	// Act
	// 	videos, err := createUrlBatchUseCase.Execute(urls)

	// 	// Assert
	// 	assert.Nil(t, err)
	// 	assert.NotNil(t, videos)
	// })
}
