package usecase

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"

	"github.com/victoraldir/myvideohuntershared/domain"
	"github.com/victoraldir/myvideohuntershared/services/getxapi"
	"github.com/victoraldir/myvideohuntershared/services/videohunterapi"
	"github.com/victoraldir/myvideohuntertwitter/pkg/message"
	"github.com/victoraldir/myvideohuntertwitter/repository/dynamodb"
)

const (
	// maxRepliedMentions caps the per-bot dedupe list stored in the settings table.
	maxRepliedMentions = 500

	// maxSearchPages caps how far back a single run paginates.
	maxSearchPages = 5
)

// statusURLRegex matches a tweet URL (with or without username) inside a mention text.
var statusURLRegex = regexp.MustCompile(`https?://(?:www\.)?(?:twitter\.com|x\.com)/(?:[A-Za-z0-9_]+/status/|i/web/status/)\d+`)

// errNoVideo is returned when a mention does not point to any video tweet.
var errNoVideo = errors.New("no video tweet found in mention")

// errParentUnavailable is returned when the tweet a mention replies to can no
// longer be read (deleted, protected, or the author blocked the bot).
var errParentUnavailable = errors.New("parent tweet unavailable")

// errDownloadLinkFailed is returned when Video Hunter could not build a
// download link for the video, usually a transient upstream failure.
var errDownloadLinkFailed = errors.New("could not create download link")

// BotConfig describes a single bot account handled by the scheduler.
type BotConfig struct {
	Username  string
	Language  string
	AuthToken string
}

type FetchMentionsRequest struct {
	Bots []BotConfig
}

type FetchMentions interface {
	Execute(request FetchMentionsRequest) error
}

type fetchMentions struct {
	getxapiService getxapi.GetXAPIService
	videoHunterApi videohunterapi.VideoHunterApi
	dynamodb       dynamodb.DynamodbRepository
}

func NewFetchMentions(getxapiService getxapi.GetXAPIService, videoHunterApi videohunterapi.VideoHunterApi, dynamodb dynamodb.DynamodbRepository) FetchMentions {
	return &fetchMentions{
		getxapiService: getxapiService,
		videoHunterApi: videoHunterApi,
		dynamodb:       dynamodb,
	}
}

