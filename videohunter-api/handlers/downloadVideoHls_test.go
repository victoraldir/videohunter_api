package handlers

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	events "github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api_events "github.com/victoraldir/myvideohunterapi/events"
)

func TestHandle(t *testing.T) {
	t.Run("Should unmarshal request body", func(t *testing.T) {

		// Arrage
		data := `{ \"url\": \"https://v.redd.it/b4cikpfnw80d1/HLSPlaylist.m3u8\\?a\\=1719161822%2CYWZkNDY2Mjg2NGUxNGMyMDRiOTExZGEzYWFkYjJjNTE1MDQxYjVjMTY5NDE2MjU4OThjNjU4ZTM4MDhjM2JlMQ%3D%3D\\&amp\\;v\\=1\\&amp\\;f\\=sd\"}`
		downloadRequest := &DownalodRequest{}

		//Act

		// Unscape the string
		data = strings.Replace(data, "\\\"", "\"", -1)

		err := json.Unmarshal([]byte(data), downloadRequest)

		// Assert
		assert.Nil(t, err)

	})
}

type stubDownloadVideoHlsUseCase struct {
	videoPath string
}

func (s stubDownloadVideoHlsUseCase) Execute(url string) (*api_events.DownloadVideoHlsResponse, error) {
	return &api_events.DownloadVideoHlsResponse{VideoPath: s.videoPath}, nil
}

func downloadVideoRequest(query map[string]string) *events.LambdaFunctionURLRequest {
	return &events.LambdaFunctionURLRequest{QueryStringParameters: query}
}

func writeVideoFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "video.mp4")
	require.NoError(t, os.WriteFile(path, []byte("fake video bytes"), 0o600))
	return path
}

func TestDownloadVideoHlsHandler_Handle(t *testing.T) {

	handler := func(t *testing.T) *DownloadVideoHlsHandler {
		return NewDownloadVideoHlsHandler(stubDownloadVideoHlsUseCase{videoPath: writeVideoFile(t)})
	}

	t.Run("serves the video as an attachment by default", func(t *testing.T) {
		response, err := handler(t).Handle(downloadVideoRequest(map[string]string{
			"url": "https://video.twimg.com/amplify_video/1234/pu/vid/720x1280/CNbyVdA7KdXLUQkM.mp4?tag=21",
		}))

		require.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
		assert.Equal(t, "video/mp4", response.Headers["Content-Type"])
		// A forced download stays the default: only the video page's iPhone
		// and iPad branch asks for inline playback.
		assert.Equal(t, `attachment; filename="videohunter-CNbyVdA7KdXLUQkM.mp4"`,
			response.Headers["Content-Disposition"])

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		assert.Equal(t, "fake video bytes", string(body))
	})

	t.Run("serves the video inline when the page asks for it", func(t *testing.T) {
		response, err := handler(t).Handle(downloadVideoRequest(map[string]string{
			"url":    "https://video.twimg.com/amplify_video/1234/pu/vid/720x1280/CNbyVdA7KdXLUQkM.mp4?tag=21",
			"inline": "1",
		}))

		require.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
		assert.Equal(t, `inline; filename="videohunter-CNbyVdA7KdXLUQkM.mp4"`,
			response.Headers["Content-Disposition"])
	})
}

func TestVideoFilename(t *testing.T) {

	t.Run("uses the base name of the video url", func(t *testing.T) {
		assert.Equal(t, "videohunter-CNbyVdA7KdXLUQkM.mp4",
			videoFilename("https://video.twimg.com/amplify_video/1/pu/vid/720x1280/CNbyVdA7KdXLUQkM.mp4?tag=21"))
	})

	t.Run("drops characters that are not safe in a file name", func(t *testing.T) {
		assert.Equal(t, "videohunter-abcd-12_3.mp4",
			videoFilename("https://cdn.example.com/videos/a b:c;*d-12_3.mp4"))
	})

	t.Run("keeps long base names short", func(t *testing.T) {
		assert.Equal(t, "videohunter-"+string([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))+".mp4",
			videoFilename("https://cdn.example.com/videos/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp4"))
	})

	t.Run("falls back when there is nothing to take from the url", func(t *testing.T) {
		assert.Equal(t, "videohunter.mp4", videoFilename("https://cdn.example.com"))
		assert.Equal(t, "videohunter.mp4", videoFilename(""))
	})
}
