package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVideoResponseVariant_GetVidResFromUrl(t *testing.T) {

	// Arrange
	url := "https://video.twimg.com/amplify_video/1745002590725910528/vid/avc1/320x428/ae-XO9iTYOCGAkyE.mp4?tag=14"
	videoResponseVariant := VideoResponseVariant{
		URL:         url,
		Bitrate:     0,
		ContentType: "test",
	}

	// Act
	vidRes := videoResponseVariant.GetVidResFromUrl()

	// Assert
	assert.Equal(t, "320x428", vidRes, "they should be equal")
}

func TestVideoResponseVariant_GetVidResFromUrl_WithBitrate(t *testing.T) {

	// Arrange
	url := "https://video.twimg.com/ext_tw_video/1744425187032944640/pu/vid/avc1/480x852/98cZjnvvxCpG1EOH.mp4?tag=12"
	videoResponseVariant := VideoResponseVariant{
		URL:         url,
		Bitrate:     832000,
		ContentType: "test",
	}

	// Act
	vidRes := videoResponseVariant.GetVidResFromUrl()

	// Assert
	assert.Equal(t, "480x852", vidRes, "they should be equal")
}
