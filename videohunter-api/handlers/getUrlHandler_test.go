package handlers

import (
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	"github.com/victoraldir/myvideohunterapi/config"
	local_events "github.com/victoraldir/myvideohunterapi/events"
	real_usecases "github.com/victoraldir/myvideohunterapi/usecases"
	usecases "github.com/victoraldir/myvideohunterapi/usecases/mocks"
	"github.com/victoraldir/myvideohunterapi/utils"
	"go.uber.org/mock/gomock"
)

var getUrlUseCaseMock *usecases.MockGetUrlUseCase

func setupGetUrlHandle(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	getUrlUseCaseMock = usecases.NewMockGetUrlUseCase(ctrl)
}

func TestGetUrlHandle_Handle(t *testing.T) {

	t.Run("Should return html template for reddit video", func(t *testing.T) {
		// Arrange
		setupGetUrlHandle(t)

		os.Setenv("DOWNLOAD_HLS_URL", "https://download-hls.com")

		handler := NewGetUrlHandler(getUrlUseCaseMock)
		reqSample := events.APIGatewayProxyRequest{
			PathParameters: map[string]string{
				"id": "123",
			},
		}

		videoResponse := &local_events.GetVideoResponse{
			Id:               "123",
			ThumbnailUrl:     "http://thumbnail.com",
			Text:             "This is a video",
			CreatedAt:        "2021-09-01",
			OriginalVideoUrl: "https://www.reddit.com/r/2latinoforyou/comments/1cr82fl/pa%C3%ADses_con_m%C3%A1s_homicidios_del_mundo",
			Variants: []local_events.VideoResponseVariant{
				{
					Bitrate:     1000,
					URL:         "https://v.redd.it/b4cikpfnw80d1/HLSPlaylist.m3u8?a=1718310735%2COGE5ZDY5NmE0MzY3NmQyM2UzMTNkNTJkZmMxMmRhNzg4MmM2MzQzNTczYzY0YTYzOGFjMzQwNWQ4ZTViN2I0Zg%3D%3D&amp;v=1&amp;f=sd",
					ContentType: "video/mp4",
				},
			},
		}

		// Act
		getUrlUseCaseMock.EXPECT().Execute("123").Return(videoResponse, nil)

		resp, err := handler.Handle(reqSample)

		// Assert
		assert.Nil(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// assert html content
		assert.Contains(t, resp.Body, "<video")
		assert.Contains(t, resp.Body, "<input type=\"hidden\" id=\"IsTwitter\" value=\"true\">")
		assert.Contains(t, resp.Body, "https://download-hls.com")
		assert.NotContains(t, resp.Body, "<img")
	})

	t.Run("Should return html template for bsky", func(t *testing.T) {
		// Arrange
		dynamoDbLocal := utils.CreateLocalDynamodbClient(config.Configuration{
			Environment:        config.Local,
			LocalDynamodbAddr:  "http://localhost:8000",
			AwsSecretAccessKey: "dummysecret",
			AwsApiKey:          "dummykey",
			VideoTableName:     "video",
		})

		os.Setenv("DOWNLOAD_HLS_URL", "https://download-hls.com")

		videoRepo := dynamodb.NewDynamodbVideoRepository(dynamoDbLocal, "video")

		getUrlUseCase := real_usecases.NewGetUrlUseCase(videoRepo)

		handler := NewGetUrlHandler(getUrlUseCase)
		reqSample := events.APIGatewayProxyRequest{
			PathParameters: map[string]string{
				"id": "YXQ6Ly9kaWQ6cGxjOmRxaXM0ZTI2bHZvaHdwamR2YXloZGI0cC9hcHAuYnNreS5mZWVkLnBvc3QvM2w2Y28zM2lmZjMycA==",
			},
		}

		// Act
		resp, err := handler.Handle(reqSample)

		// Assert
		assert.Nil(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
