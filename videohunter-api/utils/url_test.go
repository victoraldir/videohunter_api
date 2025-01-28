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

func TestParseUri(t *testing.T) {

	uri := "at://did:plc:dqis4e26lvohwpjdvayhdb4p/app.bsky.feed.post/3l5sa5yv6we2v"

	url := AtUriToUrl(uri)

	assert.Equal(t, "https://bsky.app/profile/did:plc:dqis4e26lvohwpjdvayhdb4p/post/3l5sa5yv6we2v", url)

}
