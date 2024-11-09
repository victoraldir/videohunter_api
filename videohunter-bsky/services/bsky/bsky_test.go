package bsky

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBskyService_SearchPostsByMention(t *testing.T) {

	t.Run("Should search posts by mention", func(t *testing.T) {

		// Arrange
		httpClient := &http.Client{}

		bskyService := NewBskyService(httpClient)

		// Act
		posts, err := bskyService.SearchPostsByMention("@myvideohunter.com", "2024-10-05T21:36:29.181Z")

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, posts)
	})
}

func TestBskyService_GetPostsByUris(t *testing.T) {
	t.Run("Should get posts by uris", func(t *testing.T) {
		// Arrange
		httpClient := &http.Client{}
		bskyService := NewBskyService(httpClient)

		// Act
		postsByUris, err := bskyService.GetPostsByUris([]string{
			"at://did:plc:3fibociwu7jy4bbdjhmm4nop/app.bsky.feed.post/3l5fgldunxk2y",
			"at://did:plc:aca4rpd2skm56qugeb6o4fua/app.bsky.feed.post/3l5nhkzz62d2k",
		})

		// Assert
		assert.Nil(t, err)

		assert.NotNil(t, postsByUris)

	})
}

func TestBskyService_EnrichPost(t *testing.T) {
	t.Run("Should enrich post", func(t *testing.T) {
		// Arrange
		httpClient := &http.Client{}
		bskyService := NewBskyService(httpClient)
		posts, err := bskyService.SearchPostsByMention("@myvideohunter.com", "2024-10-05T21:36:29.181Z")
		assert.Nil(t, err)

		// Act
		err = bskyService.EnrichPost(&posts)

		// Assert
		assert.Nil(t, err)

		assert.NotNil(t, posts)
	})
}

func TestBskyService_Reply(t *testing.T) {
	t.Run("Should reply", func(t *testing.T) {
		// Arrange
		httpClient := &http.Client{}
		bskyService := NewBskyService(httpClient)
		posts, err := bskyService.SearchPostsByMention("@myvideohunter.com", "2024-10-05T21:36:29.181Z")
		assert.Nil(t, err)

		// Enrich posts
		bskyService.EnrichPost(&posts)

		// Act
		for i := 0; i < len(posts); i++ {
			post := (posts)[i]

			if post.Url != nil {
				err = bskyService.Reply(post)
				break
			}
		}

		// Assert
		assert.Nil(t, err)

		assert.NotNil(t, posts)
	})
}
