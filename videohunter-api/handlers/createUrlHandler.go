package handlers

import (
	"encoding/json"
	"log"

	events_aws "github.com/aws/aws-lambda-go/events"
	"github.com/victoraldir/myvideohunterapi/usecases"
	"github.com/victoraldir/myvideohunterapi/utils"
	"golang.org/x/exp/slog"
)

type CreateUrlHandler struct {
	VideoDownloaderUseCase  usecases.VideoDownloaderUseCase
	RedditDownloaderUseCase usecases.VideoDownloaderUseCase
}

type VideoRequest struct {
	VideoUrl string `json:"video_url"`
}

func NewCreateUrlHandler(videoDownloaderUseCase usecases.VideoDownloaderUseCase) CreateUrlHandler {
	return CreateUrlHandler{
		VideoDownloaderUseCase: videoDownloaderUseCase,
	}
}

func (h *CreateUrlHandler) Handle(request events_aws.APIGatewayProxyRequest) (events_aws.APIGatewayProxyResponse, error) {
	videoRequest := &VideoRequest{}

	slog.Debug("Request: ", request)

	// Decode from base64
	bodyDecoded, _ := utils.Base64Decode(request.Body)

	log.Println("Body decoded: ", bodyDecoded)
	err := json.Unmarshal([]byte(bodyDecoded), videoRequest)

	if err != nil {
		slog.Error("Error unmarshalling request: ", err)
		return events_aws.APIGatewayProxyResponse{
			Body:       "Invalid Request",
			StatusCode: 400,
		}, nil
	}

	slog.Debug("Downloading video from: ", videoRequest)

	if !utils.IsTwitterUrl(videoRequest.VideoUrl) && !utils.IsRedditUrl(videoRequest.VideoUrl) {
		slog.Error("Invalid video_url: ", videoRequest.VideoUrl)
		return events_aws.APIGatewayProxyResponse{
			Body:       "Invalid video_url",
			StatusCode: 400,
		}, nil
	}

	var videoDownloaderUseCase usecases.VideoDownloaderUseCase

	if utils.IsRedditUrl(videoRequest.VideoUrl) {
		videoDownloaderUseCase = h.RedditDownloaderUseCase
	} else {
		videoDownloaderUseCase = h.VideoDownloaderUseCase
	}

	videoResponse, err := videoDownloaderUseCase.Execute(videoRequest.VideoUrl)

	if err != nil {
		slog.Error("Error downloading video: ", err)
		return events_aws.APIGatewayProxyResponse{
			Body:       "Error downloading video",
			StatusCode: 500,
		}, nil
	}

	videoResponseJson, err := json.Marshal(videoResponse)

	if err != nil {
		slog.Error("Error marshalling response: ", err)
		return events_aws.APIGatewayProxyResponse{
			Body:       "Error marshalling response",
			StatusCode: 500,
		}, nil
	}

	// Set CORS headers for the preflight request
	headers := map[string]string{
		"Content-Type":                "application/json",
		"Access-Control-Allow-Origin": "*",
	}

	return events_aws.APIGatewayProxyResponse{
		Body:       string(videoResponseJson),
		StatusCode: 200,
		Headers:    headers,
	}, nil
}
