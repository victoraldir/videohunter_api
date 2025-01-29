package events

import (
	"strings"
)

const (
	amplifyVideoIdx = 7
	extTwVideoIdx   = 8
	domainIdx       = 3
	redditDomainIdx = 2
	avcIdx          = 7
)

type (
	CreateVideoResponse struct {
		Id           string `json:"id"`
		OriginalId   string `json:"original_id"`
		ThumbnailUrl string `json:"thumbnail_url"`
		Description  string `json:"description"`
		Uri          string `json:"uri"`
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

	DownloadVideoHlsResponse struct {
		VideoPath string `json:"video_path"`
	}
)

func (v *VideoResponseVariant) GetVidResFromUrl() string {

	splittedUrl := strings.Split(v.URL, "/")

	if splittedUrl[redditDomainIdx] == "v.redd.it" || splittedUrl[redditDomainIdx] == "video.bsky.app" {
		return "full quality"
	}

	if len(splittedUrl) == 7 {
		mediaType := strings.Split(splittedUrl[len(splittedUrl)-1], "?")[0]

		if strings.Contains(mediaType, "m3u8") {
			return ".m3u8"
		}
	}

	if splittedUrl[domainIdx] == "ext_tw_video" {

		if splittedUrl[avcIdx] == "avc1" {
			return splittedUrl[extTwVideoIdx]
		}

		return splittedUrl[extTwVideoIdx-1]
	}

	return splittedUrl[amplifyVideoIdx]
}

func (v *GetVideoResponse) IsTwitter() bool {
	urlSplitted := strings.Split(v.OriginalVideoUrl, "/")
	return urlSplitted[redditDomainIdx] == "x.com" ||
		urlSplitted[redditDomainIdx] == "twitter.com" ||
		urlSplitted[redditDomainIdx] == "www.twitter.com"
}
