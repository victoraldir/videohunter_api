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
		video, _, err := redditDownloaderRepository.DownloadVideo("https://www.reddit.com/r/allinspanish/comments/1ic5bms/la_am%C3%A9rica_de_trump_en_menos_de_2_minutos/")

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, video)
	})
}
