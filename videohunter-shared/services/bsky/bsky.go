package bsky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/enescakir/emoji"
	"github.com/victoraldir/myvideohunterapi/utils"

	shared_domain "github.com/victoraldir/myvideohuntershared/domain"
	"github.com/victoraldir/myvideohuntershared/services"
)

const (
	scheme   = "https"
	facetUrl = "www.myvideohunter.com/..."
	host     = "public.api.bsky.app"
)

type BskyService interface {
	SearchPostsByMention(mention, since, until string) ([]shared_domain.Post, error)
	EnrichPost(posts *[]shared_domain.Post) error
	Reply(post shared_domain.Post) error
	LoadEmbed(url string) (*shared_domain.EmbedExternal, error)
	GetPostsByUris(uris []string) ([]shared_domain.Url, error)
	GetPostsByUrisAPI(uris []string) ([]shared_domain.Post, error)
	Login() (*shared_domain.Session, error)
	RefreshSession(session *shared_domain.Session) (*shared_domain.Session, error)
	SetSession(session *shared_domain.Session)
	IsSessionExpired() bool
}

type bskyService struct {
	client   services.HttpClient
	session  *shared_domain.Session
	userName string
	password string
}

func NewBskyService(client services.HttpClient, username, password string) *bskyService {
	return &bskyService{
		client:   client,
		userName: username,
		password: password,
	}
}

func (b *bskyService) SearchPostsByMention(mention, since, until string) ([]shared_domain.Post, error) {

	// curl --location 'https://public.api.bsky.app/xrpc/app.bsky.feed.searchPosts?q=%22%40myvideohunter.com%22' --header 'Authorization: Bearer token'
	slog.Debug("Searching posts by mention: %s", slog.String("mention", mention))

	req := http.Request{
		URL: &url.URL{
			Scheme: scheme,
			Host:   "bsky.social",
			Path:   "/xrpc/app.bsky.feed.searchPosts",
			RawQuery: url.Values{
				"q":     []string{mention},
				"since": []string{since},
				"until": []string{until},
				"limit": []string{"100"},
				"sort":  []string{"latest"},
			}.Encode(),
		},
	}

	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", b.session.AccessJwt)},
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
	posts := shared_domain.Posts{}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&posts)
	if err != nil {
		slog.Debug("Error unmarshalling response", slog.Any("error", err))
		return nil, err
	}

	slog.Debug("Posts found", slog.Any("posts", posts))

	return posts.Posts, nil
}

func (b *bskyService) EnrichPost(posts *[]shared_domain.Post) error {

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
	urlMap := make(map[string]shared_domain.Url)
	for i := 0; i < len(urls); i++ {
		urlMap[urls[i].OriginalId] = urls[i]
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

func (b *bskyService) GetPostsByUris(uris []string) ([]shared_domain.Url, error) {

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
	urls := make([]shared_domain.Url, 0)

	err = json.NewDecoder(resp.Body).Decode(&urls)
	if err != nil {
		slog.Debug("Error unmarshalling response", slog.Any("error", err))
		return nil, err
	}

	defer resp.Body.Close()

	return urls, nil
}

func (b *bskyService) Reply(post shared_domain.Post) error {

	url := fmt.Sprintf("https://www.myvideohunter.com/prod/url/%s", post.Url.Id)
	embed, err := b.LoadEmbed(url)
	if err != nil {
		slog.Debug("Error loading embed", slog.Any("error", err))
	}

	feature := shared_domain.Feature{
		Type: "app.bsky.richtext.facet#link",
		Uri:  url,
	}

	text := fmt.Sprintf("Hello %s\nHere's your video%s\n%s", emoji.WavingHand, emoji.BackhandIndexPointingDown, facetUrl)

	byteStart := len(text) - len(facetUrl)
	byteEnd := len(text)

	facet := shared_domain.Facet{
		Features: []shared_domain.Feature{feature},
		Index: shared_domain.Index{
			ByteStart: byteStart,
			ByteEnd:   byteEnd,
		},
	}

	reply := shared_domain.RecordReply{
		Text:          text,
		CreatedAt:     time.Now().Format(time.RFC3339),
		EmbedExternal: embed,
		Facets:        []shared_domain.Facet{facet},
		Reply: shared_domain.Reply{
			Parent: shared_domain.PostItem{
				Cid: post.Cid,
				Uri: post.Uri,
			},
			Root: shared_domain.PostItem{
				Cid: post.Record.Reply.Root.Cid,
				Uri: post.Record.Reply.Root.Uri,
			},
		}}

	postReply := shared_domain.PostReply{
		Record:     reply,
		Repo:       "did:plc:3jirv55ij45i7lmjjelu5ukn",
		Collection: "app.bsky.feed.post",
	}

	bodyBytes, _ := json.Marshal(postReply)

	req, err := http.NewRequest(http.MethodPost, "https://bsky.social/xrpc/com.atproto.repo.createRecord", bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Debug("Error creating request", slog.Any("error", err))
		return err
	}

	// Set Authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.session.AccessJwt))
	req.Header.Set("Content-Type", "application/json")

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

func (b *bskyService) LoadEmbed(url string) (*shared_domain.EmbedExternal, error) {

	slog.Debug("Loading embed", slog.String("url", url))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		slog.Debug("Error creating request", slog.Any("error", err))
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		slog.Debug("Error getting embed", slog.Any("error", err))
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Debug("Error reading response", slog.Any("error", err))
		return nil, err
	}

	// Get og:title
	ogTitle := GetMetaTag(body, "og:title")
	// Get og:description
	ogDescription := GetMetaTag(body, "og:description")

	embed := shared_domain.EmbedExternal{
		Type: "app.bsky.embed.external",
		External: shared_domain.External{
			Title:       ogTitle,
			Description: ogDescription,
			Uri:         url,
		},
	}

	return &embed, nil

}

func GetMetaTag(body []byte, metaName string) string {
	metaTag := fmt.Sprintf(`<meta property="%s" content="`, metaName)
	start := bytes.Index(body, []byte(metaTag))
	if start == -1 {
		return ""
	}

	start += len(metaTag)
	end := bytes.Index(body[start:], []byte(`"`))
	if end == -1 {
		return ""
	}

	return string(body[start : start+end])
}

func (b *bskyService) Login() (*shared_domain.Session, error) {
	url := "https://bsky.social/xrpc/com.atproto.server.createSession"
	body := struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}{
		Identifier: b.userName,
		Password:   b.password,
	}

	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error logging in: %s", resp.Status)
	}

	// Unmarshal response
	session := shared_domain.Session{}
	json.NewDecoder(resp.Body).Decode(&session)

	defer resp.Body.Close()

	b.session = &session

	return &session, nil
}

