package handlers

import (
	"encoding/json"
	"log"
	"strings"

	events_aws "github.com/aws/aws-lambda-go/events"
	"github.com/victoraldir/myvideohunterapi/usecases"
	"github.com/victoraldir/myvideohunterapi/utils"
	"github.com/victoraldir/myvideohuntershared/services/reddit"
	"golang.org/x/exp/slog"
)

type CreateUrlHandler struct {
	VideoDownloaderUseCase usecases.VideoDownloaderUseCase
	// RedditDownloaderUseCase usecases.VideoDownloaderUseCase
}

type VideoRequest struct {
	VideoUrl string `json:"video_url"`
	AudioUrl string `json:"audio_url"`
}

func NewCreateUrlHandler(videoDownloaderUseCase usecases.VideoDownloaderUseCase) CreateUrlHandler {
	return CreateUrlHandler{
		VideoDownloaderUseCase: videoDownloaderUseCase,
	}
}

func (h *CreateUrlHandler) Handle(request events_aws.APIGatewayProxyRequest) (events_aws.APIGatewayProxyResponse, error) {
	videoRequest := &VideoRequest{}

	log.Println("Request: ", request)

	body := request.Body

	err := json.Unmarshal([]byte(body), videoRequest)

	if err != nil {
		slog.Error("Error unmarshalling request: ", err)
		return events_aws.APIGatewayProxyResponse{
			Body:       "Invalid Request",
			StatusCode: 400,
		}, nil
	}

	// Trim video_url
	videoRequest.VideoUrl = strings.TrimSpace(videoRequest.VideoUrl)

	slog.Debug("Downloading video from: ", videoRequest)

	if !utils.IsTwitterUrl(videoRequest.VideoUrl) &&
		!utils.IsRedditUrl(videoRequest.VideoUrl) &&
		!utils.IsBskyUrl(videoRequest.VideoUrl) {
		slog.Error("Invalid video_url: ", videoRequest.VideoUrl)
		return events_aws.APIGatewayProxyResponse{
			Body:       "Invalid video_url",
			StatusCode: 400,
		}, nil
	}

	videoResponse, err := h.VideoDownloaderUseCase.Execute(videoRequest.VideoUrl)

	if err != nil {

		if _, ok := err.(*reddit.InvalidPostError); ok {
			slog.Error("Error downloading video: ", err)
			return events_aws.APIGatewayProxyResponse{
				Body:       "Invalid video_url",
				StatusCode: 400,
			}, nil
		}

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
