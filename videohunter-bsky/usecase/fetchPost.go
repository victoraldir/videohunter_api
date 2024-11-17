package usecase

import (
	"log/slog"
	"time"

	"github.com/victoraldir/myvideohunterbsky/domain"
	"github.com/victoraldir/myvideohunterbsky/repository/dynamodb"
	"github.com/victoraldir/myvideohunterbsky/services/bsky"
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

	slog.Info("Fetching posts", slog.Any("since", lastScanTime), slog.Any("until", now))

	_, err = f.GetSession()
	if err != nil {
		slog.Error("Error getting session", "error", err)
		return err
	}

	// search for posts
	posts, err := f.bskyService.SearchPostsByMention(request.BotName, lastScanTime, now)
	if err != nil {
		slog.Error("Error searching posts by mention", "error", err)
		return err
	}

	slog.Info("Posts found", slog.Any("posts_count", len(posts)))

	if len(posts) > 0 {
		// enrich posts
		err = f.bskyService.EnrichPost(&posts)
		slog.Info("Posts enriched", slog.Any("posts_count", len(posts)))
		if err != nil {
			slog.Error("Error enriching posts", "error", err)
		}

		// reply posts
		for _, post := range posts {

			if post.Url == nil {
				slog.Info("Post without url, skipping", slog.Any("post", post.Cid))
				continue
			}

			err = f.bskyService.Reply(post)
			if err != nil {
				slog.Error("Error replying post", "error", err)
			}
		}
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

	if token == nil || refreshToken == nil {
		slog.Info("Token or refresh token not found, logging in")
		session, err := f.bskyService.Login()
		if err != nil {
			slog.Error("Error logging in bsky", "error", err)
			return nil, err
		}

		slog.Info("Logged in", slog.Any("session", session))
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

	newSession := &domain.Session{
		AccessJwt:  token.Value,
		RefreshJwt: refreshToken.Value,
	}

	f.bskyService.SetSession(newSession)

	isExpired := f.bskyService.IsSessionExpired()
	if isExpired {
		slog.Info("Session expired, refreshing", slog.Any("session", newSession))
		newSession, err = f.bskyService.RefreshSession(newSession)
		if err != nil {
			slog.Error("Error refreshing session", "error", err)
			return nil, err
		}

		slog.Info("Session refreshed", slog.Any("session", newSession))
		f.dynamodb.SaveSetting(&domain.Settings{
			KeySetting: string(domain.BskyAccessToken),
			Value:      newSession.AccessJwt,
		})

		f.dynamodb.SaveSetting(&domain.Settings{
			KeySetting: string(domain.BskyRefreshToken),
			Value:      newSession.RefreshJwt,
		})
	}

	return newSession, nil
}
