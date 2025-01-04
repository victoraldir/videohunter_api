package videohunterapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVideoHunterApi_DownloadVideo(t *testing.T) {
	t.Run("Should download video", func(t *testing.T) {
		// Arrange
		httpClient := &http.Client{}

		videoHunterApi := NewVideoHunterApi(httpClient)

		// Act
		videoUrl, err := videoHunterApi.DownloadVideo("https://video.bsky.app/watch/did%3Aplc%3Adqis4e26lvohwpjdvayhdb4p/bafkreieube3vpdfgshbfw7nck5pzgcfnk3dyulemmphraxryngrvlwxi4a/playlist.m3u8")

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, videoUrl)
	})
}
