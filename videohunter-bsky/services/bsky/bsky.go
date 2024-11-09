package bsky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

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

	// curl --location --globoff 'https://myvideohunter.com/prod/url/batch'

	uris := make([]string, 0)

	for i := 0; i < len(*posts); i++ {
		uris = append(uris, (*posts)[i].Record.Reply.Root.Uri)
	}

	slog.Debug("Enriching posts", slog.Any("uris", uris))
	urls, err := b.GetPostsByUris(uris)
	if err != nil {
		slog.Debug("Error enriching posts", slog.Any("error", err))
		return err
	}

	// create a map of urls
	urlMap := make(map[string]domain.Url)
	for i := 0; i < len(urls); i++ {
		urlMap[urls[i].Uri] = urls[i]
	}

	// Enrich posts
	for i := 0; i < len(*posts); i++ {

		currentPost := (*posts)[i]

		if _, ok := urlMap[currentPost.Record.Reply.Root.Uri]; ok {
			url := urlMap[currentPost.Record.Reply.Root.Uri]
			(*posts)[i].Url = &url
		}
	}

	return nil
}

func (b *bskyService) GetPostsByUris(uris []string) ([]domain.Url, error) {

	slog.Debug("Getting posts by uris", slog.Any("uris", uris))

	body := struct {
		Uris []string `json:"uris"`
	}{
		Uris: uris,
	}

	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "https://myvideohunter.com/prod/url/batch", bytes.NewReader(bodyBytes))

	resp, err := b.client.Do(req)
	if err != nil {
		slog.Debug("Error getting posts from bsky", slog.Any("error", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Debug("Error getting posts from bsky", slog.Any("status", resp.Status))
		return nil, fmt.Errorf("error getting posts from bsky: %s", resp.Status)
	}

	// Unmarshal response
	urls := make([]domain.Url, 0)

	err = json.NewDecoder(resp.Body).Decode(&urls)
	if err != nil {
		slog.Debug("Error unmarshalling response", slog.Any("error", err))
		return nil, err
	}

	defer resp.Body.Close()

	return urls, nil
}

func (b *bskyService) Reply(post domain.Post) error {

	reply := domain.RecordReply{
		Text:      fmt.Sprintf("Hello! 👋 \n here's your url https://www.myvideohunter.com/prod/url/%s", post.Url.Id),
		CreatedAt: time.Now().Format(time.RFC3339),
		Reply: domain.Reply{
			Parent: domain.PostItem{
				Cid: post.Cid,
				Uri: post.Uri,
			},
			Root: domain.PostItem{
				Cid: post.Record.Reply.Root.Cid,
				Uri: post.Record.Reply.Root.Uri,
			},
		}}

	postReply := domain.PostReply{
		Record:     reply,
		Repo:       "did:plc:3jirv55ij45i7lmjjelu5ukn",
		Collection: "app.bsky.feed.post",
	}

	bodyBytes, _ := json.Marshal(postReply)
	fmt.Println(string(bodyBytes))

	req, err := http.NewRequest(http.MethodPost, "https://bsky.social/xrpc/com.atproto.repo.createRecord", bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Debug("Error creating request", slog.Any("error", err))
		return err
	}

	// Set Authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "eyJ0eXAiOiJhdCtqd3QiLCJhbGciOiJFUzI1NksifQ.eyJzY29wZSI6ImNvbS5hdHByb3RvLmFjY2VzcyIsInN1YiI6ImRpZDpwbGM6M2ppcnY1NWlqNDVpN2xtamplbHU1dWtuIiwiaWF0IjoxNzMxMTcwODQ2LCJleHAiOjE3MzExNzgwNDYsImF1ZCI6ImRpZDp3ZWI6cG9yY2luaS51cy1lYXN0Lmhvc3QuYnNreS5uZXR3b3JrIn0.xfYzBbLboLR1fDMZdqBPGU0B3UnuKhdlpG1hu0Q1xaJOQcxDK8fuZjzHrMKaQx9R-Ansg5ajLk3bzDzxpajlKQ"))

	resp, err := b.client.Do(req)

	if err != nil {
		slog.Debug("Error replying to post", slog.Any("error", err))
		return err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Debug("Error replying to post", slog.Any("status", resp.Status))
		return fmt.Errorf("error replying to post: %s. repose: %s", resp.Status, resp.Body)
	}

	return nil
}
