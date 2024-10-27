package bsky

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/victoraldir/myvideohunterbsky/domain"
	"github.com/victoraldir/myvideohunterbsky/services"
)

const (
	scheme = "https"
	host   = "public.api.bsky.app"
)

type bskyService struct {
	client services.HttpClient
	token  string
}

func NewBskyService(client services.HttpClient) *bskyService {
	return &bskyService{
		client: client,
	}
}

func (b *bskyService) SearchPostsByMention(mention, since string) ([]domain.Post, error) {

	// curl --location 'https://public.api.bsky.app/xrpc/app.bsky.feed.searchPosts?q=%22%40myvideohunter.com%22' --header 'Authorization: Bearer token'
	slog.Debug("Searching posts by mention: %s", slog.String("mention", mention))

	req := http.Request{
		URL: &url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   "/xrpc/app.bsky.feed.searchPosts",
			RawQuery: url.Values{
				"q":     []string{mention},
				"since": []string{since},
			}.Encode(),
		},
	}

	resp, err := b.client.Do(&req)
	if err != nil {
		slog.Debug("Error getting posts from bsky", slog.Any("error", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Debug("Error getting posts from bsky", slog.Any("status", resp.Status))
		return nil, fmt.Errorf("error getting posts from bsky: %s", resp.Status)
	}

	// Unmarshal response
	posts := domain.Posts{}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&posts)
	if err != nil {
		slog.Debug("Error unmarshalling response", slog.Any("error", err))
		return nil, err
	}

	slog.Debug("Posts found", slog.Any("posts", posts))

	return posts.Posts, nil
}

func (b *bskyService) EnrichPost(posts *[]domain.Post) error {

	// curl --location --globoff 'https://public.api.bsky.app/xrpc/app.bsky.feed.getPosts?uris[]=at%3A%2F%2Fdid%3Aplc%3A3fibociwu7jy4bbdjhmm4nop%2Fapp.bsky.feed.post%2F3l5fgldunxk2y&uris[]=at%3A%2F%2Fdid%3Aplc%3Aaca4rpd2skm56qugeb6o4fua%2Fapp.bsky.feed.post%2F3l5nhkzz62d2k'

	// uriSize := 25
	uris := make([]string, 0)
	// postsEnriched := make([]domain.Post, 0)
	// postMap := make(map[string]domain.Post)

	for i := 0; i < len(*posts); i++ {
		uris = append(uris, (*posts)[i].Record.Reply.Root.Uri)
	}

	slog.Debug("Enriching posts", slog.Any("uris", uris))
	rootPosts, err := b.GetPostsByUris(uris)
	if err != nil {
		slog.Debug("Error enriching posts", slog.Any("error", err))
		return err
	}

	// create a map of posts
	postMap := make(map[string]domain.Post, len(rootPosts))
	for i := 0; i < len(rootPosts); i++ {
		postMap[rootPosts[i].Uri] = rootPosts[i]
	}

	// Enrich posts
	for i := 0; i < len(*posts); i++ {

		currentPost := postMap[(*posts)[i].Record.Reply.Root.Uri]

		// If the post is a video, enrich it
		if currentPost.Record.Embed.Video.MimeType == "video/mp4" {
			(*posts)[i].RootVideo = domain.RootVideo{
				Cid:        currentPost.Embed.Cid,
				Thumbnail:  currentPost.Embed.Thumbnail,
				Playlist:   currentPost.Embed.Playlist,
				AspecRatio: currentPost.Embed.AspecRatio,
			}
		}
	}

	return nil
}

func (b *bskyService) GetPostsByUris(uris []string) ([]domain.Post, error) {

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

	resp, err := b.client.Do(&req)
	if err != nil {
		slog.Debug("Error getting posts from bsky", slog.Any("error", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Debug("Error getting posts from bsky", slog.Any("status", resp.Status))
		return nil, fmt.Errorf("error getting posts from bsky: %s", resp.Status)
	}

	// Unmarshal response
	posts := domain.Posts{}

	err = json.NewDecoder(resp.Body).Decode(&posts)
	if err != nil {
		slog.Debug("Error unmarshalling response", slog.Any("error", err))
		return nil, err
	}

	defer resp.Body.Close()

	return posts.Posts, nil

}
