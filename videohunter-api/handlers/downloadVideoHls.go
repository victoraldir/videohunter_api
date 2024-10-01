package handlers

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/victoraldir/myvideohunterapi/usecases"
)

type DownalodRequest struct {
	Url string `json:"url"`
}

type DownloadVideoHlsHandler struct {
	DownloadVideoHlsUseCase usecases.DownloadVideoHlsUseCase
}

func NewDownloadVideoHlsHandler(downloadVideoHlsUseCase usecases.DownloadVideoHlsUseCase) *DownloadVideoHlsHandler {
	return &DownloadVideoHlsHandler{
		DownloadVideoHlsUseCase: downloadVideoHlsUseCase,
	}
}

func (h *DownloadVideoHlsHandler) Handle(request *events.LambdaFunctionURLRequest) (*events.LambdaFunctionURLStreamingResponse, error) {

	log.Println("Request download: ", request)

	// Get video from query parameters
	url := request.QueryStringParameters["url"]

	// log.Println("Body: ", body)
	// unscapeBody := strings.Replace(body, "\\\"", "\"", -1)
	// log.Println("Unscape body: ", unscapeBody)

	downloadRequest := &DownalodRequest{}
	// err := json.Unmarshal([]byte(unscapeBody), downloadRequest)
	downloadRequest.Url = url
	// if err != nil {
	// 	log.Println("Error unmarshalling request: ", err)
	// 	return &events.LambdaFunctionURLStreamingResponse{
	// 		Body:       strings.NewReader("Invalid Request"),
	// 		Headers:    map[string]string{"Content-Type": "text/plain"},
	// 		StatusCode: 400,
	// 	}, nil
	// }

	// log.Println("Decoding url: ", downloadRequest.Url)

	// if err != nil {
	// 	log.Println("Error decoding url: ", err)
	// 	return &events.LambdaFunctionURLStreamingResponse{
	// 		Body:       strings.NewReader("Invalid url"),
	// 		Headers:    map[string]string{"Content-Type": "text/plain"},
	// 		StatusCode: 400,
	// 	}, nil
	// }

	log.Println("Downloading video from: ", downloadRequest.Url)

	videoResponse, err := h.DownloadVideoHlsUseCase.Execute(downloadRequest.Url)

	if err != nil {
		log.Println("Error downloading video: ", err)
		return &events.LambdaFunctionURLStreamingResponse{
			Body:       strings.NewReader("Error downloading video"),
			Headers:    map[string]string{"Content-Type": "text/plain"},
			StatusCode: 500,
		}, nil
	}

	if err != nil {
		log.Println("Error marshalling response: ", err)
		return &events.LambdaFunctionURLStreamingResponse{
			Body:       strings.NewReader("Error marshalling response"),
			Headers:    map[string]string{"Content-Type": "text/plain"},
			StatusCode: 500,
		}, nil
	}

	// Input stream
	log.Println("Opening video file: ", videoResponse.VideoPath)
	file, err := os.Open(videoResponse.VideoPath)

	if err != nil {
		log.Println("Error opening video file: ", err)
	}

	// Read the video file
	content, err := io.ReadAll(file)

	if err != nil {
		log.Println("Error reading video file: ", err)
	}

	return &events.LambdaFunctionURLStreamingResponse{
		Body: strings.NewReader(string(content)),
		Headers: map[string]string{
			"Content-Type":        "video/mp4",
			"Content-Disposition": "attachment; filename=myfile.mp4",
		},
		StatusCode: 200,
	}, nil
}
