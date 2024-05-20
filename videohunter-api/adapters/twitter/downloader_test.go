package twitter

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTwitterDownloaderRepository_DownloadVideo(t *testing.T) {

	// Arrange
	url := "https://twitter.com/gunsnrosesgirl3/status/1791770018243367237"
	httpClient := &http.Client{}
	repo := NewTwitterDownloaderRepository(httpClient)

	// Act
	video, _, err := repo.DownloadVideo(url)

	// Assert
	assert.Nil(t, err)
	assert.NotNil(t, video)
	assert.Equal(t, "https://pbs.twimg.com/amplify_video_thumb/1791768540497788928/img/Yzb1F-T6In9eI6EW.jpg", video.ThumbnailUrl)

}
