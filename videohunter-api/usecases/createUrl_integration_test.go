//go:build integration
// +build integration

package usecases

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	"github.com/victoraldir/myvideohunterapi/adapters/twitter"
	"github.com/victoraldir/myvideohunterapi/utils"
	"go.uber.org/mock/gomock"
)

func TestVideoDownloaderUseCase_Execute_Integration(t *testing.T) {

	realHttpClient := &http.Client{}

	t.Run("Should download video from twitter", func(t *testing.T) {

		setup(t)
		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

		// Arrange
		videoUrl := "https://x.com/PicturesFoIder/status/1745002642089349387?s=20"

		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

		videoDownloaderUseCase := NewVideoDownloaderUseCase(
			videoRepository,
			downloadeRepository,
			settingsRepository,
		)

		// Act
		video, err := videoDownloaderUseCase.Execute(videoUrl)

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, video)

	})

	t.Run("Should not download video from twitter. Not video midia", func(t *testing.T) {

		setup(t)
		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

		// Arrange
		videoUrl := "https://twitter.com/victoraldir/status/1736141224891822316"

		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

		videoDownloaderUseCase := NewVideoDownloaderUseCase(
			videoRepository,
			downloadeRepository,
			settingsRepository,
		)

		// Act
		video, err := videoDownloaderUseCase.Execute(videoUrl)

		// Assert
		assert.NotNil(t, err)
		assert.ErrorContains(t, err, "no video found")
		assert.Nil(t, video)
	})

	t.Run("Should not download video from twitter. Page doesn't exist", func(t *testing.T) {

		setup(t)
		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()

		// Arrange
		videoUrl := "https://twitter.com/samplerandompage/status/2627763231482538584"

		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

		videoDownloaderUseCase := NewVideoDownloaderUseCase(
			videoRepository,
			downloadeRepository,
			settingsRepository,
		)

		// Act
		video, err := videoDownloaderUseCase.Execute(videoUrl)

		// Assert
		assert.NotNil(t, err)
		assert.ErrorContains(t, err, "no video found")
		assert.Nil(t, video)
	})

	t.Run("Should md5 hash the original video url", func(t *testing.T) {

		setup(t)
		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()
		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()

		// Arrange
		videoUrl := "https://x.com/historyinmemes/status/1746260828704157829?s=20"
		expectedMd5Hash := utils.GenerateShortID(videoUrl)

		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		downloadeRepository := twitter.NewTwitterDownloaderRepository(realHttpClient)

		videoDownloaderUseCase := NewVideoDownloaderUseCase(
			videoRepository,
			downloadeRepository,
			settingsRepository,
		)

		// Act
		video, err := videoDownloaderUseCase.Execute(videoUrl)

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, video)
		assert.Equal(t, expectedMd5Hash, video.Id)
		assert.NotEmpty(t, video.ThumbnailUrl)
		assert.NotEmpty(t, video.Description)
	})

}
