package usecases

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"

	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohunterapi/repositories"
	"github.com/victoraldir/myvideohunterapi/utils"
	"github.com/victoraldir/myvideohuntershared/services"
)

//go:generate mockgen -destination=../usecases/mocks/mockVideoDownloaderUseCase.go -package=usecases github.com/victoraldir/myvideohunterapi/usecases VideoDownloaderUseCase
type VideoDownloaderUseCase interface {
	Execute(url string) (*events.CreateVideoResponse, error)
	DownloadVideo(url, videoId string, repo services.DownloadRepository, useAuthToken bool) (*events.CreateVideoResponse, error)
}

type videoDownloaderUseCase struct {
	VideoRepository          repositories.VideoRepository
	TwitterRepository        services.DownloadRepository
	SettingsRepository       repositories.SettingsRepository
	RedditDownloadRepository services.DownloadRepository
	BskyDownloadRepository   services.DownloadRepository
}

func NewVideoDownloaderUseCase(videoRepository repositories.VideoRepository,
	twitterRepository services.DownloadRepository,
	RedditDownloadRepository services.DownloadRepository,
	BskyDownloadRepository services.DownloadRepository,
	settingsRepository repositories.SettingsRepository) *videoDownloaderUseCase {
	return &videoDownloaderUseCase{
		VideoRepository:          videoRepository,
		TwitterRepository:        twitterRepository,
		RedditDownloadRepository: RedditDownloadRepository,
		BskyDownloadRepository:   BskyDownloadRepository,
		SettingsRepository:       settingsRepository,
	}
}

func (v *videoDownloaderUseCase) Execute(url string) (*events.CreateVideoResponse, error) {

	if utils.IsTwitterUrl(url) {
		url = utils.NormalizeVideoUrl(url)
	}

	videoId := utils.GenerateShortID(url)

	slog.Debug("checking if video %v already exists", "videoId", videoId)
	existingVideo, err := v.VideoRepository.GetVideo(videoId)
	if err != nil {
		return nil, err
	}

	if existingVideo != nil {
		slog.Debug("video %v already exists", "videoId", videoId)
		return videoToCreateVideoResponse(existingVideo), nil
	}

	var createVideoResponse *events.CreateVideoResponse

	if utils.IsTwitterUrl(url) {
		createVideoResponse, err = v.DownloadVideo(url, videoId, v.TwitterRepository, true)
	} else if utils.IsBskyUrl(url) {
		createVideoResponse, err = v.DownloadVideo(url, videoId, v.BskyDownloadRepository, false)
	} else if utils.IsRedditUrl(url) {
		createVideoResponse, err = v.DownloadVideo(url, videoId, v.RedditDownloadRepository, false)
	}

	if err != nil {
		return nil, err
	}

	return createVideoResponse, nil
}

func videoToCreateVideoResponse(video *domain.Video) *events.CreateVideoResponse {

	videoResponse := &events.CreateVideoResponse{}
	videoResponse.OriginalId = video.OriginalId
	videoResponse.Id = video.IdDB
	videoResponse.Description = video.Text
	videoResponse.ThumbnailUrl = video.ThumbnailUrl
	videoResponse.Uri = fmt.Sprintf("/url/%s", video.IdDB)

	slog.Debug("videoToCreateVideoResponse", "videoResponse", videoResponse)

	return videoResponse
}

func (v *videoDownloaderUseCase) DownloadVideo(url, videoId string, repo services.DownloadRepository, useAuthToken bool) (*events.CreateVideoResponse, error) {
	var newVideo *domain.Video
	var err error

	if useAuthToken {
		authToken, err := v.SettingsRepository.GetSetting(domain.KeySetting(domain.AuthToken))
		if err != nil {
			return nil, err
		}

		if authToken != nil {
			log.Println("authToken found. Downloading with it...", "authToken", authToken.Value)
			videoApi, _, err := repo.DownloadVideo(url, authToken.Value)
			if err != nil {
				return nil, err
			}

			deepCopy(&videoApi, &newVideo)
		} else {
			log.Println("authToken not found. Downloading without it...")
			videoApi, currentToken, err := repo.DownloadVideo(url)
			if err != nil {
				return nil, err
			}

			deepCopy(&videoApi, &newVideo)

			log.Println("saving authToken", "authToken", *currentToken)
			v.SettingsRepository.SaveSetting(&domain.Settings{
				KeySetting: domain.AuthToken,
				Value:      *currentToken,
			})
		}

		if err != nil {
			if err.Error() == "status code error: 401 401 Unauthorized" {
				log.Println("authToken expired. Downloading again...")
				videoApi, currentToken, err := repo.DownloadVideo(url)
				if err != nil {
					return nil, err
				}

				deepCopy(&videoApi, &newVideo)

				log.Println("saving authToken", "authToken", *currentToken)
				v.SettingsRepository.SaveSetting(&domain.Settings{
					KeySetting: domain.AuthToken,
					Value:      *currentToken,
				})
			} else {
				return nil, err
			}
		}
	} else {
		videoApi, _, err := repo.DownloadVideo(url)
		if err != nil {
			return nil, err
		}

		deepCopy(&videoApi, &newVideo)
	}

	newVideo.OriginalVideoUrl = url
	videoDb, err := v.VideoRepository.SaveVideo(newVideo)
	if err != nil {
		return nil, err
	}

	return videoToCreateVideoResponse(videoDb), nil
}

func deepCopy(src, dst interface{}) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dst)
}
