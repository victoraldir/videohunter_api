package twitter

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTwitterService(t *testing.T) {

	url := "https://x.com/analise2025/status/1953923802703737304"
	httpClient := &http.Client{}

	twitterService := NewTwitterDownloaderRepository(httpClient)

	video, _, err := twitterService.DownloadVideo(url)

	assert.NotNil(t, video)
	assert.Nil(t, err)

}
