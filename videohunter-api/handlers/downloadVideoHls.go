package handlers

import (
	"io"
	"log"
	"net/url"
	"os"
	"path"
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
	encodedUrl := request.QueryStringParameters["url"]
	decodedUrl, err := url.QueryUnescape(encodedUrl)
	if err != nil {
		log.Println("Error decoding URL: ", err)
		decodedUrl = encodedUrl
	}
	log.Println("Decoded URL: ", decodedUrl)

	// log.Println("Body: ", body)
	// unscapeBody := strings.Replace(body, "\\\"", "\"", -1)
	// log.Println("Unscape body: ", unscapeBody)

	downloadRequest := &DownalodRequest{}
	// err := json.Unmarshal([]byte(unscapeBody), downloadRequest)
	downloadRequest.Url = decodedUrl
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

	log.Println("Downloading video from: ", decodedUrl)

	videoResponse, err := h.DownloadVideoHlsUseCase.Execute(decodedUrl)

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

	// "inline" serves the video in the browser instead of forcing a download.
	// On iPhone and iPad a forced download lands in the Files app, where it
	// has to be found again before it can be shared; opened inline, the system
	// player offers the share sheet that saves the video to Photos or sends it
	// to any app in one tap.
	disposition := "attachment"
	if request.QueryStringParameters["inline"] == "1" {
		disposition = "inline"
	}

	return &events.LambdaFunctionURLStreamingResponse{
		Body: strings.NewReader(string(content)),
		Headers: map[string]string{
			"Content-Type":        "video/mp4",
			"Content-Disposition": disposition + `; filename="` + videoFilename(decodedUrl) + `"`,
		},
		StatusCode: 200,
	}, nil
}

// videoFilename derives a stable, filesystem-safe file name from the
// requested video URL. The download page names its blob downloads itself, so
// this name is only seen when the video is opened directly or inline.
func videoFilename(rawUrl string) string {
	const fallback = "videohunter.mp4"

	parsed, err := url.Parse(rawUrl)
	if err != nil || parsed.Path == "" {
		return fallback
	}

	base := strings.TrimSuffix(path.Base(parsed.Path), path.Ext(parsed.Path))

	// Keep only the characters every file system accepts, so a CDN file name
	// can never produce a broken or hostile Content-Disposition value.
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return -1
		}
	}, base)

	if base == "" {
		return fallback
	}

	if len(base) > 50 {
		base = base[:50]
	}

	return "videohunter-" + base + ".mp4"
}
