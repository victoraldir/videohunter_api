package handlers

import (
	"bufio"
	"log"
	"os"
	"strings"

	events_aws "github.com/aws/aws-lambda-go/events"

	"github.com/victoraldir/myvideohunterapi/usecases"
	"golang.org/x/exp/slog"
)

type MixAudioVideoHandler struct {
	MixAudioVideoUseCase usecases.MixAudioVideoUseCase
}

func NewMixAudioVideoHandler(mixAudioVideoUseCase usecases.MixAudioVideoUseCase) *MixAudioVideoHandler {
	return &MixAudioVideoHandler{
		MixAudioVideoUseCase: mixAudioVideoUseCase,
	}
}

func (h *MixAudioVideoHandler) Handle(request events_aws.APIGatewayProxyRequest) (events_aws.LambdaFunctionURLStreamingResponse, error) {

	log.Printf("Handling request: %v", request)

	videoUrl := request.QueryStringParameters["video_url"]
	audioUrl := request.QueryStringParameters["audio_url"]

	if videoUrl == "" || audioUrl == "" {
		slog.Error("Missing video_url or audio_url")
		return events_aws.LambdaFunctionURLStreamingResponse{
			Body:       strings.NewReader("Missing video_url or audio_url"),
			Headers:    map[string]string{"Content-Type": "text/plain"},
			StatusCode: 400,
		}, nil
	}

	videoRequest := &VideoRequest{
		VideoUrl: videoUrl,
		AudioUrl: audioUrl,
	}

	videoResponse, err := h.MixAudioVideoUseCase.Execute(videoRequest.VideoUrl, videoRequest.AudioUrl)

	if err != nil {
		slog.Error("Error downloading video: ", err)
		return events_aws.LambdaFunctionURLStreamingResponse{
			Body:       strings.NewReader("Error downloading video"),
			StatusCode: 500,
		}, nil
	}

	if err != nil {
		slog.Error("Error marshalling response: ", err)
		return events_aws.LambdaFunctionURLStreamingResponse{
			Body:       strings.NewReader("Error marshalling response"),
			StatusCode: 500,
		}, nil
	}

	// Input stream
	log.Printf("Opening video file: %s", videoResponse.VideoPath)
	file, err := os.Open(videoResponse.VideoPath)

	if err != nil {
		log.Printf("Error opening video file: %s", err)
	}

	// Read the video file
	// content, err := io.ReadAll(file)

	if err != nil {
		log.Printf("Error reading video file: %s", err)
	}

	return events_aws.LambdaFunctionURLStreamingResponse{
		Body:       bufio.NewReader(file),
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":        "video/mp4",
			"Content-Disposition": "attachment; filename=video.mp4",
		},
	}, nil
}
