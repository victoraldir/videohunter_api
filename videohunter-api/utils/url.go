package utils

import (
	"encoding/base64"
	"net/url"
	"strings"
)

const (
	urlVideoSeparator = "/"
)

func IsTwitterUrl(twitterUrl string) bool {

	if twitterUrl == "" {
		return false
	}

	url, err := url.Parse(twitterUrl)

	if err != nil {
		return false
	}

	if url.Host != "twitter.com" { // Make it better
		if url.Host != "www.twitter.com" {
			if url.Host != "x.com" {
				if url.Host != "www.x.com" {
					return false
				}
			}
		}
	}

	if url.Scheme != "https" {
		return false
	}

	if url.Path == "" {
		return false
	}

	return true

}

func GetVideoId(twitterUrl string) string {

	urlSplit := strings.Split(twitterUrl, urlVideoSeparator)

	videoId := strings.Split(twitterUrl, urlVideoSeparator)[len(urlSplit)-1]

	return videoId
}

func GenerateShortID(inputString string) string {
	encodedBytes := base64.URLEncoding.EncodeToString([]byte(inputString))
	return encodedBytes
}

func NormalizeVideoUrl(videoUrl string) string {

	url, _ := url.Parse(videoUrl)

	url.Host = "twitter.com"
	// Clear query params
	url.RawQuery = ""

	return url.String()
}
