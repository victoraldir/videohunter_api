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

	videoRequest.AudioUrl = "https://v.redd.it/6i6fu75bme2d1/DASH_AUDIO_128.mp4"
	videoRequest.VideoUrl = "https://v.redd.it/6i6fu75bme2d1/DASH_480.mp4?source=fallback"

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

	if err != nil {
		slog.Error("Error marshalling response: ", err)
		return &events_aws.LambdaFunctionURLStreamingResponse{
			Body:       strings.NewReader("Error marshalling response"),
			StatusCode: 500,
		}, nil
	}

	// Pipe the video file to the output stream
	pr, pw := io.Pipe()

	// TeeReader gets the data from the file and also writes it to the PipeWriter
	// tr := io.TeeReader(file, w)

	go func() {
		pw.Close()
		// Input stream
		log.Printf("Opening video file: %s", videoResponse.VideoPath)
		file, err := os.Open(videoResponse.VideoPath)

		if err != nil {
			log.Printf("Error opening video file: %s", err)
		}

		io.Copy(pw, file)
	}()

	if err != nil {
		log.Printf("Error reading video file: %s", err)
	}

	return &events_aws.LambdaFunctionURLStreamingResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                           "video/mp4",
			"Content-Disposition":                    "attachment; filename=videofile.mp4",
			"Lambda-Runtime-Function-Response-Model": "streaming",
		},
		Body: pr,
	}, nil
}
