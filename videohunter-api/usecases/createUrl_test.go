package usecases

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	dynamodb_mock "github.com/victoraldir/myvideohunterapi/adapters/dynamodb/mocks"
	"github.com/victoraldir/myvideohuntershared/services/bsky"
	"github.com/victoraldir/myvideohuntershared/services/reddit"
	"github.com/victoraldir/myvideohuntershared/services/twitter"
	"go.uber.org/mock/gomock"
)

var dynamodDBClientMock *dynamodb_mock.MockDynamodDBClient

func setup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dynamodDBClientMock = dynamodb_mock.NewMockDynamodDBClient(ctrl)
}

func TestVideoDownloaderUseCase_Execute_Integration(t *testing.T) {

	realHttpClient := &http.Client{}

	t.Run("Should download video from twitter", func(t *testing.T) {

		setup(t)
		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

		// Arrange
		videoUrl := "https://x.com/enfuisback/status/1882542726845223422?s=46"

		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)
		bskyRepo := bsky.NewBskyService(realHttpClient, "", "")
		redditRepo := reddit.NewRedditDownloaderRepository(realHttpClient)

		videoDownloaderUseCase := NewVideoDownloaderUseCase(
			videoRepository,
			downloadeRepository,
			redditRepo,
			bskyRepo,
			settingsRepository,
		)

		// Act
		video, err := videoDownloaderUseCase.Execute(videoUrl)

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, video)

	})

	// t.Run("Should not download video from twitter. Not video midia", func(t *testing.T) {

	// 	setup(t)
	// 	dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
	// 	dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

	// 	// Arrange
	// 	videoUrl := "https://twitter.com/victoraldir/status/1736141224891822316"

	// 	videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
	// 	settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
	// 	downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

	// 	videoDownloaderUseCase := NewVideoDownloaderUseCase(
	// 		videoRepository,
	// 		downloadeRepository,
	// 		settingsRepository,
	// 	)

	// 	// Act
	// 	video, err := videoDownloaderUseCase.Execute(videoUrl)

	// 	// Assert
	// 	assert.NotNil(t, err)
	// 	assert.ErrorContains(t, err, "no video found")
	// 	assert.Nil(t, video)
	// })

	// t.Run("Should not download video from twitter. Page doesn't exist", func(t *testing.T) {

	// 	setup(t)
	// 	dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()

	// 	// Arrange
	// 	videoUrl := "https://twitter.com/samplerandompage/status/2627763231482538584"

	// 	videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
	// 	settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
	// 	downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

	// 	videoDownloaderUseCase := NewVideoDownloaderUseCase(
	// 		videoRepository,
	// 		downloadeRepository,
	// 		settingsRepository,
	// 	)

	// 	// Act
	// 	video, err := videoDownloaderUseCase.Execute(videoUrl)

	// 	// Assert
	// 	assert.NotNil(t, err)
	// 	assert.ErrorContains(t, err, "no video found")
	// 	assert.Nil(t, video)
	// })

	// t.Run("Should md5 hash the original video url", func(t *testing.T) {

	// 	setup(t)
	// 	dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()
	// 	dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()

	// 	// Arrange
	// 	videoUrl := "https://x.com/historyinmemes/status/1746260828704157829?s=20"
	// 	expectedMd5Hash := utils.GenerateShortID(videoUrl)

	// 	videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
	// 	settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
	// 	downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

	// 	videoDownloaderUseCase := NewVideoDownloaderUseCase(
	// 		videoRepository,
	// 		downloadeRepository,
	// 		settingsRepository,
	// 	)

	// 	// Act
	// 	video, err := videoDownloaderUseCase.Execute(videoUrl)

	// 	// Assert
	// 	assert.Nil(t, err)
	// 	assert.NotNil(t, video)
	// 	assert.Equal(t, expectedMd5Hash, video.Id)
	// 	assert.NotEmpty(t, video.ThumbnailUrl)
	// 	assert.NotEmpty(t, video.Description)
	// })

	// t.Run("Should download video from reddit", func(t *testing.T) {

	// 	// Arrange
	// 	setup(t)

	// 	videoUrl := "https://www.reddit.com/r/nextfuckinglevel/comments/1gdclo5/the_class_above_first_exclusive_to_singapore/"

	// 	dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()
	// 	dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
	// 	videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
	// 	redditDownloadRepo := reddit.NewRedditDownloaderRepository(realHttpClient)

	// 	redditVideoDownloaderUseCase := NewRedditVideoDownloaderUseCase(
	// 		videoRepository,
	// 		redditDownloadRepo,
	// 	)

	// 	video, err := redditVideoDownloaderUseCase.Execute(videoUrl)

	// 	// Assert
	// 	assert.Nil(t, err)
	// 	assert.NotNil(t, video)

	// })

	t.Run("Should download video from bsky", func(t *testing.T) {

		// Arrange
		setup(t)

		videoUrl := "https://bsky.app/profile/spookieshelbie.bsky.social/post/3legshbjons2h"

		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()
		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		twitterRepo := twitter.NewTwitterDownloaderRepository(realHttpClient)
		bskyRepo := bsky.NewBskyService(realHttpClient, "", "")
		redditRepo := reddit.NewRedditDownloaderRepository(realHttpClient)
		settingRepo := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")

		videoDownloaderUseCase := NewVideoDownloaderUseCase(
			videoRepository,
			twitterRepo,
			redditRepo,
			bskyRepo,
			settingRepo,
		)

		video, err := videoDownloaderUseCase.Execute(videoUrl)

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, video)

	})
}