func (f *fetchMentions) Execute(request FetchMentionsRequest) error {
	var firstErr error

	for _, bot := range request.Bots {
		if bot.Username == "" {
			continue
		}

		if err := f.processBot(bot); err != nil {
			slog.Error("Error processing bot", slog.String("bot", bot.Username), slog.Any("error", err))
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

func (f *fetchMentions) processBot(bot BotConfig) error {
	lastMentionIDKey := domain.TwitterBotSettingKey(domain.TwitterLastMentionID, bot.Username)
	repliedMentionsKey := domain.TwitterBotSettingKey(domain.TwitterRepliedMentions, bot.Username)

	lastMentionID, err := f.getLastMentionID(lastMentionIDKey)
	if err != nil {
		slog.Error("Error getting last mention id", slog.String("bot", bot.Username), slog.Any("error", err))
		return err
	}

	tweets, err := f.collectNewMentions(bot.Username, lastMentionID)
	if err != nil {
		slog.Error("Error searching mentions", slog.String("bot", bot.Username), slog.Any("error", err))
		return err
	}

	slog.Info("Mentions fetched",
		slog.String("bot", bot.Username),
		slog.Int("mentions_count", len(tweets)),
		slog.String("last_mention_id", lastMentionID))

	if len(tweets) == 0 {
		runStats{lastMentionID: lastMentionID, newWatermark: lastMentionID}.log(bot.Username)
		return nil
	}

	replied := f.getRepliedMentions(repliedMentionsKey)

	// Process oldest-first so the watermark only advances over handled mentions.
	sort.Slice(tweets, func(i, j int) bool {
		return isNewer(tweets[j].ID, tweets[i].ID)
	})

	newWatermark := lastMentionID
	stats := runStats{lastMentionID: lastMentionID}

	for _, tweet := range tweets {
		// Never reply to the bot's own tweets.
		if strings.EqualFold(tweet.Author.UserName, bot.Username) {
			stats.ownTweets++
			newWatermark = maxID(newWatermark, tweet.ID)
			continue
		}

		// Safety net for search re-indexing: skip mentions already replied to.
		if _, ok := replied[tweet.ID]; ok {
			stats.alreadyReplied++
			slog.Debug("Mention already replied, skipping", slog.String("mention_id", tweet.ID))
			newWatermark = maxID(newWatermark, tweet.ID)
			continue
		}

		if bot.AuthToken == "" {
			slog.Warn("Auth token missing, cannot reply", slog.String("bot", bot.Username))
			stats.failed = true
			break
		}

		link, err := f.replyToMention(bot, tweet)
		if err != nil {
			if isRetryable(err) {
				// Posting throttles and upstream outages resolve on their own, so
				// stop here and let the next run pick these mentions up. The
				// watermark stays put so nothing is lost.
				stats.throttled++

				action := "throttled by X, will retry next run"
				if !isThrottleError(err) {
					action = "temporary upstream failure, will retry next run"
				}

				slog.Info("Deferring mention",
					slog.String("bot", bot.Username),
					slog.String("mention_id", tweet.ID),
					slog.String("reason", getxapi.Reason(err)),
					slog.String("action", action))
				break
			}

			// Permanent failures (no video, deleted parent, rejected reply) are
			// skipped so they cannot stall the watermark forever.
			stats.skipped++
			slog.Info("Skipping mention",
				slog.String("bot", bot.Username),
				slog.String("mention_id", tweet.ID),
				slog.String("reason", skipReason(err)))
			newWatermark = maxID(newWatermark, tweet.ID)
			continue
		}

		slog.Info("Replied to mention",
			slog.String("bot", bot.Username),
			slog.String("mention_id", tweet.ID),
			slog.String("link", link))
		stats.replied++
		replied[tweet.ID] = struct{}{}
		newWatermark = maxID(newWatermark, tweet.ID)

		// Persist the dedupe list right away so a mid-run failure (e.g. Lambda
		// timeout while working through the first-run backlog) can never lead to
		// a duplicate reply.
		f.saveRepliedMentions(repliedMentionsKey, replied)
	}

	stats.newWatermark = newWatermark
	stats.log(bot.Username)

	if isNewer(newWatermark, lastMentionID) {
		if _, err := f.dynamodb.SaveSetting(&domain.Settings{
			KeySetting: string(lastMentionIDKey),
			Value:      newWatermark,
		}); err != nil {
			slog.Error("Error saving last mention id", slog.String("bot", bot.Username), slog.Any("error", err))
			return err
		}
	}

	return nil
}

// runStats summarizes a single bot run. Individual skips are informational, so
// they are counted and emitted as a single scannable line at the end.
type runStats struct {
	replied        int
	skipped        int
	alreadyReplied int
	ownTweets      int
	throttled      int
	failed         bool
	lastMentionID  string
	newWatermark   string
}

func (s runStats) log(bot string) {
	attrs := []any{
		slog.String("bot", bot),
		slog.Int("replied", s.replied),
		slog.Int("skipped", s.skipped),
		slog.Int("already_replied", s.alreadyReplied),
		slog.Int("own_tweets", s.ownTweets),
		slog.Int("deferred", s.throttled),
		slog.String("last_mention_id", s.lastMentionID),
		slog.String("new_mention_id", s.newWatermark),
	}

	// A deferred mention is the only kind worth flagging, since it means the bot
	// could not finish the queue and will retry.
	if s.needsAttention() {
		slog.Warn("MENTION_SUMMARY", attrs...)
		return
	}

	slog.Info("MENTION_SUMMARY", attrs...)
}

// needsAttention reports whether the run leaves work behind, which is the only
// situation an operator has to care about.
func (s runStats) needsAttention() bool {
	return s.throttled > 0 || s.failed
}

// collectNewMentions returns the mentions newer than lastMentionID. On the
// first run (empty watermark) only the first page is returned, which replies to
// the recent backlog. Afterwards it paginates until it reaches the watermark.
func (f *fetchMentions) collectNewMentions(botUsername, lastMentionID string) ([]domain.Tweet, error) {
	collected := make([]domain.Tweet, 0)
	cursor := ""

	for page := 0; page < maxSearchPages; page++ {
		response, err := f.getxapiService.SearchMentions(botUsername, cursor)
		if err != nil {
			return nil, err
		}

		if response == nil {
			break
		}

		reachedWatermark := false
		for _, tweet := range response.Tweets {
			if lastMentionID != "" && !isNewer(tweet.ID, lastMentionID) {
				reachedWatermark = true
				break
			}
			collected = append(collected, tweet)
		}

		if lastMentionID == "" || reachedWatermark || !response.HasMore || response.NextCursor == "" {
			break
		}

		cursor = response.NextCursor
	}

	return collected, nil
}

func (f *fetchMentions) replyToMention(bot BotConfig, tweet domain.Tweet) (string, error) {
	videoURL, err := f.resolveVideoURL(tweet)
	if err != nil {
		return "", err
	}

	videoUrl, err := f.videoHunterApi.DownloadVideo(videoURL)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errDownloadLinkFailed, err)
	}

	link := videoUrl.GetUrl()
	text := message.BuildReply(bot.Language, link)

	if _, err := f.getxapiService.CreateReply(bot.AuthToken, tweet.ID, text); err != nil {
		return "", err
	}

	return link, nil
}

// resolveVideoURL finds the tweet that owns the video: a status URL inside the
// mention text, the parent tweet when the mention is a reply, or the mention
// itself when it carries media.
func (f *fetchMentions) resolveVideoURL(tweet domain.Tweet) (string, error) {
	if url := statusURLRegex.FindString(tweet.Text); url != "" {
		return url, nil
	}

	if tweet.InReplyToID != "" {
		parent, err := f.getxapiService.GetTweetDetail(tweet.InReplyToID)
		if err != nil {
			if errors.Is(err, getxapi.ErrTweetNotFound) {
				return "", fmt.Errorf("%w: %v", errParentUnavailable, err)
			}
			return "", err
		}
		if parent != nil && parent.URL != "" {
			return parent.URL, nil
		}
	}

	if tweet.HasMedia() {
		return tweet.URL, nil
	}

	return "", errNoVideo
}

// isRetryable reports whether the failure is worth another attempt on the next
// scheduled run. Permanent failures (no video, duplicated/restricted reply) are
// skipped so they do not stall the watermark.
func isRetryable(err error) bool {
	return isThrottleError(err) || errors.Is(err, getxapi.ErrUnavailable)
}

// isThrottleError reports whether X throttled posting for this IP (error 344).
func isThrottleError(err error) bool {
	return errors.Is(err, getxapi.ErrPostingThrottled)
}

// skipReason returns a short, stable phrase describing why a mention could not
// be answered, so the log line stays scannable.
func skipReason(err error) string {
	switch {
	case errors.Is(err, errNoVideo):
		return "no_video"
	case errors.Is(err, errParentUnavailable):
		return "parent_unavailable"
	case errors.Is(err, errDownloadLinkFailed):
		return "download_link_failed"
	case errors.Is(err, getxapi.ErrDuplicateTweet):
		return "duplicate_reply"
	case errors.Is(err, getxapi.ErrReplyRestricted):
		return "reply_restricted"
	default:
		return getxapi.Reason(err)
	}
}

func (f *fetchMentions) getLastMentionID(key domain.KeySetting) (string, error) {
	setting, err := f.dynamodb.GetSetting(key)
	if err != nil {
		return "", err
	}

	if setting == nil {
		return "", nil
	}

	return setting.Value, nil
}

func (f *fetchMentions) getRepliedMentions(key domain.KeySetting) map[string]struct{} {
	replied := map[string]struct{}{}

	setting, err := f.dynamodb.GetSetting(key)
	if err != nil {
		slog.Error("Error getting replied mentions", slog.Any("error", err))
		return replied
	}

	if setting == nil || setting.Value == "" {
		return replied
	}

	var ids []string
	if err := json.Unmarshal([]byte(setting.Value), &ids); err != nil {
		slog.Error("Error unmarshalling replied mentions", slog.Any("error", err))
		return replied
	}

	for _, id := range ids {
		replied[id] = struct{}{}
	}

	return replied
}

func (f *fetchMentions) saveRepliedMentions(key domain.KeySetting, replied map[string]struct{}) {
	ids := make([]string, 0, len(replied))
	for id := range replied {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Cap the list so the settings item stays small.
	if len(ids) > maxRepliedMentions {
		ids = ids[len(ids)-maxRepliedMentions:]
	}

	body, err := json.Marshal(ids)
	if err != nil {
		slog.Error("Error marshalling replied mentions", slog.Any("error", err))
		return
	}

	if _, err := f.dynamodb.SaveSetting(&domain.Settings{
		KeySetting: string(key),
		Value:      string(body),
	}); err != nil {
		slog.Error("Error saving replied mentions", slog.Any("error", err))
	}
}

// isNewer compares two tweet snowflake ids numerically. Ids are digit strings
// of monotonically increasing value, so a longer id is always newer.
func isNewer(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)

	if left == "" {
		return false
	}

	if right == "" {
		return true
	}

	if len(left) != len(right) {
		return len(left) > len(right)
	}

	return left > right
}

func maxID(a, b string) string {
	if isNewer(b, a) {
		return b
	}

	return a
}
