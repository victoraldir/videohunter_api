package usecases

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	"github.com/victoraldir/myvideohuntershared/services/bsky"
	"go.uber.org/mock/gomock"
)

func TestCreateUrlBatchUseCase_Execute(t *testing.T) {
	t.Run("Should create a batch of urls", func(t *testing.T) {
		// Arrange
		setup(t)
		urls := []string{"at://did:plc:aca4rpd2skm56qugeb6o4fua/app.bsky.feed.post/3l5nhkzz62d2k",
			"",
			"at://did:plc:gy6awl66beidwfv5uq7vnmlo/app.bsky.feed.post/3latnslmcl22a"}

		httpClient := http.Client{}

		socialNetworkRepository := bsky.NewBskyService(&httpClient, "", "")
		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")

		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

		createUrlBatchUseCase := NewCreateUrlBatchUseCase(socialNetworkRepository, videoRepository)

		// Act
		videos, err := createUrlBatchUseCase.Execute(urls)

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, videos)
	})
}
