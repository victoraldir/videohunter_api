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

func TestVideoResponseVariant_GetVidResFromUrl_WithoutAvc(t *testing.T) {

	// Arrange
	url := "https://video.twimg.com/ext_tw_video/1656656738727362561/pu/vid/320x568/SvOetDa460v4voY8.mp4?tag=12"
	videoResponseVariant := VideoResponseVariant{
		URL:         url,
		Bitrate:     832000,
		ContentType: "test",
	}

	// Act
	vidRes := videoResponseVariant.GetVidResFromUrl()

	// Assert
	assert.Equal(t, "320x568", vidRes, "they should be equal")
}

func TestVideoResponseVariant_GetVidResFromUrl_RedditUrl(t *testing.T) {

	// Arrange
	url := "https://v.redd.it/b4cikpfnw80d1/HLSPlaylist.m3u8?a=1718309730%2CNDI2YjVhYTA4NTUxZmZjNjFmNjA2NDg2Y2QyOTI5MzVhZmViMmJjMTIwOWYwM2M4MGQyZjUzNDgzMDQyODIwZg%3D%3D&amp;v=1&amp;f=sd"
	videoResponseVariant := VideoResponseVariant{
		URL:         url,
		Bitrate:     832000,
		ContentType: "test",
	}

	// Act
	vidRes := videoResponseVariant.GetVidResFromUrl()

	// Assert
	assert.Equal(t, "full quality", vidRes, "they should be equal")

}

func TestGetVideoVariant_GetRes_M3u8(t *testing.T) {

	// Arrange
	url := "https://video.twimg.com/amplify_video/1884339231641759744/pl/zLqjBBxSX9Fsa0xG.m3u8?tag=16"
	videoResponseVariant := VideoResponseVariant{
		URL:         url,
		Bitrate:     832000,
		ContentType: "test",
	}

	// Act
	vidRes := videoResponseVariant.GetVidResFromUrl()

	// Assert
	assert.Equal(t, ".m3u8", vidRes, "they should be equal")

}
