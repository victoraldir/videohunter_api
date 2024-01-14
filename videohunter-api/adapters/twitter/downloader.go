package twitter

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/victoraldir/myvideohunterapi/domain"
	"github.com/victoraldir/myvideohunterapi/utils"
)

const (
	crawlerUserAgent = "Mozilla/5.0 (Windows NT 6.3; Win64; x64; rv:76.0) Gecko/20100101 Firefox/76.0','accept' : 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9','accept-language' : 'es-419,es;q=0.9,es-ES;q=0.8,en;q=0.7,en-GB;q=0.6,en-US;q=0.5"
	activationUrl    = "https://api.twitter.com/1.1/guest/activate.json"
	apiUrl           = "https://api.twitter.com/1.1/statuses/lookup.json?id=%s&tweet_mode=extended"
	bearerFileUrl    = "https://twitter.com/i/videos/tweet/%s"
	bearerTokenExp   = "Bearer ([a-zA-Z0-9%-])+"
	bearerFileExp    = `src="(.*js)`
)

type HeaderKey string

const (
	Authorization HeaderKey = "authorization"
	UserAgent     HeaderKey = "User-agent"
	XGuestToken   HeaderKey = "x-guest-token"
)

type HttpMethod string

const (
	GET  HttpMethod = "GET"
	POST HttpMethod = "POST"
)

type twitterDownloaderRepository struct {
	client  *http.Client
	headers map[HeaderKey]string
}

func NewTwitterDownloaderRepository(client *http.Client) *twitterDownloaderRepository {

	headers := map[HeaderKey]string{
		UserAgent: crawlerUserAgent,
	}

	return &twitterDownloaderRepository{
		client:  client,
		headers: headers,
	}
}

func (t *twitterDownloaderRepository) DownloadVideo(url string, authToken ...string) (videoDownload *domain.Video, token *string, err error) {
	videoId := utils.GetVideoId(url)
	var currentToken *string

	if len(authToken) == 0 {
		currentToken, err = t.claimNewGuestToken(videoId, &t.headers)

		if err != nil {
			return nil, nil, err
		}
	}

	if len(authToken) > 0 {
		t.headers[Authorization] = authToken[0]
		currentToken = &authToken[0]
	}

	// Get video status
	showStatusResp, err := t.showStatus(videoId, t.headers)
	slog.Debug("/statuses/lookup.json result", "url", url, "response", string(showStatusResp))

	if err != nil {
		return nil, nil, err
	}

	videoList := &domain.VideoList{}

	err = json.Unmarshal(showStatusResp, videoList)

	if err != nil {
		return nil, nil, err
	}

	if len(*videoList) == 0 {
		return nil, nil, fmt.Errorf("no video found")
	}

	var video domain.Video

	for _, videoIn := range *videoList {
		video = videoIn.Video
		break
	}

	if video.ExtendedEntities.Media == nil || video.ExtendedEntities.Media[0].Type != "video" {

		if video.QuotedStatus.ExtendedEntities.Media == nil {
			return nil, nil, fmt.Errorf("no video found")
		}

		if video.QuotedStatus.ExtendedEntities.Media[0].Type != "video" {
			return nil, nil, fmt.Errorf("no video found")
		}
	}

	return &video, currentToken, nil
}

func (t *twitterDownloaderRepository) claimNewGuestToken(videoId string, headers *map[HeaderKey]string) (authToken *string, err error) {

	// Get bearer file (.js)
	bearerFile, err := t.getBearerFile(videoId, t.headers)

	if err != nil {
		return nil, err
	}

	// Get bearer token
	bearerToken, err := t.getBearerToken(bearerFile, t.headers)

	if err != nil {
		return nil, err
	}

	t.headers[Authorization] = bearerToken

	// Activate guest token
	activation, err := t.activateGuestToken(t.headers)

	if err != nil {
		return nil, err
	}

	t.headers[XGuestToken] = activation

	return &bearerToken, nil
}

func (t *twitterDownloaderRepository) activateGuestToken(headers map[HeaderKey]string) (string, error) {

	activation, err := t.sendRequest(activationUrl, POST, headers)

	if err != nil {
		return "", err
	}

	return string(*activation), nil
}

func (t *twitterDownloaderRepository) getBearerToken(bearerFileUrl string, headers map[HeaderKey]string) (string, error) {

	bearerFileContent, err := t.sendRequest(bearerFileUrl, GET, headers)

	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(bearerTokenExp)

	bearerToken := re.FindStringSubmatch(string(*bearerFileContent))[0]

	return bearerToken, nil
}

func (t *twitterDownloaderRepository) getBearerFile(videoId string, headers map[HeaderKey]string) (string, error) {

	videoUrl := fmt.Sprintf(bearerFileUrl, videoId)

	// get guest token
	tokenRequest, err := t.sendRequest(videoUrl, GET, headers)

	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(bearerFileExp)
	bearerFile := re.FindStringSubmatch(string(*tokenRequest))[1]

	return bearerFile, nil
}

func (t *twitterDownloaderRepository) showStatus(videoId string, headers map[HeaderKey]string) ([]byte, error) {

	apiEp := fmt.Sprintf(apiUrl, videoId)

	resp, err := t.sendRequest(apiEp, GET, headers)

	if err != nil {
		return nil, err
	}

	return *resp, nil
}

func (t *twitterDownloaderRepository) sendRequest(url string, method HttpMethod, headers map[HeaderKey]string) (*[]byte, error) {

	client := &http.Client{}

	req, err := http.NewRequest(string(method), url, nil)

	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(string(key), value)
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return &body, nil
}
