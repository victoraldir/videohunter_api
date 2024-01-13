package handlers

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/victoraldir/myvideohunterapi/usecases"
	"github.com/victoraldir/myvideohunterapi/utils"
	"golang.org/x/exp/slog"
)

type CreateUrlHandler struct {
	VideoDownloaderUseCase usecases.VideoDownloaderUseCase
}

type VideoRequest struct {
	VideoUrl string `json:"video_url"`
}

func NewCreateUrlHandler(videoDownloaderUseCase usecases.VideoDownloaderUseCase) CreateUrlHandler {
	return CreateUrlHandler{
		VideoDownloaderUseCase: videoDownloaderUseCase,
	}
}

func (h *CreateUrlHandler) Handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	videoRequest := &VideoRequest{}

	err := json.Unmarshal([]byte(request.Body), videoRequest)

	if err != nil {
		slog.Error("Error unmarshalling request: ", err)
		return events.APIGatewayProxyResponse{
			Body:       "Invalid Request",
			StatusCode: 400,
		}, nil
	}

	slog.Debug("Downloading video from: ", videoRequest)

	if !utils.IsTwitterUrl(videoRequest.VideoUrl) {
		slog.Error("Invalid video_url: ", videoRequest.VideoUrl)
		return events.APIGatewayProxyResponse{
			Body:       "Invalid video_url",
			StatusCode: 400,
		}, nil
	}

	videoResponse, err := h.VideoDownloaderUseCase.Execute(videoRequest.VideoUrl)

	if err != nil {
		slog.Error("Error downloading video: ", err)
		return events.APIGatewayProxyResponse{
			Body:       "Error downloading video",
			StatusCode: 500,
		}, nil
	}

	videoResponseJson, err := json.Marshal(videoResponse)

	if err != nil {
		slog.Error("Error marshalling response: ", err)
		return events.APIGatewayProxyResponse{
			Body:       "Error marshalling response",
			StatusCode: 500,
		}, nil
	}

	// Set CORS headers for the preflight request
	headers := map[string]string{
		"Content-Type":                "application/json",
		"Access-Control-Allow-Origin": "*",
	}

	return events.APIGatewayProxyResponse{
		Body:       string(videoResponseJson),
		StatusCode: 200,
		Headers:    headers,
	}, nil
}
