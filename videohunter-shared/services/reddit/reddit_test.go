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
		// +18 https://www.reddit.com/r/Cuckold/comments/1hp1th1/my_husband_didnt_know_but_after_this_time_with_my/
		video, _, err := redditDownloaderRepository.DownloadVideo("https://www.reddit.com/r/RoastMe/comments/1hoonbn/27f_unemployed_divorced_2_cats/")

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, video)
	})
}
