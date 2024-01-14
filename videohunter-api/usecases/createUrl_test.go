package usecases

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/victoraldir/myvideohunterapi/adapters/dynamodb"
	dynamodb_mock "github.com/victoraldir/myvideohunterapi/adapters/dynamodb/mocks"
	"github.com/victoraldir/myvideohunterapi/adapters/twitter"
	"github.com/victoraldir/myvideohunterapi/utils"
	"go.uber.org/mock/gomock"
)

var dynamodDBClientMock *dynamodb_mock.MockDynamodDBClient

func setup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dynamodDBClientMock = dynamodb_mock.NewMockDynamodDBClient(ctrl)
}

func TestVideoDownloaderUseCase_Execute(t *testing.T) {

	t.Run("Should download video from twitter", func(t *testing.T) {

		setup(t)
		dynamodDBClientMock.EXPECT().GetItem(gomock.Any()).Return(nil, nil).AnyTimes()
		dynamodDBClientMock.EXPECT().PutItem(gomock.Any()).Return(nil, nil).AnyTimes()

		// Arrange
		videoUrl := "https://x.com/PicturesFoIder/status/1745002642089349387?s=20"

		httpClient := &http.Client{}

		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		downloadeRepository := twitter.NewTwitterDownloaderRepository(httpClient)

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

		httpClient := &http.Client{}

		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		downloadeRepository := twitter.NewTwitterDownloaderRepository(httpClient)

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

		httpClient := &http.Client{}

		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		downloadeRepository := twitter.NewTwitterDownloaderRepository(httpClient)

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
		videoUrl := "https://twitter.com/victoraldir/status/1746565606705426501"
		expectedMd5Hash := utils.GenerateShortID(videoUrl)

		httpClient := &http.Client{}

		videoRepository := dynamodb.NewDynamodbVideoRepository(dynamodDBClientMock, "video")
		settingsRepository := dynamodb.NewDynamoSettingsRepository(dynamodDBClientMock, "settings")
		downloadeRepository := twitter.NewTwitterDownloaderRepository(httpClient)

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
