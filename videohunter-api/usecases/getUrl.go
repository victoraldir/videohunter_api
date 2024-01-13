package usecases

import (
	"log/slog"

	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohunterapi/repositories"
)

//go:generate mockgen -destination=../usecases/mocks/mockVideoDownloaderUseCase.go -package=usecases github.com/victoraldir/myvideohunterapi/usecases VideoDownloaderUseCase
type GetUrlUseCase interface {
	Execute(videoId string) (*events.GetVideoResponse, error)
}

type getUrlUseCase struct {
	VideoRepository repositories.VideoRepository
}

func NewGetUrlUseCase(videoRepository repositories.VideoRepository) *getUrlUseCase {
	return &getUrlUseCase{
		VideoRepository: videoRepository,
	}
}

func (v *getUrlUseCase) Execute(videoId string) (*events.GetVideoResponse, error) {

	video, err := v.VideoRepository.GetVideo(videoId)

	if err != nil {
		return nil, err
	}

	videoReponse := videoToGetVideoResponse(video)

	return videoReponse, nil
}

func videoToGetVideoResponse(video *domain.Video) *events.GetVideoResponse {

	slog.Debug("Parsing video to GetVideoResponse", "video", video)

	videoResponse := &events.GetVideoResponse{}
	videoResponse.Id = video.IdDB
	videoResponse.Text = video.Text
	videoResponse.OriginalVideoUrl = video.OriginalVideoUrl
	videoResponse.CreatedAt = video.CreatedAt
	videoResponse.Variants = make([]events.VideoResponseVariant, len(video.ExtendedEntities.Media[0].VideoInfo.Variants))

	for _, media := range video.ExtendedEntities.Media {

		videoResponse.ThumbnailUrl = media.MediaUrl // TODO - this might lead to problems if there are more than one media. I have to check this.

		for idx, variant := range media.VideoInfo.Variants {
			videoResponse.Variants[idx].Bitrate = variant.Bitrate
			videoResponse.Variants[idx].URL = variant.URL
			videoResponse.Variants[idx].ContentType = variant.ContentType
		}
	}

	return videoResponse
}
