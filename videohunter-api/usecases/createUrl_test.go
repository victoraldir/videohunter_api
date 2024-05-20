package usecases

import (
	"testing"

	dynamodb_mock "github.com/victoraldir/myvideohunterapi/adapters/dynamodb/mocks"
	"go.uber.org/mock/gomock"
)

var dynamodDBClientMock *dynamodb_mock.MockDynamodDBClient

func setup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dynamodDBClientMock = dynamodb_mock.NewMockDynamodDBClient(ctrl)
}

// func TestVideoDownloaderUseCase_Execute_Integration(t *testing.T) {

// 	realHttpClient := &http.Client{}

// 	t.Run("Should download video from twitter", func(t *testing.T) {

// 		setup(t)
// 		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
// 		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

// 		// Arrange
// 		videoUrl := "https://twitter.com/gunsnrosesgirl3/status/1792166453849858364"

// 		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
// 		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
// 		downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

// 		videoDownloaderUseCase := NewVideoDownloaderUseCase(
// 			videoRepository,
// 			downloadeRepository,
// 			settingsRepository,
// 		)

// 		// Act
// 		video, err := videoDownloaderUseCase.Execute(videoUrl)

// 		// Assert
// 		assert.Nil(t, err)
// 		assert.NotNil(t, video)

// 	})

// 	t.Run("Should not download video from twitter. Not video midia", func(t *testing.T) {

// 		setup(t)
// 		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
// 		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

// 		// Arrange
// 		videoUrl := "https://twitter.com/victoraldir/status/1736141224891822316"

// 		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
// 		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
// 		downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

// 		videoDownloaderUseCase := NewVideoDownloaderUseCase(
// 			videoRepository,
// 			downloadeRepository,
// 			settingsRepository,
// 		)

// 		// Act
// 		video, err := videoDownloaderUseCase.Execute(videoUrl)

// 		// Assert
// 		assert.NotNil(t, err)
// 		assert.ErrorContains(t, err, "no video found")
// 		assert.Nil(t, video)
// 	})

// 	t.Run("Should not download video from twitter. Page doesn't exist", func(t *testing.T) {

// 		setup(t)
// 		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()

// 		// Arrange
// 		videoUrl := "https://twitter.com/samplerandompage/status/2627763231482538584"

// 		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
// 		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
// 		downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

// 		videoDownloaderUseCase := NewVideoDownloaderUseCase(
// 			videoRepository,
// 			downloadeRepository,
// 			settingsRepository,
// 		)

// 		// Act
// 		video, err := videoDownloaderUseCase.Execute(videoUrl)

// 		// Assert
// 		assert.NotNil(t, err)
// 		assert.ErrorContains(t, err, "no video found")
// 		assert.Nil(t, video)
// 	})

// 	t.Run("Should md5 hash the original video url", func(t *testing.T) {

// 		setup(t)
// 		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()
// 		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()

// 		// Arrange
// 		videoUrl := "https://x.com/historyinmemes/status/1746260828704157829?s=20"
// 		expectedMd5Hash := utils.GenerateShortID(videoUrl)

// 		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
// 		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
// 		downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

// 		videoDownloaderUseCase := NewVideoDownloaderUseCase(
// 			videoRepository,
// 			downloadeRepository,
// 			settingsRepository,
// 		)

// 		// Act
// 		video, err := videoDownloaderUseCase.Execute(videoUrl)

// 		// Assert
// 		assert.Nil(t, err)
// 		assert.NotNil(t, video)
// 		assert.Equal(t, expectedMd5Hash, video.Id)
// 		assert.NotEmpty(t, video.ThumbnailUrl)
// 		assert.NotEmpty(t, video.Description)
// 	})

// 	t.Run("Should download video from reddit", func(t *testing.T) {

// 		// Arrange
// 		setup(t)

// 		videoUrl := "https://www.reddit.com/r/2latinoforyou/comments/1cr82fl/pa%C3%ADses_con_m%C3%A1s_homicidios_del_mundo"

// 		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()
// 		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
// 		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
// 		redditDownloadRepo := reddit.NewRedditDownloaderRepository(realHttpClient)

// 		redditVideoDownloaderUseCase := NewRedditVideoDownloaderUseCase(
// 			videoRepository,
// 			redditDownloadRepo,
// 		)

// 		video, err := redditVideoDownloaderUseCase.Execute(videoUrl)

// 		// Assert
// 		assert.Nil(t, err)
// 		assert.NotNil(t, video)

// 	})
// }
