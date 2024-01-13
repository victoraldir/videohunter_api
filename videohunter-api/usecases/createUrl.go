package usecases

import (
	"log/slog"

	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohunterapi/repositories"
	"github.com/victoraldir/myvideohunterapi/utils"
)

//go:generate mockgen -destination=../usecases/mocks/mockVideoDownloaderUseCase.go -package=usecases github.com/victoraldir/myvideohunterapi/usecases VideoDownloaderUseCase
type VideoDownloaderUseCase interface {
	Execute(url string) (*events.CreateVideoResponse, error)
}

type videoDownloaderUseCase struct {
	VideoRepository    repositories.VideoRepository
	DownloadRepository repositories.DownloadRepository
	SettingsRepository repositories.SettingsRepository
}

func NewVideoDownloaderUseCase(videoRepository repositories.VideoRepository,
	downloadRepository repositories.DownloadRepository,
	settingsRepository repositories.SettingsRepository) *videoDownloaderUseCase {
	return &videoDownloaderUseCase{
		VideoRepository:    videoRepository,
		DownloadRepository: downloadRepository,
		SettingsRepository: settingsRepository,
	}
}

func (v *videoDownloaderUseCase) Execute(url string) (*events.CreateVideoResponse, error) {

	url = utils.NormalizeVideoUrl(url)

	videoId := utils.GenerateShortID(url)

	var err error

	slog.Debug("checking if video %v already exists", "videoId", videoId)
	existingVideo, err := v.VideoRepository.GetVideo(videoId)

	if err != nil {
		return nil, err
	}

	if existingVideo != nil {
		slog.Debug("video %v already exists", "videoId", videoId)
		return videoToCreateVideoResponse(existingVideo), nil
	}

	authToken, err := v.SettingsRepository.GetSetting(domain.KeySetting(domain.AuthToken))

	if err != nil {
		return nil, err
	}

	slog.Debug("video %v does not exist. Downloading...", "videoId", videoId)

	var newVideo *domain.Video
	var currentToken *string

	if authToken != nil {
		slog.Debug("authToken found. Downloading with it...", "authToken", authToken.Value)
		newVideo, _, err = v.DownloadRepository.DownloadVideo(url, authToken.Value)
	} else {
		slog.Debug("authToken not found. Downloading without it...")
		newVideo, currentToken, err = v.DownloadRepository.DownloadVideo(url)

		if err != nil {
			return nil, err
		}

		slog.Debug("saving authToken %v", "authToken", *currentToken)
		v.SettingsRepository.SaveSetting(&domain.Settings{
			KeySetting: domain.AuthToken,
			Value:      *currentToken,
		})
	}

	if err != nil {
		if err.Error() == "status code error: 401 401 Unauthorized" {
			slog.Debug("authToken expired. Downloading again...")
			newVideoIn, currentToken, err := v.DownloadRepository.DownloadVideo(url)
			newVideo = newVideoIn

			if err != nil {
				return nil, err
			}

			slog.Debug("saving authToken %v", "authToken", *currentToken)
			v.SettingsRepository.SaveSetting(&domain.Settings{
				KeySetting: domain.AuthToken,
				Value:      *currentToken,
			})

		} else {
			return nil, err
		}
	}

	newVideo.OriginalVideoUrl = url
	videoDb, err := v.VideoRepository.SaveVideo(newVideo)

	if err != nil {
		return nil, err
	}

	return videoToCreateVideoResponse(videoDb), nil
}

func videoToCreateVideoResponse(video *domain.Video) *events.CreateVideoResponse {

	slog.Debug("Parsing video to VideoResponse", "video", video)

	videoResponse := &events.CreateVideoResponse{}
	videoResponse.Id = video.IdDB
	return videoResponse
}
