package dynamodb

import (
	"testing"

	"github.com/victoraldir/myvideohunterbsky/domain"
)

func TestDynamodbRepository_SaveLastPostScanTime(t *testing.T) {
	t.Run("Should save last post scan time", func(t *testing.T) {
		// Arrange
		dynamodbRepository := NewDynamodbRepository()

		// Act
		err := dynamodbRepository.SaveLastPostScanTime("2024-10-05T21:36:29.181Z")

		// Assert
		if err != nil {
			t.Error("Error saving last post scan time")
		}
	})
}

func TestDynamodbRepository_GetLastPostScanTime(t *testing.T) {
	t.Run("Should get last post scan time", func(t *testing.T) {
		// Arrange
		dynamodbRepository := NewDynamodbRepository()
		dynamodbRepository.SaveLastPostScanTime("2024-10-05T21:36:29.181Z")

		// Act
		lastPostScanTime, err := dynamodbRepository.GetLastPostScanTime()

		// Assert
		if err != nil {
			t.Error("Error getting last post scan time")
		}

		if lastPostScanTime == "" {
			t.Error("Last post scan time not found")
		}
	})
}

func TestDynamodbRepository_SavePost(t *testing.T) {
	t.Run("Should save post", func(t *testing.T) {
		// Arrange
		dynamodbRepository := NewDynamodbRepository()
		post := domain.Post{
			Cid: "at://did:plc:3fibociwu7jy4bbdjhmm4nop/app.bsky.feed.post/3l5fgldunxk2y",
		}

		// Act
		err := dynamodbRepository.SavePost(post)

		// Assert
		if err != nil {
			t.Error("Error saving post")
		}
	})
}

func TestDynamodbRepository_GetPostByCid(t *testing.T) {
	t.Run("Should get post by cid", func(t *testing.T) {
		// Arrange
		dynamodbRepository := NewDynamodbRepository()
		post := domain.Post{
			Cid: "at://did:plc:3fibociwu7jy4bbdjhmm4nop/app.bsky.feed.post/3l5fgldunxk2y",
		}
		dynamodbRepository.SavePost(post)

		// Act
		postByCid, err := dynamodbRepository.GetPostByCid("at://did:plc:3fibociwu7jy4bbdjhmm4nop/app.bsky.feed.post/3l5fgldunxk2y")

		// Assert
		if err != nil {
			t.Error("Error getting post by cid")
		}

		if postByCid.Cid != post.Cid {
			t.Error("Post not found")
		}
	})
}
