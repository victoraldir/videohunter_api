package reddit

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedditDownloaderRepository_DownloadVideo(t *testing.T) {
	t.Run("Should test something", func(t *testing.T) {
		// Arrange
		httpClient := &http.Client{}
		redditDownloaderRepository := NewRedditDownloaderRepository(httpClient)

		// Act
		video, _, err := redditDownloaderRepository.DownloadVideo("https://www.reddit.com/r/CrazyFuckingVideos/comments/1htn0a4/crazy_road_rage/")

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, video)
	})
}
