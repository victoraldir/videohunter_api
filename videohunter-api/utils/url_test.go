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

func TestIsRedditUrl(t *testing.T) {

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"www permalink", "https://www.reddit.com/r/videos/comments/1abcxyz/test/", true},
		{"apex permalink", "https://reddit.com/r/videos/comments/1abcxyz/test/", true},
		{"old reddit", "https://old.reddit.com/r/videos/comments/1abcxyz/test/", true},
		{"new reddit", "https://new.reddit.com/r/videos/comments/1abcxyz/test/", true},
		{"share link", "https://www.reddit.com/r/allinspanish/s/utvRHjBxoM", true},
		{"uppercase host", "https://WWW.REDDIT.COM/r/videos/comments/1abcxyz/test/", true},
		{"no path", "https://www.reddit.com", false},
		{"subreddit only", "https://www.reddit.com/r/videos", false},
		{"http not allowed", "http://www.reddit.com/r/videos/comments/1abcxyz/test/", false},
		{"lookalike host", "https://notreddit.com/r/videos/comments/1abcxyz/test/", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRedditUrl(tt.url))
		})
	}
}

func TestIsTwitterUrl(t *testing.T) {

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"x.com", "https://x.com/user/status/1", true},
		{"twitter.com", "https://twitter.com/user/status/1", true},
		{"www.twitter.com", "https://www.twitter.com/user/status/1", true},
		{"http", "http://x.com/user/status/1", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsTwitterUrl(tt.url))
		})
	}
}

func TestIsBskyUrl(t *testing.T) {

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"post", "https://bsky.app/profile/bsky.app/post/3lxxo3i4qzs2c", true},
		{"profile only", "https://bsky.app/profile/bsky.app", true},
		{"other host", "https://example.com/profile/a/post/1", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsBskyUrl(tt.url))
		})
	}
}
