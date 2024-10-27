package bsky

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/victoraldir/myvideohunterapi/adapters/httpclient"
)

const (
	scheme = "https"
	host   = "public.api.bsky.app"
)

type BskyDownloaderRepository struct {
	client httpclient.HttpClient
}

func NewBskyDownloaderRepository(client httpclient.HttpClient) *BskyDownloaderRepository {
	return &BskyDownloaderRepository{
		client: client,
	}
}

func (r *BskyDownloaderRepository) GetPostsByUris(uris []string) ([]Post, error) {

	slog.Debug("Getting posts by uris", slog.Any("uris", uris))

	req := http.Request{
		URL: &url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   "/xrpc/app.bsky.feed.getPosts",
			RawQuery: url.Values{
				"uris": uris,
			}.Encode(),
		},
	}

	resp, err := r.client.Do(&req)
	if err != nil {
		slog.Debug("Error getting posts from bsky", slog.Any("error", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Debug("Error getting posts from bsky", slog.Any("status", resp.Status))
		return nil, fmt.Errorf("error getting posts from bsky: %s", resp.Status)
	}

	// Unmarshal response
	posts := Posts{}

	err = json.NewDecoder(resp.Body).Decode(&posts)
	if err != nil {
		slog.Debug("Error unmarshalling response", slog.Any("error", err))
		return nil, err
	}

	defer resp.Body.Close()

	return posts.Posts, nil

}
