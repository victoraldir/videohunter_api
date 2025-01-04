package repositories

import (
	"github.com/victoraldir/myvideohunterapi/domain"
)

type VideoRepository interface {
	SaveVideo(video *domain.Video) (*domain.Video, error)
	GetVideo(videoId string) (*domain.Video, error)
}

type SettingsRepository interface {
	SaveSetting(setting *domain.Settings) (*domain.Settings, error)
	GetSetting(settingName domain.KeySetting) (*domain.Settings, error)
}

type DownloadHlsRepository interface {
	DownloadHls(url string) (videoDownload *domain.Video, err error)
	MixAudioAndVideo(videoUrl, audioUrl string) (videoDownload *domain.Video, err error)
}
