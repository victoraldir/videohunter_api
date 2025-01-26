package usecases

import (
	"fmt"
	"log/slog"

	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohunterapi/repositories"
	"github.com/victoraldir/myvideohuntershared/services/bsky"
)

type CreateUrlBatchUseCase interface {
	Execute(uris []string) ([]events.CreateVideoResponse, error)
}

type createUrlBatchUseCase struct {
	socialNetworkRepository bsky.BskyService
	videoRepository         repositories.VideoRepository
}

func NewCreateUrlBatchUseCase(socialNetworkRepository bsky.BskyService,
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

	// If uris is greater than 25 return error
	if len(uris) > 25 {
		return nil, fmt.Errorf("maximum of 25 uris allowed")
	}

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
		postsApi, err := u.socialNetworkRepository.GetPostsByUrisAPI(remainingVideos)
		if err != nil {
			return nil, err
		}

		// Save videos to database
		for _, video := range postsApi {
			videoDb, err := u.videoRepository.SaveVideo(&domain.Video{
				OriginalVideoUrl: video.Uri,
				OriginalId:       video.Uri,
				ThumbnailUrl:     video.Embed.Thumbnail,
				Text:             video.Record.Text,
				CreatedAt:        video.Record.CreatedAt,
				ExtendedEntities: domain.ExtendedEntities{
					Media: []domain.Media{
						{
							Type:     "video",
							MediaUrl: video.Uri,
							VideoInfo: domain.VideoInfo{
								Variants: []domain.Variants{
									{
										URL:         video.Uri,
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
			OriginalId:   video.OriginalId,
			Description:  video.Text,
			ThumbnailUrl: video.ThumbnailUrl,
			Uri:          fmt.Sprint("/url/", video.IdDB),
		})
	}

	return responses, nil

}
