package usecase

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/victoraldir/myvideohunterbsky/repository/dynamodb"
	"github.com/victoraldir/myvideohuntershared/domain"
	"github.com/victoraldir/myvideohuntershared/services/bsky"
)

const (
	// searchOverlap widens the search window backwards to tolerate Bluesky
	// search indexing lag: the server-side 'since' filter works on the post
	// createdAt, which can precede the moment a post becomes searchable.
	searchOverlap = 10 * time.Minute

	// maxRepliedPosts caps the dedupe list stored in the settings table.
	maxRepliedPosts = 500
)

type FetchPostRequest struct {
	BotName string
}

type FetchPost interface {
	Execute(request FetchPostRequest) error
}

type fetchPost struct {
	bskyService bsky.BskyService
	dynamodb    dynamodb.DynamodbRepository
}

func NewFetchPost(bskyService bsky.BskyService, dynamodb dynamodb.DynamodbRepository) FetchPost {
	return &fetchPost{
		dynamodb:    dynamodb,
		bskyService: bskyService,
	}
}

func (f *fetchPost) Execute(request FetchPostRequest) error {

	var lastScanTime string

	// Get now zulu time (UTC)
	now := time.Now().UTC().Format(time.RFC3339)

	lastScan, err := f.dynamodb.GetSetting(domain.BskyLastExecutionTime)
	if err != nil {
		slog.Error("Error getting last scan from dynamodb", "error", err)
	}

	if lastScan != nil {
		lastScanTime = lastScan.Value
	}

	if lastScan == nil {
		slog.Info("Last scan not found, setting to now")
		lastScanTime = now
	}

	// Widen the search window backwards: a post created near the window
	// boundary may only become searchable after the run that covered its
	// createdAt already executed (Bluesky search indexing lag).
	since := lastScanTime
	if parsed, parseErr := time.Parse(time.RFC3339, lastScanTime); parseErr == nil {
		since = parsed.Add(-searchOverlap).UTC().Format(time.RFC3339)
	}

	slog.Info("Fetching posts", slog.Any("since", since), slog.Any("last_scan", lastScanTime), slog.Any("until", now))

	_, err = f.GetSession()
	if err != nil {
		slog.Error("Error getting session", "error", err)
		return err
	}

	// search for posts
	posts, err := f.bskyService.SearchPostsByMention(request.BotName, since, now)
	if err != nil {
		slog.Error("Error searching posts by mention", "error", err)
		return err
	}

	slog.Info("Posts found", slog.Any("posts_count", len(posts)))

	// The overlapped window re-shows posts from previous scans. Only process
	// posts indexed after the previous scan and never replied to.
	replied := f.getRepliedPosts()
	newPosts := make([]domain.Post, 0, len(posts))
	for _, post := range posts {
		if f.wasProcessed(post, lastScanTime, replied) {
			continue
		}
		newPosts = append(newPosts, post)
	}

	if len(newPosts) > 0 {
		// enrich posts
		err = f.bskyService.EnrichPost(&newPosts)
		slog.Info("Posts enriched", slog.Any("posts_count", len(newPosts)))
		if err != nil {
			slog.Error("Error enriching posts", "error", err)
		}

		// reply posts
		for _, post := range newPosts {

			if post.Url == nil {
				slog.Info("Post without url, skipping", slog.Any("post", post.Cid))
				continue
			}

			err = f.bskyService.Reply(post)
			if err != nil {
				slog.Error("Error replying post", "error", err)
				continue
			}

			replied[post.Uri] = struct{}{}
		}

		f.saveRepliedPosts(replied)
	}

	slog.Info("Saving last scan time", slog.Any("last_scan_time", now))

	f.dynamodb.SaveSetting(&domain.Settings{
		KeySetting: string(domain.BskyLastExecutionTime),
		Value:      now,
	})

	return nil
}

