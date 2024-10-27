package usecases

import (
	"log/slog"

	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohunterapi/repositories"
)

type CreateUrlBatchUseCase interface {
	Execute(uris []string) ([]events.CreateVideoResponse, error)
}

type createUrlBatchUseCase struct {
	socialNetworkRepository repositories.SocialNetworkRepository
	videoRepository         repositories.VideoRepository
}

func NewCreateUrlBatchUseCase(socialNetworkRepository repositories.SocialNetworkRepository,
	videoRepository repositories.VideoRepository) CreateUrlBatchUseCase {
	return &createUrlBatchUseCase{
		socialNetworkRepository: socialNetworkRepository,
		videoRepository:         videoRepository,
	}
}

func (u *createUrlBatchUseCase) Execute(uris []string) ([]events.CreateVideoResponse, error) {

	var responses []events.CreateVideoResponse
	var videos []domain.Video
	var remainingVideos []string

	// Find videos already in the database
	for _, uri := range uris {
		video, err := u.videoRepository.GetVideo(uri)
		if err != nil {
			slog.Error("Error getting video", slog.Any("error", err))
			continue
		}

		if video == nil {
			remainingVideos = append(remainingVideos, uri)
			continue
		}

		videos = append(videos, *video)
	}

	if len(remainingVideos) > 0 {
		// Fetch video from api
		videosApi, err := u.socialNetworkRepository.GetPostsByUris(remainingVideos)
		if err != nil {
			return nil, err
		}

		// Save videos to database
		for _, video := range videosApi {
			videoDb, err := u.videoRepository.SaveVideo(&domain.Video{
				OriginalVideoUrl: video.Uri,
				ThumbnailUrl:     video.Embed.Thumbnail,
				Text:             video.Record.Text,
				CreatedAt:        video.Record.CreatedAt,
				ExtendedEntities: domain.ExtendedEntities{
					Media: []domain.Media{
						{
							Type:     "video",
							MediaUrl: video.Embed.Playlist,
							VideoInfo: domain.VideoInfo{
								Variants: []domain.Variants{
									{
										URL:         video.Embed.Playlist,
										ContentType: "video/mp4",
									},
								},
							},
						},
					},
				},
			})
			if err != nil {
				slog.Error("Error saving video", slog.Any("error", err))
				continue
			}

			videos = append(videos, *videoDb)
		}
	}

	for _, video := range videos {
		responses = append(responses, events.CreateVideoResponse{
			Id:           video.IdDB,
			Description:  video.Text,
			ThumbnailUrl: video.ThumbnailUrl,
			Uri:          video.OriginalVideoUrl,
		})
	}

	return responses, nil

}
