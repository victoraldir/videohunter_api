package handlers

import (
	"encoding/base64"
	"io"
	"os"

	events_aws "github.com/aws/aws-lambda-go/events"
	"github.com/victoraldir/myvideohunterapi/usecases"
	"golang.org/x/exp/slog"
)

type DownloadVideoHlsHandler struct {
	DownloadVideoHlsUseCase usecases.DownloadVideoHlsUseCase
}

func NewDownloadVideoHlsHandler(downloadVideoHlsUseCase usecases.DownloadVideoHlsUseCase) *DownloadVideoHlsHandler {
	return &DownloadVideoHlsHandler{
		DownloadVideoHlsUseCase: downloadVideoHlsUseCase,
	}
}

func (h *DownloadVideoHlsHandler) Handle(request events_aws.APIGatewayProxyRequest) (events_aws.APIGatewayProxyResponse, error) {

	slog.Debug("Handling request test: ", request)

	// Get video from query parameters
	urlBase64Enconded := request.QueryStringParameters["url"]

	slog.Debug("Decoding url: ", urlBase64Enconded)
	urlDecoded, err := base64.StdEncoding.DecodeString(urlBase64Enconded)

	if err != nil {
		slog.Error("Error decoding url: ", err)
		return events_aws.APIGatewayProxyResponse{
			Body:       "Error decoding url",
			StatusCode: 400,
		}, nil
	}

	videoRequest := &VideoRequest{
		VideoUrl: string(urlDecoded),
	}

	slog.Debug("Downloading video from: ", videoRequest)

	videoResponse, err := h.DownloadVideoHlsUseCase.Execute(videoRequest.VideoUrl)

	if err != nil {
		slog.Error("Error downloading video: ", err)
		return events_aws.APIGatewayProxyResponse{
			Body:       "Error downloading video",
			StatusCode: 500,
		}, nil
	}

	if err != nil {
		slog.Error("Error marshalling response: ", err)
		return events_aws.APIGatewayProxyResponse{
			Body:       "Error marshalling response",
			StatusCode: 500,
		}, nil
	}

	// Input stream
	slog.Debug("Opening video file: ", videoResponse.VideoPath)
	file, err := os.Open(videoResponse.VideoPath)

	if err != nil {
		slog.Error("Error opening video file: ", err)
	}

	// Read the video file
	content, err := io.ReadAll(file)

	if err != nil {
		slog.Error("Error reading video file: ", err)
	}

	return events_aws.APIGatewayProxyResponse{
		Body:            base64.StdEncoding.EncodeToString(content),
		StatusCode:      200,
		IsBase64Encoded: true,
		Headers: map[string]string{
			"Content-Type":        "video/mp4",
			"Content-Disposition": "attachment; filename=video.mp4",
		},
	}, nil

}
