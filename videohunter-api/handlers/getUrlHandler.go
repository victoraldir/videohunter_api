package handlers

import (
	"bytes"
	"embed"
	"html/template"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"

	"github.com/victoraldir/myvideohunterapi/usecases"
)

const (
	getVideoTemplate = "templates/getvideo.html"
)

//go:embed templates
var res embed.FS

type GetUrlHandler struct {
	GerUrlUseCase  usecases.GetUrlUseCase
	downloadHlsUrl string
}

func NewGetUrlHandler(getUrlUseCase usecases.GetUrlUseCase) *GetUrlHandler {

	url := os.Getenv("DOWNLOAD_HLS_URL")

	if url == "" {
		slog.Error("DOWNLOAD_HLS_URL is required")
		os.Exit(1)
	}

	return &GetUrlHandler{
		GerUrlUseCase:  getUrlUseCase,
		downloadHlsUrl: url,
	}
}

func (h *GetUrlHandler) Handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	videoId := request.PathParameters["id"]

	slog.Debug("Getting video", "videoId", videoId)

	video, err := h.GerUrlUseCase.Execute(videoId)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error",
		}, nil
	}

	slog.Debug("Video found", "video", video)

	// Use the embedded HTML template
	templateFile, err := template.ParseFS(res, getVideoTemplate)
	if err != nil {
		panic(err)
	}

	var htmlBuffer bytes.Buffer

	videoMap := map[string]interface{}{
		"Video":          video,
		"DownloadHlsUrl": h.downloadHlsUrl,
	}

	// Parse the HTML template
	if err := templateFile.Execute(&htmlBuffer, videoMap); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error",
		}, nil
	}

	// Set CORS headers for the preflight request
	headers := map[string]string{
		"Access-Control-Allow-Origin": "*",
		"Content-Type":                "text/html",
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       htmlBuffer.String(),
		Headers:    headers,
	}, nil
}
