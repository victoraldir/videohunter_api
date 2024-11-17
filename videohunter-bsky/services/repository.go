package services

import "github.com/victoraldir/myvideohunterbsky/domain"

type PlatformRepository interface {
	Login() error
	SearchPostsByMention(mention string) ([]domain.Post, error)
	EnrichPost(posts []domain.Post) ([]domain.Post, error)
	GetPostsByUris(uris []string) ([]domain.Post, error)
	// ReplyUser(post Post, downloadLink string) error
}

type VideoDownloaderRepository interface {
	DownloadVideo(video domain.Video) error
}
