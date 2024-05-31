package usecases

import (
	"github.com/victoraldir/myvideohunterapi/events"
	"github.com/victoraldir/myvideohunterapi/repositories"
)

type MixAudioVideoUseCase interface {
	Execute(videoUrl, audioUrl string) (*events.DownloadVideoHlsResponse, error)
}

type mixAudioVideoUseCase struct {
	DownloadHlsRepository repositories.DownloadHlsRepository
}

func NewMixAudioVideoUseCase(downloadHlsRepository repositories.DownloadHlsRepository) *mixAudioVideoUseCase {
	return &mixAudioVideoUseCase{
		DownloadHlsRepository: downloadHlsRepository,
	}
}

func (v *mixAudioVideoUseCase) Execute(videoUrl, audioUrl string) (*events.DownloadVideoHlsResponse, error) {

	videoDownload, err := v.DownloadHlsRepository.MixAudioAndVideo(videoUrl, audioUrl)

	if err != nil {
		return nil, err
	}

	videoReponse := events.DownloadVideoHlsResponse{
		VideoPath: videoDownload.Path,
	}

	return &videoReponse, nil
}
