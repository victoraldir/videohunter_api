package usecases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	"github.com/victoraldir/myvideohunterapi/config"
	"github.com/victoraldir/myvideohunterapi/utils"
)

func TestGetUrlUseCase_Execute(t *testing.T) {
	t.Run("Should get a video url", func(t *testing.T) {

		// Arrange
		dynamoDbLocal := utils.CreateLocalDynamodbClient(config.Configuration{
			Environment:        config.Local,
			LocalDynamodbAddr:  "http://localhost:8000",
			AwsSecretAccessKey: "dummysecret",
			AwsApiKey:          "dummykey",
			VideoTableName:     "video",
		})

		videoRepo := dynamodb.NewDynamodbVideoRepository(dynamoDbLocal, "video")

		getUrlUseCase := NewGetUrlUseCase(videoRepo)

		// Act
		video, err := getUrlUseCase.Execute("YXQ6Ly9kaWQ6cGxjOmRxaXM0ZTI2bHZvaHdwamR2YXloZGI0cC9hcHAuYnNreS5mZWVkLnBvc3QvM2w2Y28zM2lmZjMycA==")

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, video)

	})
}