func (f *fetchPost) GetSession() (*domain.Session, error) {

	// Get token, refresh token, and last scan and
	token, err := f.dynamodb.GetSetting(domain.BskyAccessToken)
	if err != nil {
		slog.Error("Error getting token from dynamodb", "error", err)
		return nil, err
	}

	refreshToken, err := f.dynamodb.GetSetting(domain.BskyRefreshToken)
	if err != nil {
		slog.Error("Error getting refresh token from dynamodb", "error", err)
		return nil, err
	}

	if token == nil || refreshToken == nil || refreshToken.Value == "" || token.Value == "" {
		slog.Info("Token or refresh token not found, logging in")
		return f.login()
	}

	newSession := &domain.Session{
		AccessJwt:  token.Value,
		RefreshJwt: refreshToken.Value,
	}

	f.bskyService.SetSession(newSession)

	isExpired := f.bskyService.IsSessionExpired()
	if isExpired {
		slog.Info("Session expired, refreshing")
		newSessionRefreshed, err := f.bskyService.RefreshSession(newSession)
		if err != nil {
			slog.Error("Error refreshing session, falling back to login", "error", err)
			return f.login()
		}

		if newSessionRefreshed == nil || newSessionRefreshed.AccessJwt == "" {
			slog.Error("Refreshed session is empty, falling back to login")
			return f.login()
		}

		slog.Info("Session refreshed")
		f.dynamodb.SaveSetting(&domain.Settings{
			KeySetting: string(domain.BskyAccessToken),
			Value:      newSessionRefreshed.AccessJwt,
		})

		f.dynamodb.SaveSetting(&domain.Settings{
			KeySetting: string(domain.BskyRefreshToken),
			Value:      newSessionRefreshed.RefreshJwt,
		})
	}

	return newSession, nil
}

func (f *fetchPost) login() (*domain.Session, error) {

	session, err := f.bskyService.Login()
	if err != nil {
		slog.Error("Error logging in bsky", "error", err)
		return nil, err
	}

	slog.Info("Logged in")
	f.dynamodb.SaveSetting(&domain.Settings{
		KeySetting: string(domain.BskyAccessToken),
		Value:      session.AccessJwt,
	})

	f.dynamodb.SaveSetting(&domain.Settings{
		KeySetting: string(domain.BskyRefreshToken),
		Value:      session.RefreshJwt,
	})

	return session, nil
}

func (f *fetchPost) wasProcessed(post domain.Post, lastScanTime string, replied map[string]struct{}) bool {

	if _, ok := replied[post.Uri]; ok {
		slog.Info("Post already replied, skipping", slog.Any("post", post.Uri))
		return true
	}

	prevScan, err := time.Parse(time.RFC3339Nano, lastScanTime)
	if err != nil {
		slog.Error("Error parsing last scan time", slog.Any("last_scan", lastScanTime), slog.Any("error", err))
		return false
	}

	indexedAt, err := time.Parse(time.RFC3339Nano, post.IndexedAt)
	if err != nil {
		slog.Error("Error parsing post indexedAt", slog.Any("post", post.Uri), slog.Any("error", err))
		return false
	}

	if !indexedAt.After(prevScan) {
		slog.Info("Post already covered by previous scan, skipping",
			slog.Any("post", post.Uri),
			slog.Any("indexed_at", post.IndexedAt),
			slog.Any("last_scan", lastScanTime))
		return true
	}

	return false
}

func (f *fetchPost) getRepliedPosts() map[string]struct{} {

	replied := map[string]struct{}{}

	setting, err := f.dynamodb.GetSetting(domain.BskyRepliedPosts)
	if err != nil {
		slog.Error("Error getting replied posts from dynamodb", "error", err)
		return replied
	}

	if setting == nil || setting.Value == "" {
		return replied
	}

	var uris []string
	if err := json.Unmarshal([]byte(setting.Value), &uris); err != nil {
		slog.Error("Error unmarshalling replied posts", "error", err)
		return replied
	}

	for _, uri := range uris {
		replied[uri] = struct{}{}
	}

	return replied
}

func (f *fetchPost) saveRepliedPosts(replied map[string]struct{}) {

	uris := make([]string, 0, len(replied))
	for uri := range replied {
		uris = append(uris, uri)
	}

	// Cap the list so the settings item stays small.
	if len(uris) > maxRepliedPosts {
		uris = uris[len(uris)-maxRepliedPosts:]
	}

	body, err := json.Marshal(uris)
	if err != nil {
		slog.Error("Error marshalling replied posts", "error", err)
		return
	}

	f.dynamodb.SaveSetting(&domain.Settings{
		KeySetting: string(domain.BskyRepliedPosts),
		Value:      string(body),
	})
}
