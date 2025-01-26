package twitter

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTwitterService(t *testing.T) {

	url := "https://x.com/enfuisback/status/1882542726845223422"
	httpClient := &http.Client{}

	twitterService := NewTwitterDownloaderRepository(httpClient)

	video, _, err := twitterService.DownloadVideo(url)

	assert.NotNil(t, video)
	assert.Nil(t, err)

}
