package bsky

// import (
// 	"net/http"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// func TestBskyDownloaderRepository_GetPostsByUris(t *testing.T) {
// 	t.Run("Test GetPostsByUris", func(t *testing.T) {
// 		// Arrange
// 		uris := []string{"at://did:plc:dqis4e26lvohwpjdvayhdb4p/app.bsky.feed.post/3l6co33iff32p", "at://did:plc:aca4rpd2skm56qugeb6o4fua/app.bsky.feed.post/3l5nhkzz62d2k"}
// 		client := http.Client{}
// 		repo := NewBskyDownloaderRepository(&client)

// 		// Act
// 		posts, err := repo.GetPostsByUris(uris)

// 		// Assert
// 		assert.Nil(t, err)
// 		assert.NotNil(t, posts)

// 	})
// }