func (b *bskyService) RefreshSession(session *shared_domain.Session) (*shared_domain.Session, error) {
	url := "https://bsky.social/xrpc/com.atproto.server.refreshSession"

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	if session.RefreshJwt == "" {
		return nil, fmt.Errorf("refreshJwt is empty")
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", session.RefreshJwt))

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error refreshing session: %s", resp.Status)
	}

	// Unmarshal response
	newSession := shared_domain.Session{}

	json.NewDecoder(resp.Body).Decode(&session)

	defer resp.Body.Close()

	b.session = &newSession

	return &newSession, nil
}

func (b *bskyService) SetSession(session *shared_domain.Session) {
	b.session = session
}

func (b *bskyService) IsSessionExpired() bool {
	// https://bsky.social/xrpc/com.atproto.server.getSession
	req, err := http.NewRequest(http.MethodGet, "https://bsky.social/xrpc/com.atproto.server.getSession", nil)
	if err != nil {
		return true
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", b.session.AccessJwt))

	resp, err := b.client.Do(req)
	if err != nil {
		return true
	}

	if resp.StatusCode != http.StatusOK {
		return true
	}

	return false
}

func (r *bskyService) GetPostsByUrisAPI(uris []string) ([]shared_domain.Post, error) {

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
	posts := shared_domain.Posts{}

	err = json.NewDecoder(resp.Body).Decode(&posts)
	if err != nil {
		slog.Debug("Error unmarshalling response", slog.Any("error", err))
		return nil, err
	}

	defer resp.Body.Close()

	return posts.Posts, nil

}

func (r *bskyService) DownloadVideo(urlPost string, authToken ...string) (videoDownload *shared_domain.Video, currentToken *string, err error) {

	videoId := utils.GetVideoId(urlPost)
	uri := utils.UrlToUriAt(urlPost)

	// https://public.api.bsky.app/xrpc/app.bsky.feed.getPostThread?uri=at://amontis.bsky.social/app.bsky.feed.post/3lecw7cvho22n&depth=10
	req := http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   "/xrpc/app.bsky.feed.getPostThread",
			RawQuery: url.Values{
				"uri": []string{uri},
			}.Encode(),
		},
	}

	resp, err := r.client.Do(&req)
	if err != nil {
		slog.Debug("Error getting posts from bsky", slog.Any("error", err))
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Debug("Error getting posts from bsky", slog.Any("status", resp.Status))
		return nil, nil, fmt.Errorf("error getting posts from bsky: %s", resp.Status)
	}

	// Read response
	content, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		slog.Debug("Error reading response", slog.Any("error", err))
		return nil, nil, err
	}

	// Unmarshal response
	root := Root{}
	json.Unmarshal(content, &root)

	if root.Thread.Post.Embed.Type == "" {
		return nil, nil, fmt.Errorf("no video found")
	}

	video := ThreadToVideo(&root.Thread, urlPost, videoId)

	return video, nil, nil
}

func ThreadToVideo(thread *Thread, url, videoId string) *shared_domain.Video {

	var media Media

	if thread.Post.Embed.Type == "app.bsky.embed.video#view" {
		media = Media{
			AspectRatio: AspectRatio{
				Height: thread.Post.Embed.AspectRatio.Height,
				Width:  thread.Post.Embed.AspectRatio.Width,
			},
			Playlist:  thread.Post.Embed.Playlist,
			Thumbnail: thread.Post.Embed.Thumbnail,
		}
	} else if thread.Post.Embed.Type == "app.bsky.embed.record#view" {
		embed := thread.Post.Embed.Record.Embeds[0]

		media = Media{
			AspectRatio: AspectRatio{
				Height: embed.AspectRatio.Height,
				Width:  embed.AspectRatio.Width,
			},
			Playlist:  embed.Playlist,
			Thumbnail: embed.Thumbnail,
		}
	} else {
		media = thread.Post.Embed.Media
	}

	video := shared_domain.Video{
		ThumbnailUrl:     media.Thumbnail,
		OriginalId:       thread.Post.URI,
		OriginalVideoUrl: url,
		Text:             thread.Post.Record.Text,
		CreatedAt:        thread.Post.Record.CreatedAt.String(),
		IdDB:             videoId,
		Size:             thread.Post.Record.Embed.Video.Size,
		MimeType:         "video/mp4",
		ExtendedEntities: shared_domain.ExtendedEntities{
			Media: []shared_domain.Media{
				{
					MediaUrl: media.Playlist,
					Type:     "video",
					VideoInfo: shared_domain.VideoInfo{
						Variants: []shared_domain.Variants{
							{
								Bitrate:     media.AspectRatio.Height * media.AspectRatio.Width,
								URL:         media.Playlist,
								ContentType: "video/mp4",
							},
						},
					},
				},
			},
		},
	}

	return &video

}
