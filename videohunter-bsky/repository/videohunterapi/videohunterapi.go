package videohunterapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/victoraldir/myvideohunterbsky/domain"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type videoHunterApi struct {
	client HttpClient
}

func NewVideoHunterApi(client HttpClient) *videoHunterApi {
	return &videoHunterApi{
		client: client,
	}
}

func (v *videoHunterApi) DownloadVideo(videoUrl string) (domain.VideoUrl, error) {

	url, err := url.Parse("https://myvideohunter.com/prod/url/")
	if err != nil {
		slog.Debug("Error parsing videohunter url", slog.Any("error", err))
		return domain.VideoUrl{}, err
	}

	req := http.Request{
		Method: http.MethodPost,
		URL:    url,
		Body:   io.NopCloser(bytes.NewBuffer([]byte(fmt.Sprintf(`{"video_url": "%s"}`, videoUrl)))),
	}

	resp, err := v.client.Do(&req)
	if err != nil {
		slog.Debug("Error getting video url from videohunter", slog.Any("error", err))
		return domain.VideoUrl{}, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Debug("Error getting video url from videohunter", slog.Any("status", resp.Status))
		return domain.VideoUrl{}, fmt.Errorf("error getting video url from videohunter: %s", resp.Status)
	}

	// Unmarshal response
	videoUrlResponse := domain.VideoUrl{}
	err = json.NewDecoder(resp.Body).Decode(&videoUrl)
	if err != nil {
		slog.Debug("Error decoding video url from videohunter", slog.Any("error", err))
		return domain.VideoUrl{}, err
	}

	return videoUrlResponse, nil
}
