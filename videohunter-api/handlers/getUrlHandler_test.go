package handlers

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	local_events "github.com/victoraldir/myvideohunterapi/events"
	usecases "github.com/victoraldir/myvideohunterapi/usecases/mocks"
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
		assert.Contains(t, resp.Body, "<input type=\"hidden\" id=\"IsRedditVideo\" value=\"true\">")
		assert.NotContains(t, resp.Body, "<img")
	})
}
