package handlers

import (
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/victoraldir/myvideohunterapi/adapters/bsky"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	"github.com/victoraldir/myvideohunterapi/config"
	"github.com/victoraldir/myvideohunterapi/usecases"
	"github.com/victoraldir/myvideohunterapi/utils"
)

func TestCreateUrlBatchHandler_Handle(t *testing.T) {

	t.Run("Should return html template for bsky", func(t *testing.T) {
		// Arrange
		dynamoDbLocal := utils.CreateLocalDynamodbClient(config.Configuration{
			Environment:        config.Local,
			LocalDynamodbAddr:  "http://localhost:8000",
			AwsSecretAccessKey: "dummysecret",
			AwsApiKey:          "dummykey",
			VideoTableName:     "video",
		})

		httpClient := http.Client{}

		videoRepo := dynamodb.NewDynamodbVideoRepository(dynamoDbLocal, "video")
		socialMediaRepo := bsky.NewBskyDownloaderRepository(&httpClient)

		getUrlUseCase := usecases.NewCreateUrlBatchUseCase(socialMediaRepo, videoRepo)

		handler := NewCreateUrlBatchHandler(getUrlUseCase)

		// Act
		resp, err := handler.Handle(events.APIGatewayProxyRequest{
			Body: "{\"uris\":[\"https://www.bsky.com/123\"]}",
		})

		// Assert
		assert.Nil(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

}
