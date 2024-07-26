package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	events_aws "github.com/aws/aws-lambda-go/events"
	"golang.org/x/exp/slog"

	"github.com/victoraldir/myvideohunterapi/usecases"
)

type MixAudioVideoHandler struct {
	MixAudioVideoUseCase usecases.MixAudioVideoUseCase
}

func NewMixAudioVideoHandler(mixAudioVideoUseCase usecases.MixAudioVideoUseCase) *MixAudioVideoHandler {
	return &MixAudioVideoHandler{
		MixAudioVideoUseCase: mixAudioVideoUseCase,
	}
}

func (h *MixAudioVideoHandler) Handle(request *events_aws.LambdaFunctionURLRequest) (*events_aws.LambdaFunctionURLStreamingResponse, error) {

	log.Printf("Handling request: %v", request)

	videoRequest := &VideoRequest{}

	// Get audioUrl from query parameters
	audioUrl := request.QueryStringParameters["audio_url"]

	// Get videoUrl from query parameters
	videoUrl := request.QueryStringParameters["video_url"]

	// Video and audio URLs are required
	if audioUrl == "" && videoUrl == "" {
		slog.Error("Missing video_url and audio_url")
		return &events_aws.LambdaFunctionURLStreamingResponse{
			Body:       strings.NewReader("Missing video_url and audio_url"),
			Headers:    map[string]string{"Content-Type": "text/plain"},
			StatusCode: 400,
		}, nil
	}

	videoRequest.AudioUrl = audioUrl
	videoRequest.VideoUrl = videoUrl

	if videoRequest.AudioUrl == "" || videoRequest.VideoUrl == "" {
		slog.Error("Missing video_url or audio_url")
		return &events_aws.LambdaFunctionURLStreamingResponse{
			Body:       strings.NewReader("Missing video_url or audio_url"),
			Headers:    map[string]string{"Content-Type": "text/plain"},
			StatusCode: 400,
		}, nil
	}

	videoResponse, err := h.MixAudioVideoUseCase.Execute(videoRequest.VideoUrl, videoRequest.AudioUrl)

	if err != nil {
		slog.Error("Error downloading video: ", err)
		return &events_aws.LambdaFunctionURLStreamingResponse{
			Body:       strings.NewReader("Error downloading video"),
			StatusCode: 500,
		}, nil
	}

	// Pipe the video file to the output stream
	pr, pw := io.Pipe()

	go func() {
		// Input stream
		log.Printf("Opening video file: %s", videoResponse.VideoPath)
		file, err := os.Open(videoResponse.VideoPath)

		if err != nil {
			log.Printf("Error opening video file: %s", err)
		}

		io.Copy(pw, file)
	}()

	return &events_aws.LambdaFunctionURLStreamingResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":        "video/mp4",
			"Content-Disposition": "attachment; filename=videofile.mp4",
		},
		Body: pr,
	}, nil
}
