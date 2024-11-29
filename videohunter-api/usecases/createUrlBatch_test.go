package usecases

// import (
// 	"net/http"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/victoraldir/myvideohunterapi/adapters/bsky"
// 	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
// 	"github.com/victoraldir/myvideohunterapi/utils"

// 	config_api "github.com/victoraldir/myvideohunterapi/config"
// )

// func TestCreateUrlBatchUseCase_Execute(t *testing.T) {
// 	t.Run("Should create a batch of urls", func(t *testing.T) {
// 		// Arrange
// 		urls := []string{"at://did:plc:dqis4e26lvohwpjdvayhdb4p/app.bsky.feed.post/3l6co33iff322"}

// 		httpClient := http.Client{}
// 		localDynamo := utils.CreateLocalDynamodbClient(config_api.Configuration{
// 			Environment:        config_api.Local,
// 			LocalDynamodbAddr:  "http://localhost:8000",
// 			AwsSecretAccessKey: "dummysecret",
// 			AwsApiKey:          "dummykey",
// 		})

// 		socialNetworkRepository := bsky.NewBskyDownloaderRepository(&httpClient)
// 		videoRepository := dynamodb.NewDynamodbVideoRepository(localDynamo, "video")

// 		createUrlBatchUseCase := NewCreateUrlBatchUseCase(socialNetworkRepository, videoRepository)

// 		// Act
// 		videos, err := createUrlBatchUseCase.Execute(urls)

// 		// Assert
// 		assert.Nil(t, err)
// 		assert.NotNil(t, videos)
// 	})

// }
