package reddit

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedditDownloaderRepository_DownloadVideo(t *testing.T) {
	t.Run("Should download video from reddit", func(t *testing.T) {
		// Arrange
		url := "https://www.reddit.com/r/Unexpected/comments/1cznts8/learned_his_lesson"
		httpClient := &http.Client{}
		repo := NewRedditDownloaderRepository(httpClient)

		// Act
		video, _, err := repo.DownloadVideo(url)

		// Assert
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		assert.NotNil(t, video)
		assert.Equal(t, "https://v.redd.it/b4cikpfnw80d1/DASH_96.mp4", video.ThumbnailUrl)
	})
}
