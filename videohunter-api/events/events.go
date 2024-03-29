package events

import "strings"

const (
	amplifyVideoIdx = 7
	extTwVideoIdx   = 8
	domainIdx       = 3
	avcIdx          = 7
)

type (
	CreateVideoResponse struct {
		Id           string `json:"id"`
		ThumbnailUrl string `json:"thumbnail_url"`
		Description  string `json:"description"`
	}

	GetVideoResponse struct {
		Id               string                 `json:"id"`
		ThumbnailUrl     string                 `json:"thumbnail_url"`
		Text             string                 `json:"text"`
		CreatedAt        string                 `json:"created_at"`
		OriginalVideoUrl string                 `json:"original_video_url"`
		Variants         []VideoResponseVariant `json:"variants"`
	}

	VideoResponseVariant struct {
		Bitrate     int    `json:"bitrate"`
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
	}
)

func (v *VideoResponseVariant) GetVidResFromUrl() string {

	splittedUrl := strings.Split(v.URL, "/")

	if splittedUrl[domainIdx] == "ext_tw_video" {

		if splittedUrl[avcIdx] == "avc1" {
			return splittedUrl[extTwVideoIdx]
		}

		return splittedUrl[extTwVideoIdx-1]
	}

	return splittedUrl[amplifyVideoIdx]
}
