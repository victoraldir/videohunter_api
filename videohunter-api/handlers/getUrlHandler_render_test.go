package handlers

import (
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	api_events "github.com/victoraldir/myvideohunterapi/events"
)

type stubGetUrlUseCase struct {
	video *api_events.GetVideoResponse
	err   error
}

func (s stubGetUrlUseCase) Execute(videoId string) (*api_events.GetVideoResponse, error) {
	return s.video, s.err
}

func sampleTwitterVideo() *api_events.GetVideoResponse {
	return &api_events.GetVideoResponse{
		Id:               "abc123",
		ThumbnailUrl:     "https://pbs.twimg.com/thumb.jpg",
		Text:             "Activistas de la acampada en Sol expulsan a provocadores",
		CreatedAt:        "2026-09-01",
		OriginalVideoUrl: "https://twitter.com/eeugee_/status/2105038695577661785",
		Variants: []api_events.VideoResponseVariant{
			{
				Bitrate:     832000,
				URL:         "https://video.twimg.com/amplify_video/2105038602392440832/vid/avc1/640x360/CNbyVdA7KdXLUQkM.mp4?tag=29",
				ContentType: "video/mp4",
			},
			{
				Bitrate:     0,
				URL:         "https://video.twimg.com/amplify_video/2105038602392440832/pl/FuaF7_-c0ughEZU3.m3u8?tag=29&v=cfc",
				ContentType: "application/x-mpegURL",
			},
		},
	}
}

func TestGetUrlHandle_Handle_RendersVideoPage(t *testing.T) {

	handler := &GetUrlHandler{GerUrlUseCase: stubGetUrlUseCase{video: sampleTwitterVideo()}}

	response, err := handler.Handle(events.APIGatewayProxyRequest{
		PathParameters: map[string]string{"id": "abc123"},
	})

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)

	body := response.Body

	// Unique, keyword oriented metadata.
	assert.Contains(t, body, "<title>Download X (Twitter) video: Activistas de la acampada en Sol expulsan a provocadores — Video Hunter</title>")
	assert.Contains(t, body, `<link rel="canonical" href="https://www.myvideohunter.com/prod/url/abc123">`)
	assert.Contains(t, body, `<meta property="og:url" content="https://www.myvideohunter.com/prod/url/abc123" />`)
	assert.Contains(t, body, `<h1 class="h3 mb-1">Download X (Twitter) video</h1>`)

	// Structured data.
	assert.Contains(t, body, `application/ld+json`)
	assert.Contains(t, body, `"@type":"WebPage"`)

	// Analytics must record page views.
	assert.NotContains(t, body, "send_page_view")

	// No fake freshness text.
	assert.NotContains(t, body, "Last updated 3 mins ago")

	// HLS playlist is not offered as a direct download.
	assert.NotContains(t, body, "Download: .m3u8")
	assert.Contains(t, body, "Download: 640x360")

	// No ad units rendered while the AdSense slot IDs are not configured.
	assert.NotContains(t, body, `<ins class="adsbygoogle"`)

	// Internal link to the matching platform landing page.
	assert.Contains(t, body, `href="/x-video-downloader.html"`)

	// The nav matches the site's, so the video pages are not a dead end.
	assert.Contains(t, body, `href="/reddit-video-downloader.html"`)
	assert.Contains(t, body, `href="/bluesky-video-downloader.html"`)
	assert.Contains(t, body, `href="/telegram-bot.html"`)

	// Caching headers.
	assert.Equal(t, "public, max-age=3600, s-maxage=3600", response.Headers["Cache-Control"])
	assert.Equal(t, "text/html; charset=utf-8", response.Headers["Content-Type"])
}

func TestGetUrlHandle_Handle_RendersErrorPageWhenVideoIsNotFound(t *testing.T) {

	handler := &GetUrlHandler{GerUrlUseCase: stubGetUrlUseCase{video: nil}}

	response, err := handler.Handle(events.APIGatewayProxyRequest{
		PathParameters: map[string]string{"id": "missing"},
	})

	assert.NoError(t, err)
	assert.Equal(t, 404, response.StatusCode)

	body := response.Body

	assert.True(t, strings.Contains(body, "Video not found"))
	assert.Contains(t, body, `<form action="/" method="get"`)
	assert.Contains(t, body, `<meta name="robots" content="noindex, follow">`)
	assert.NotContains(t, body, "send_page_view")
	assert.Equal(t, "no-store", response.Headers["Cache-Control"])
}

func TestGetUrlHandle_Handle_Returns500WhenUseCaseFails(t *testing.T) {

	handler := &GetUrlHandler{GerUrlUseCase: stubGetUrlUseCase{err: errors.New("dynamodb unavailable")}}

	response, err := handler.Handle(events.APIGatewayProxyRequest{
		PathParameters: map[string]string{"id": "abc123"},
	})

	assert.NoError(t, err)
	assert.Equal(t, 500, response.StatusCode)
	assert.Contains(t, response.Body, "Something went wrong")
}
