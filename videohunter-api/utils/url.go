package utils

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

const (
	urlVideoSeparator = "/"
)

func IsRedditUrl(redditUrl string) bool {

	if redditUrl == "" {
		return false
	}

	url, err := url.Parse(redditUrl)

	if err != nil {
		return false
	}

	if url.Host != "www.reddit.com" {
		return false
	}

	if url.Scheme != "https" {
		return false
	}

	if url.Path == "" {
		return false
	}

	return true
}

func IsBskyUrl(bskyUrl string) bool {

	if bskyUrl == "" {
		return false
	}

	url, err := url.Parse(bskyUrl)

	if err != nil {
		return false
	}

	if url.Host != "bsky.app" {
		return false
	}

	if url.Scheme != "https" {
		return false
	}

	if url.Path == "" {
		return false
	}

	return true
}

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

func IsBlueskyUrl(blueskyUrl string) bool {

	if blueskyUrl == "" {
		return false
	}

	url, err := url.Parse(blueskyUrl)

	if err != nil {
		return false
	}

	if url.Host != "bsky.app" {
		return false
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

	// Remove query params
	videoId = strings.Split(videoId, "?")[0]

	return videoId
}

func GetVideoIdReddit(redditUrl string) string {

	urlSplit := strings.Split(redditUrl, urlVideoSeparator)

	videoId := strings.Split(redditUrl, urlVideoSeparator)[len(urlSplit)-1]

	return videoId

}

func GenerateShortID(inputString string) string {

	inputString = NormalizeVideoUrl(inputString)

	return Base64Encode(inputString)
}

func Base64Encode(inputString string) string {
	encodedBytes := base64.URLEncoding.EncodeToString([]byte(inputString))
	return encodedBytes
}

func Base64Decode(encodedString string) (string, error) {
	decodedBytes, err := base64.URLEncoding.DecodeString(encodedString)
	if err != nil {
		return "", err
	}
	return string(decodedBytes), nil
}

func NormalizeVideoUrl(videoUrl string) string {

	url, _ := url.Parse(videoUrl)

	url.Host = "twitter.com"
	// Clear query params
	url.RawQuery = ""

	return url.String()
}

func UrlToUriAt(url string) string {

	//https://bsky.app/profile/fun-viral-vids.bsky.social/post/
	// at://fun-viral-vids.bsky.social/app.bsky.feed.post/3ldnrtdet3c2e

	urlSplit := strings.Split(url, urlVideoSeparator)

	uriAt := "at://" + urlSplit[4] + "/app.bsky.feed.post/" + urlSplit[len(urlSplit)-1]

	return uriAt

}

func DeepCopy(src, dst interface{}) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dst)
}

func AtUriToUrl(uri string) string {

	// https://bsky.app/profile/<DID>/post/<RKEY>

	uriSplit := strings.Split(uri, "/")

	url := "https://bsky.app/profile/" + uriSplit[2] + "/post/" + uriSplit[4]

	return url

}
