package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	events_aws "github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	api_events "github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohuntershared/services"
	"github.com/victoraldir/myvideohuntershared/services/reddit"
)

type stubVideoDownloader struct {
	response *api_events.CreateVideoResponse
	err      error
}

func (s stubVideoDownloader) Execute(url string) (*api_events.CreateVideoResponse, error) {
	return s.response, s.err
}

func (s stubVideoDownloader) DownloadVideo(url, videoId string, repo services.DownloadRepository, useAuthToken bool) (*api_events.CreateVideoResponse, error) {
	return s.response, s.err
}

func createUrlRequest(t *testing.T, body string) events_aws.APIGatewayProxyRequest {
	t.Helper()
	return events_aws.APIGatewayProxyRequest{Body: body}
}

func messageOf(t *testing.T, body string) string {
	t.Helper()
	var parsed map[string]string
	assert.NoError(t, json.Unmarshal([]byte(body), &parsed))
	return parsed["message"]
}

func TestCreateUrlHandler_Handle(t *testing.T) {

	t.Run("returns the created video for a supported link", func(t *testing.T) {
		handler := CreateUrlHandler{
			VideoDownloaderUseCase: stubVideoDownloader{
				response: &api_events.CreateVideoResponse{Id: "abc123", OriginalId: "1", ThumbnailUrl: "https://thumb"},
			},
		}

		response, err := handler.Handle(createUrlRequest(t, `{"video_url":"https://www.reddit.com/r/videos/comments/1abcxyz/test/"}`))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, response.Body, `"id":"abc123"`)
		assert.Equal(t, "application/json", response.Headers["Content-Type"])
	})

	t.Run("rejects a link from an unsupported platform", func(t *testing.T) {
		handler := CreateUrlHandler{VideoDownloaderUseCase: stubVideoDownloader{}}

		response, err := handler.Handle(createUrlRequest(t, `{"video_url":"https://example.com/video/1"}`))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		assert.Contains(t, messageOf(t, response.Body), "not supported")
	})

	t.Run("rejects a malformed request body", func(t *testing.T) {
		handler := CreateUrlHandler{VideoDownloaderUseCase: stubVideoDownloader{}}

		response, err := handler.Handle(createUrlRequest(t, `not json`))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	})

	t.Run("reports a post without a video as not found, not as a bad link", func(t *testing.T) {
		handler := CreateUrlHandler{
			VideoDownloaderUseCase: stubVideoDownloader{
				err: &reddit.InvalidPostError{StatusCode: 404, Err: errors.New("post is not a video")},
			},
		}

		response, err := handler.Handle(createUrlRequest(t, `{"video_url":"https://www.reddit.com/r/videos/comments/1abcxyz/test/"}`))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		assert.Contains(t, messageOf(t, response.Body), "no downloadable video")
	})

	t.Run("reports an unreachable platform as a bad gateway", func(t *testing.T) {
		handler := CreateUrlHandler{
			VideoDownloaderUseCase: stubVideoDownloader{
				err: &reddit.UpstreamError{StatusCode: 429, Err: errors.New("reddit returned status 429")},
			},
		}

		response, err := handler.Handle(createUrlRequest(t, `{"video_url":"https://www.reddit.com/r/videos/comments/1abcxyz/test/"}`))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadGateway, response.StatusCode)
		assert.Contains(t, messageOf(t, response.Body), "not responding right now")
	})

	t.Run("reports a wrapped invalid post error the same way", func(t *testing.T) {
		handler := CreateUrlHandler{
			VideoDownloaderUseCase: stubVideoDownloader{
				err: errors.Join(errors.New("context"), &reddit.InvalidPostError{StatusCode: 404, Err: errors.New("invalid post")}),
			},
		}

		response, err := handler.Handle(createUrlRequest(t, `{"video_url":"https://www.reddit.com/r/videos/comments/1abcxyz/test/"}`))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
	})

	t.Run("falls back to a server error for anything else", func(t *testing.T) {
		handler := CreateUrlHandler{
			VideoDownloaderUseCase: stubVideoDownloader{err: errors.New("dynamodb unavailable")},
		}

		response, err := handler.Handle(createUrlRequest(t, `{"video_url":"https://bsky.app/profile/bsky.app/post/3lxxo3i4qzs2c"}`))

		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		assert.Contains(t, messageOf(t, response.Body), "Something went wrong")
	})
}
