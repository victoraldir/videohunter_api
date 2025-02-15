package domain

import "github.com/victoraldir/myvideohuntershared/domain"

type KeySetting string

const (
	AuthToken string = "auth_token"
)

type Settings struct {
	KeySetting string `json:"key"`
	Value      string `json:"value"`
}

type VideoList []struct {
	domain.Video
}

type Video struct {
	IdDB             string           `json:"id_db"`
	OriginalId       string           `json:"original_id"`
	OriginalVideoUrl string           `json:"original_video_url"`
	ThumbnailUrl     string           `json:"media_url_https"`
	CreatedAt        string           `json:"created_at"`
	ExtendedEntities ExtendedEntities `json:"extended_entities"`
	Text             string           `json:"full_text"`
	QuotedStatus     Status           `json:"quoted_status"`
	RetweetedStatus  Status           `json:"retweeted_status"`
	Path             string           `json:"path"`
}

type Status struct {
	ExtendedEntities ExtendedEntities `json:"extended_entities"`
	Text             string           `json:"full_text"`
}

type ExtendedEntities struct {
	Media []Media `json:"media"`
}

type Media struct {
	VideoInfo VideoInfo `json:"video_info"`
	MediaUrl  string    `json:"media_url_https"`
	Type      string    `json:"type"`
}

type VideoInfo struct {
	Variants []Variants `json:"variants"`
}

type Variants struct {
	Bitrate     int    `json:"bitrate"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
}

func (v Video) GetText() string {

	if v.ExtendedEntities.Media != nil && v.ExtendedEntities.Media[0].Type == "video" {
		return v.Text
	}

	if v.QuotedStatus.ExtendedEntities.Media != nil && v.QuotedStatus.ExtendedEntities.Media[0].Type == "video" {
		return v.QuotedStatus.Text
	}

	if v.RetweetedStatus.ExtendedEntities.Media != nil && v.RetweetedStatus.ExtendedEntities.Media[0].Type == "video" {
		return v.RetweetedStatus.Text
	}

	return ""
}

func (v Video) GetMedia() Media {

	if v.ExtendedEntities.Media != nil && v.ExtendedEntities.Media[0].Type == "video" {
		return v.ExtendedEntities.Media[0]
	}

	if v.QuotedStatus.ExtendedEntities.Media != nil && v.QuotedStatus.ExtendedEntities.Media[0].Type == "video" {
		return v.QuotedStatus.ExtendedEntities.Media[0]
	}

	if v.RetweetedStatus.ExtendedEntities.Media != nil && v.RetweetedStatus.ExtendedEntities.Media[0].Type == "video" {
		return v.RetweetedStatus.ExtendedEntities.Media[0]
	}

	return Media{}
}
