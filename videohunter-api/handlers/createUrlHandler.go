package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
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
		return jsonResponse(http.StatusBadRequest, "Invalid request."), nil
	}

	// Trim video_url
	videoRequest.VideoUrl = strings.TrimSpace(videoRequest.VideoUrl)

	slog.Debug("Downloading video from: ", videoRequest)

	if !utils.IsTwitterUrl(videoRequest.VideoUrl) &&
		!utils.IsRedditUrl(videoRequest.VideoUrl) &&
		!utils.IsBskyUrl(videoRequest.VideoUrl) {
		slog.Error("Invalid video_url: ", videoRequest.VideoUrl)
		return jsonResponse(http.StatusBadRequest,
			"That link is not supported. Paste a link to a post from X (Twitter), Reddit or Bluesky."), nil
	}

	videoResponse, err := h.VideoDownloaderUseCase.Execute(videoRequest.VideoUrl)

	if err != nil {
		return h.errorResponse(err), nil
	}

	videoResponseJson, err := json.Marshal(videoResponse)

	if err != nil {
		slog.Error("Error marshalling response", "error", err)
		return jsonResponse(http.StatusInternalServerError, "Something went wrong. Please try again."), nil
	}

	return jsonResponseWithBody(http.StatusOK, string(videoResponseJson)), nil
}

// errorResponse maps a download failure to an HTTP response.
//
// The distinction matters to the user: a link that cannot work (deleted post,
// post without a video) is not the same thing as a platform that is down or
// rate limiting us, and only the first one is worth fixing by pasting
// something else. Bodies are JSON with a "message" field, which the website
// already shows verbatim.
func (h *CreateUrlHandler) errorResponse(err error) events_aws.APIGatewayProxyResponse {

	var invalidPost *reddit.InvalidPostError
	if errors.As(err, &invalidPost) {
		slog.Info("Post has no downloadable video", "status", invalidPost.StatusCode, "error", invalidPost.Err)
		return jsonResponse(http.StatusNotFound,
			"That post has no downloadable video. It may have been deleted, or it may not be a video.")
	}

	var upstream *reddit.UpstreamError
	if errors.As(err, &upstream) {
		slog.Error("Upstream platform unavailable", "status", upstream.StatusCode, "error", upstream.Err)
		return jsonResponse(http.StatusBadGateway,
			"The platform is not responding right now. Please try again in a moment.")
	}

	slog.Error("Error downloading video", "error", err)
	return jsonResponse(http.StatusInternalServerError,
		"Something went wrong while fetching the video. Please try again.")
}

func jsonResponse(statusCode int, message string) events_aws.APIGatewayProxyResponse {
	body, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		slog.Error("Error marshalling error response", "error", err)
		return events_aws.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"message":"Something went wrong. Please try again."}`,
		}
	}

	return jsonResponseWithBody(statusCode, string(body))
}

func jsonResponseWithBody(statusCode int, body string) events_aws.APIGatewayProxyResponse {
	return events_aws.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
	}
}
