package usecases

import (
	"log/slog"

	"github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohunterapi/repositories"
)

type DownloadVideoHlsUseCase interface {
	Execute(url string) (*events.DownloadVideoHlsResponse, error)
}

type downloadVideoHlsUseCase struct {
	VideoRepository       repositories.VideoRepository
	DownloadHlsRepository repositories.DownloadHlsRepository
}

func NewDownloadVideoHlsUseCase(videoRepository repositories.VideoRepository, downloadHlsRepository repositories.DownloadHlsRepository) *downloadVideoHlsUseCase {
	return &downloadVideoHlsUseCase{
		VideoRepository:       videoRepository,
		DownloadHlsRepository: downloadHlsRepository,
	}
}

func (v *downloadVideoHlsUseCase) Execute(url string) (*events.DownloadVideoHlsResponse, error) {

	videoDownload, err := v.DownloadHlsRepository.DownloadHls(url)
	slog.Debug("downloadVideoHlsUseCase excute() DownloadHls():", "videoDownload", videoDownload)

	if err != nil {
		return nil, err
	}

	videoReponse := events.DownloadVideoHlsResponse{
		VideoPath: videoDownload.Path,
	}

	return &videoReponse, nil
}
