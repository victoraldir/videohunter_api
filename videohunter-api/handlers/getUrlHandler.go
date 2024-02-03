package handlers

import (
	"bytes"
	"embed"
	"html/template"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"

	"github.com/victoraldir/myvideohunterapi/usecases"
)

const (
	getVideoTemplate = "templates/getvideo.html"
)

//go:embed templates
var res embed.FS

type GetUrlHandler struct {
	GerUrlUseCase usecases.GetUrlUseCase
}

func NewGetUrlHandler(getUrlUseCase usecases.GetUrlUseCase) *GetUrlHandler {
	return &GetUrlHandler{
		GerUrlUseCase: getUrlUseCase,
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
		"Video": video,
	}

	// Parse the HTML template
	err = templateFile.Execute(&htmlBuffer, videoMap)
	if err != nil {
		panic(err)
	}

	if err != nil {
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
