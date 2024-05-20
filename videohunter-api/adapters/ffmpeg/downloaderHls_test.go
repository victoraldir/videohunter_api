package ffmpeg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloaderHlsRepository(t *testing.T) {

	// Arrange
	url := "https://v.redd.it/b4cikpfnw80d1/HLSPlaylist.m3u8?a=1718310735%2COGE5ZDY5NmE0MzY3NmQyM2UzMTNkNTJkZmMxMmRhNzg4MmM2MzQzNTczYzY0YTYzOGFjMzQwNWQ4ZTViN2I0Zg%3D%3D&amp;v=1&amp;f=sd"
	downloaderHlsRepository := NewDownloaderHlsRepository()

	// Act
	video, err := downloaderHlsRepository.DownloadHls(url)

	// Assert
	assert.Nil(t, err)
	assert.NotNil(t, video)
}
