package handlers

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	events_api "github.com/victoraldir/myvideohunterapi/events"
	usecases "github.com/victoraldir/myvideohunterapi/usecases/mocks"
	"go.uber.org/mock/gomock"
)

var videoDownloaderUseCaseMock *usecases.MockVideoDownloaderUseCase

func setup(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	videoDownloaderUseCaseMock = usecases.NewMockVideoDownloaderUseCase(ctrl)
}

func TestHandler(t *testing.T) {

	setup(t)

	videoResponse := &events_api.CreateVideoResponse{
		Id: "123",
		// Variants: []events_api.VideoResponseVariant{
		// 	{
		// 		Bitrate:     100,
		// 		URL:         "http://www.google.com",
		// 		ContentType: "video/mp4",
		// 	},
		// },
	}

	videoDownloaderUseCaseMock.EXPECT().Execute(gomock.Any()).Return(videoResponse, nil).AnyTimes()

	createUrlHandler := CreateUrlHandler{
		VideoDownloaderUseCase: videoDownloaderUseCaseMock,
	}

	testCases := []struct {
		name          string
		request       events.APIGatewayProxyRequest
		expectedBody  string
		expectedError error
		expectedCode  int
	}{
		{
			// mock a request with a valid video_url
			name: "Valid video_url",
			request: events.APIGatewayProxyRequest{
				Body: "{\"video_url\":\"https://twitter.com/victoraldir/status/1746348632146690148\"}",
			},
			expectedBody:  "{\"id\":\"123\",\"thumbnail_url\":\"\",\"description\":\"\"}",
			expectedError: nil,
			expectedCode:  200,
		},
		{
			// mock a request with empty body
			name:          "Empty body",
			request:       events.APIGatewayProxyRequest{},
			expectedBody:  "Invalid Request",
			expectedError: nil,
			expectedCode:  400,
		},
		{
			// mock a non-twitter url
			name: "Empty body",
			request: events.APIGatewayProxyRequest{
				Body: "{\"video_url\":\"https://reddit.com/elonmusk/status/1273792507348928512\"}",
			},
			expectedBody:  "Invalid video_url",
			expectedError: nil,
			expectedCode:  400,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response, err := createUrlHandler.Handle(testCase.request)
			if err != testCase.expectedError {
				t.Errorf("Expected error %v, but got %v", testCase.expectedError, err)
			}

			if response.Body != testCase.expectedBody {
				t.Errorf("Expected response %v, but got %v", testCase.expectedBody, response.Body)
			}

			if response.StatusCode != testCase.expectedCode {
				t.Errorf("Expected status code %v, but got %v", testCase.expectedCode, response.StatusCode)
			}
		})
	}
}
