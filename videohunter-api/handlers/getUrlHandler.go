package handlers

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"

	api_events "github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohunterapi/usecases"
)

const (
	getVideoTemplate = "getvideo.html"
	errorTemplate    = "error.html"
	templatesGlob    = "templates/*.html"

	// adsenseClientID is the AdSense publisher ID used on the video pages.
	adsenseClientID = "ca-pub-6526073251385305"
	// AdSense ad unit slot IDs. Ads are only rendered when the matching slot
	// ID is filled in. Create "Display" ad units in the AdSense dashboard and
	// paste their slot IDs here to start serving ads on the video pages.
	adsenseSlotTop    = ""
	adsenseSlotBottom = ""
)

//go:embed templates
var res embed.FS

// adUnit is passed to the "adunit" template partial.
type adUnit struct {
	Slot   string
	Client string
}

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
		slog.Error("Error getting video", "videoId", videoId, "error", err)

		return h.renderError(500, "Something went wrong",
			"We could not load this video right now. Please try again in a moment."), nil
	}

	if video == nil {
		slog.Info("Video not found", "videoId", videoId)

		return h.renderError(404, "Video not found",
			"This video may have been deleted, made private, or the link may be incorrect."), nil
	}

	slog.Debug("Video found", "video", video)

	templateFile, err := parseTemplates()
	if err != nil {
		slog.Error("Error parsing templates", "error", err)
		return plainErrorResponse(), nil
	}

	var htmlBuffer bytes.Buffer

	videoMap := map[string]interface{}{
		"Video":          video,
		"DownloadHlsUrl": h.downloadHlsUrl,
		"AdsenseClient":  adsenseClientID,
		"AdTop":          adUnit{Slot: adsenseSlotTop, Client: adsenseClientID},
		"AdBottom":       adUnit{Slot: adsenseSlotBottom, Client: adsenseClientID},
		"JsonLD":         jsonLD(video),
	}

	// Parse the HTML template
	if err := templateFile.ExecuteTemplate(&htmlBuffer, getVideoTemplate, videoMap); err != nil {

		slog.Error("Error parsing template", "error", err)

		return plainErrorResponse(), nil
	}

	return htmlResponse(200, htmlBuffer.String(), "public, max-age=3600, s-maxage=3600"), nil
}

// renderError renders the branded error page so users still get a way to
// download a video (and see ads) when the requested video is unavailable.
func (h *GetUrlHandler) renderError(statusCode int, title, message string) events.APIGatewayProxyResponse {

	templateFile, err := parseTemplates()
	if err != nil {
		slog.Error("Error parsing templates", "error", err)
		return plainErrorResponse()
	}

	var htmlBuffer bytes.Buffer

	errorMap := map[string]interface{}{
		"Title":         title,
		"Message":       message,
		"AdsenseClient": adsenseClientID,
		"AdTop":         adUnit{Slot: adsenseSlotTop, Client: adsenseClientID},
		"AdBottom":      adUnit{Slot: adsenseSlotBottom, Client: adsenseClientID},
	}

	if err := templateFile.ExecuteTemplate(&htmlBuffer, errorTemplate, errorMap); err != nil {
		slog.Error("Error parsing error template", "error", err)
		return plainErrorResponse()
	}

	return htmlResponse(statusCode, htmlBuffer.String(), "no-store")
}

func parseTemplates() (*template.Template, error) {
	return template.ParseFS(res, templatesGlob)
}

func plainErrorResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       "Error",
	}
}

func htmlResponse(statusCode int, body, cacheControl string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
			"Content-Type":                "text/html; charset=utf-8",
			"Cache-Control":               cacheControl,
		},
	}
}

// jsonLD builds the structured data embedded in the video page.
func jsonLD(video *api_events.GetVideoResponse) template.JS {

	type webSite struct {
		Type string `json:"@type"`
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	payload := struct {
		Context     string  `json:"@context"`
		Type        string  `json:"@type"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		URL         string  `json:"url"`
		Thumbnail   string  `json:"thumbnailUrl,omitempty"`
		IsPartOf    webSite `json:"isPartOf"`
	}{
		Context:     "https://schema.org",
		Type:        "WebPage",
		Name:        video.PageTitle(),
		Description: video.PageDescription(),
		URL:         video.CanonicalURL(),
		Thumbnail:   video.ThumbnailUrl,
		IsPartOf:    webSite{Type: "WebSite", Name: "Video Hunter", URL: "https://www.myvideohunter.com/"},
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Error encoding JSON-LD", "error", err)
		return ""
	}

	return template.JS(encoded)
}
