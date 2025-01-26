package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXxx(t *testing.T) {

	url := "https://x.com/enfuisback/status/1882542726845223422?s=46"

	videoId := GetVideoId(url)

	assert.Equal(t, "1882542726845223422", videoId)

}
