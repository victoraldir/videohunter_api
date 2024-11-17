package bsky

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	userName = "myvideohunter.com"
	password = "bla"
)

func TestBskyService_SearchPostsByMention(t *testing.T) {

	// Arrange
	httpClient := &http.Client{}
	bskyService := NewBskyService(httpClient, userName, password)
	bskyService.Login()

	t.Run("Should search posts by mention", func(t *testing.T) {

		// Act
		posts, err := bskyService.SearchPostsByMention("@myvideohunter.com", "2024-11-16T19:03:53Z", "2024-11-16T19:05:55Z")

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, posts)
	})
}

func TestBskyService_GetPostsByUris(t *testing.T) {

	// Arrange
	httpClient := &http.Client{}
	bskyService := NewBskyService(httpClient, userName, password)
	bskyService.Login()

	t.Run("Should get posts by uris", func(t *testing.T) {

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

	// Arrange
	httpClient := &http.Client{}
	bskyService := NewBskyService(httpClient, userName, password)
	bskyService.Login()

	t.Run("Should enrich post", func(t *testing.T) {

		// Arrange
		posts, err := bskyService.SearchPostsByMention("@myvideohunter.com", "2024-11-16T19:03:53Z", "2024-11-16T19:05:55Z")
		assert.Nil(t, err)

		// Act
		err = bskyService.EnrichPost(&posts)

		// Assert
		assert.Nil(t, err)

		assert.NotNil(t, posts)
	})
}

func TestBskyService_Reply(t *testing.T) {

	// Arrange
	httpClient := &http.Client{}
	bskyService := NewBskyService(httpClient, userName, password)
	bskyService.Login()

	t.Run("Should reply", func(t *testing.T) {
		// Arrange
		posts, err := bskyService.SearchPostsByMention("@myvideohunter.com", "2024-11-16T19:03:53Z", "2024-11-16T19:05:55Z")
		assert.Nil(t, err)

		// Enrich posts
		bskyService.EnrichPost(&posts)

		// Act
		for i := 0; i < len(posts); i++ {
			post := (posts)[i]

			if post.Url != nil {
				err = bskyService.Reply(post)
				if err != nil {
					slog.Debug("Error replying", slog.Any("error", err))
				}
			}
		}

		// Assert
		assert.Nil(t, err)

		assert.NotNil(t, posts)
	})
}

func TestBskyService_Login(t *testing.T) {

	// Arrange
	httpClient := &http.Client{}
	bskyService := NewBskyService(httpClient, userName, password)

	t.Run("Should login", func(t *testing.T) {

		// Act
		session, err := bskyService.Login()

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, session)
	})

	t.Run("Should not login", func(t *testing.T) {
		// Arrange
		httpClient := &http.Client{}
		bskyService := NewBskyService(httpClient, userName, "123")

		// Act
		session, err := bskyService.Login()

		// Assert
		assert.Nil(t, session)
		assert.NotNil(t, err)
	})
}