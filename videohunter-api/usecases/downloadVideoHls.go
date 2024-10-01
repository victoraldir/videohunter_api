package usecases

import (
	"log"

	"github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohunterapi/repositories"
)

type DownloadVideoHlsUseCase interface {
	Execute(url string) (*events.DownloadVideoHlsResponse, error)
}

type downloadVideoHlsUseCase struct {
	DownloadHlsRepository repositories.DownloadHlsRepository
}

func NewDownloadVideoHlsUseCase(videoRepository repositories.VideoRepository, downloadHlsRepository repositories.DownloadHlsRepository) *downloadVideoHlsUseCase {
	return &downloadVideoHlsUseCase{
		DownloadHlsRepository: downloadHlsRepository,
	}
}

func (v *downloadVideoHlsUseCase) Execute(url string) (*events.DownloadVideoHlsResponse, error) {

	videoDownload, err := v.DownloadHlsRepository.DownloadHls(url)
	log.Println("downloadVideoHlsUseCase excute() DownloadHls():", "videoDownload", videoDownload)

	if err != nil {
		return nil, err
	}

	videoReponse := events.DownloadVideoHlsResponse{
		VideoPath: videoDownload.Path,
	}

	return &videoReponse, nil
}
