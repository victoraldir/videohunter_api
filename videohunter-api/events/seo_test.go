package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVideoResponse_Platform(t *testing.T) {

	tests := []struct {
		name string
		url  string
		want string
	}{
		{"x.com", "https://x.com/eeugee_/status/2105038695577661785", "X (Twitter)"},
		{"twitter.com", "https://twitter.com/elonmusk/status/1392602041025846272", "X (Twitter)"},
		{"www.twitter.com", "https://www.twitter.com/elonmusk/status/1392602041025846272", "X (Twitter)"},
		{"reddit", "https://www.reddit.com/r/videos/comments/1abcxyz/test/", "Reddit"},
		{"reddit video", "https://v.redd.it/b4cikpfnw80d1", "Reddit"},
		{"bsky", "https://bsky.app/profile/did:plc:tgummumeffrrfbb3ujf6vt6r/post/3lp3smwmmns2x", "Bluesky"},
		{"bsky at uri", "at://did:plc:6wthaiuqiy3y7eztkpsdam2/app.bsky.feed.post/3lazkrxnpbs26", "Bluesky"},
		{"unknown", "https://example.com/video/1", "social media"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video := &GetVideoResponse{OriginalVideoUrl: tt.url}
			assert.Equal(t, tt.want, video.Platform())
		})
	}
}

func TestGetVideoResponse_PageTitle(t *testing.T) {

	t.Run("should include platform and video text", func(t *testing.T) {
		video := &GetVideoResponse{
			Text:             "Activistas de la acampada en Sol expulsan a provocadores",
			OriginalVideoUrl: "https://twitter.com/eeugee_/status/2105038695577661785",
		}

		assert.Equal(t, "Download X (Twitter) video: Activistas de la acampada en Sol expulsan a provocadores — Video Hunter", video.PageTitle())
	})

	t.Run("should fall back to a generic title when text is empty", func(t *testing.T) {
		video := &GetVideoResponse{OriginalVideoUrl: "https://bsky.app/profile/x/post/1"}

		assert.Equal(t, "Download Bluesky video online — Video Hunter", video.PageTitle())
	})

	t.Run("should collapse whitespace and truncate long text", func(t *testing.T) {
		video := &GetVideoResponse{
			Text:             "line one\n\n   line two that goes on and on and on and on and on and on and on and on",
			OriginalVideoUrl: "https://x.com/a/status/1",
		}

		title := video.PageTitle()

		assert.NotContains(t, title, "\n")
		assert.LessOrEqual(t, len([]rune(title)), 120)
		assert.Contains(t, title, "…")
	})
}

func TestGetVideoResponse_PageDescription(t *testing.T) {

	t.Run("should include video text and call to action", func(t *testing.T) {
		video := &GetVideoResponse{
			Text:             "A short description",
			OriginalVideoUrl: "https://twitter.com/a/status/1",
		}

		assert.Equal(t, "A short description Download it free in HD with Video Hunter.", video.PageDescription())
	})

	t.Run("should stay within meta description length", func(t *testing.T) {
		video := &GetVideoResponse{
			Text:             "This is a very long video description that should definitely be truncated because it keeps going and going and going and going and going and going and going and going",
			OriginalVideoUrl: "https://reddit.com/r/videos/comments/1abcxyz/test/",
		}

		assert.LessOrEqual(t, len([]rune(video.PageDescription())), maxDescription)
	})

	t.Run("should fall back to a generic description when text is empty", func(t *testing.T) {
		video := &GetVideoResponse{OriginalVideoUrl: "https://reddit.com/r/videos/comments/1abcxyz/test/"}

		assert.Contains(t, video.PageDescription(), "Download this Reddit video online for free.")
	})
}

func TestGetVideoResponse_CanonicalURL(t *testing.T) {

	video := &GetVideoResponse{Id: "aHR0cHM6Ly90d2l0dGVyLmNvbS9lZXVnZWVfL3N0YXR1cy8yMTA1MDM4Njk1NTc3NjYxNzg1"}

	assert.Equal(t,
		"https://www.myvideohunter.com/prod/url/aHR0cHM6Ly90d2l0dGVyLmNvbS9lZXVnZWVfL3N0YXR1cy8yMTA1MDM4Njk1NTc3NjYxNzg1",
		video.CanonicalURL())
}
