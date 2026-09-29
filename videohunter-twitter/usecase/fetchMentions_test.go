package usecase

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/victoraldir/myvideohuntershared/domain"
	"github.com/victoraldir/myvideohuntershared/services/getxapi"
	"github.com/victoraldir/myvideohuntershared/services/videohunterapi"
	"github.com/victoraldir/myvideohuntertwitter/repository/dynamodb"
)

const (
	botUsername = "BaixadorDeVideo"
	botToken    = "auth-token"
)

var (
	_ dynamodb.DynamodbRepository   = (*fakeSettingsRepository)(nil)
	_ getxapi.GetXAPIService        = (*fakeGetXAPIService)(nil)
	_ videohunterapi.VideoHunterApi = (*fakeVideoHunterApi)(nil)
)

type fakeSettingsRepository struct {
	settings map[domain.KeySetting]*domain.Settings
}

func newFakeSettingsRepository() *fakeSettingsRepository {
	return &fakeSettingsRepository{settings: map[domain.KeySetting]*domain.Settings{}}
}

func (f *fakeSettingsRepository) SaveSetting(setting *domain.Settings) (*domain.Settings, error) {
	f.settings[domain.KeySetting(setting.KeySetting)] = setting
	return setting, nil
}

func (f *fakeSettingsRepository) GetSetting(key domain.KeySetting) (*domain.Settings, error) {
	return f.settings[key], nil
}

type capturedReply struct {
	authToken string
	replyTo   string
	text      string
}

type fakeGetXAPIService struct {
	pages       []*domain.TweetSearchResponse
	searchCalls int
	searchErr   error

	details   map[string]*domain.Tweet
	detailErr map[string]error

	replies   []capturedReply
	createErr map[string]error
}

func (f *fakeGetXAPIService) SearchMentions(botUsername, cursor string) (*domain.TweetSearchResponse, error) {
	f.searchCalls++

	if f.searchErr != nil {
		return nil, f.searchErr
	}

	if f.searchCalls > len(f.pages) {
		return &domain.TweetSearchResponse{}, nil
	}

	return f.pages[f.searchCalls-1], nil
}

func (f *fakeGetXAPIService) GetTweetDetail(id string) (*domain.Tweet, error) {
	if err, ok := f.detailErr[id]; ok {
		return nil, err
	}

	if tweet, ok := f.details[id]; ok {
		return tweet, nil
	}

	return nil, errors.New("tweet not found")
}

func (f *fakeGetXAPIService) CreateReply(authToken, replyToTweetID, text string) (string, error) {
	if err, ok := f.createErr[replyToTweetID]; ok {
		return "", err
	}

	f.replies = append(f.replies, capturedReply{authToken: authToken, replyTo: replyToTweetID, text: text})
	return "created-reply-id", nil
}

type fakeVideoHunterApi struct {
	urls  map[string]domain.VideoUrl
	errs  map[string]error
	calls []string
}

func (f *fakeVideoHunterApi) DownloadVideo(videoUrl string) (domain.VideoUrl, error) {
	f.calls = append(f.calls, videoUrl)

	if err, ok := f.errs[videoUrl]; ok {
		return domain.VideoUrl{}, err
	}

	if url, ok := f.urls[videoUrl]; ok {
		return url, nil
	}

	return domain.VideoUrl{Id: "generated-id"}, nil
}

func mention(id, author, text string) domain.Tweet {
	return domain.Tweet{
		ID:     id,
		URL:    "https://x.com/" + author + "/status/" + id,
		Text:   text,
		Author: domain.TweetAuthor{UserName: author},
	}
}

func defaultVideoApi() *fakeVideoHunterApi {
	return &fakeVideoHunterApi{
		urls: map[string]domain.VideoUrl{
			"https://x.com/someone/status/999": {Id: "vid999"},
			"https://x.com/someone/status/888": {Id: "vid888"},
			"https://x.com/someone/status/777": {Id: "vid777"},
			"https://x.com/author/status/555":  {Id: "vid555"},
		},
	}
}

func watermarkKey() domain.KeySetting {
	return domain.TwitterBotSettingKey(domain.TwitterLastMentionID, botUsername)
}

func repliedMentionsKey() domain.KeySetting {
	return domain.TwitterBotSettingKey(domain.TwitterRepliedMentions, botUsername)
}

func TestFetchMentions_RepliesToMentionsNewestFirstProcessedOldestFirst(t *testing.T) {
	repo := newFakeSettingsRepository()
	service := &fakeGetXAPIService{pages: []*domain.TweetSearchResponse{{
		Tweets: []domain.Tweet{
			mention("300", "user2", "@BaixadorDeVideo https://x.com/someone/status/888"),
			mention("200", "user1", "@BaixadorDeVideo https://x.com/someone/status/999"),
		},
	}}}
	videoApi := defaultVideoApi()

	uc := NewFetchMentions(service, videoApi, repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	require.Len(t, service.replies, 2)

	assert.Equal(t, "200", service.replies[0].replyTo)
	assert.Contains(t, service.replies[0].text, "https://www.myvideohunter.com/prod/url/vid999")
	assert.Equal(t, "300", service.replies[1].replyTo)

	assert.Equal(t, botToken, service.replies[0].authToken)

	require.NotNil(t, repo.settings[watermarkKey()])
	assert.Equal(t, "300", repo.settings[watermarkKey()].Value)
}

func TestFetchMentions_PersistsRepliedMentions(t *testing.T) {
	repo := newFakeSettingsRepository()
	service := &fakeGetXAPIService{pages: []*domain.TweetSearchResponse{{
		Tweets: []domain.Tweet{
			mention("300", "user2", "@BaixadorDeVideo https://x.com/someone/status/888"),
			mention("200", "user1", "@BaixadorDeVideo https://x.com/someone/status/999"),
		},
	}}}

	uc := NewFetchMentions(service, defaultVideoApi(), repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	setting := repo.settings[repliedMentionsKey()]
	require.NotNil(t, setting)
	assert.Contains(t, setting.Value, "200")
	assert.Contains(t, setting.Value, "300")
}

func TestFetchMentions_StopsAtWatermark(t *testing.T) {
	repo := newFakeSettingsRepository()
	repo.settings[watermarkKey()] = &domain.Settings{KeySetting: string(watermarkKey()), Value: "250"}

	service := &fakeGetXAPIService{pages: []*domain.TweetSearchResponse{{
		Tweets: []domain.Tweet{
			mention("300", "user2", "@BaixadorDeVideo https://x.com/someone/status/888"),
			mention("200", "user1", "@BaixadorDeVideo https://x.com/someone/status/999"),
		},
	}}}
	videoApi := defaultVideoApi()

	uc := NewFetchMentions(service, videoApi, repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	require.Len(t, service.replies, 1)
	assert.Equal(t, "300", service.replies[0].replyTo)
	assert.Equal(t, "300", repo.settings[watermarkKey()].Value)
}

func TestFetchMentions_PaginatesUntilWatermark(t *testing.T) {
	repo := newFakeSettingsRepository()
	repo.settings[watermarkKey()] = &domain.Settings{KeySetting: string(watermarkKey()), Value: "100"}

	service := &fakeGetXAPIService{pages: []*domain.TweetSearchResponse{
		{
			HasMore:    true,
			NextCursor: "cursor-1",
			Tweets: []domain.Tweet{
				mention("300", "user2", "@BaixadorDeVideo https://x.com/someone/status/888"),
				mention("200", "user1", "@BaixadorDeVideo https://x.com/someone/status/999"),
			},
		},
		{
			HasMore: true,
			Tweets: []domain.Tweet{
				mention("150", "user3", "@BaixadorDeVideo https://x.com/someone/status/777"),
				mention("050", "user4", "@BaixadorDeVideo https://x.com/someone/status/777"),
			},
		},
	}}
	videoApi := defaultVideoApi()

	uc := NewFetchMentions(service, videoApi, repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	require.Len(t, service.replies, 3)
	assert.Equal(t, 2, service.searchCalls)
	assert.Equal(t, "300", repo.settings[watermarkKey()].Value)
}

func TestFetchMentions_SkipsAlreadyRepliedMentions(t *testing.T) {
	repo := newFakeSettingsRepository()
	repo.settings[repliedMentionsKey()] = &domain.Settings{KeySetting: string(repliedMentionsKey()), Value: `["300"]`}

	service := &fakeGetXAPIService{pages: []*domain.TweetSearchResponse{{
		Tweets: []domain.Tweet{
			mention("300", "user2", "@BaixadorDeVideo https://x.com/someone/status/888"),
		},
	}}}
	videoApi := defaultVideoApi()

	uc := NewFetchMentions(service, videoApi, repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	assert.Empty(t, service.replies)
	assert.Equal(t, "300", repo.settings[watermarkKey()].Value)
}

func TestFetchMentions_SkipsOwnTweets(t *testing.T) {
	repo := newFakeSettingsRepository()
	service := &fakeGetXAPIService{pages: []*domain.TweetSearchResponse{{
		Tweets: []domain.Tweet{
			mention("300", botUsername, "here is your video https://x.com/someone/status/888"),
		},
	}}}

	uc := NewFetchMentions(service, defaultVideoApi(), repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	assert.Empty(t, service.replies)
	assert.Equal(t, "300", repo.settings[watermarkKey()].Value)
}

func TestFetchMentions_ResolvesParentTweetForReplies(t *testing.T) {
	repo := newFakeSettingsRepository()
	service := &fakeGetXAPIService{
		pages: []*domain.TweetSearchResponse{{
			Tweets: []domain.Tweet{
				mention("300", "user2", "@BaixadorDeVideo"),
			},
		}},
		details: map[string]*domain.Tweet{
			"555": {ID: "555", URL: "https://x.com/author/status/555", Author: domain.TweetAuthor{UserName: "author"}},
		},
	}
	service.pages[0].Tweets[0].InReplyToID = "555"
	videoApi := defaultVideoApi()

	uc := NewFetchMentions(service, videoApi, repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	require.Len(t, videoApi.calls, 1)
	assert.Equal(t, "https://x.com/author/status/555", videoApi.calls[0])
	require.Len(t, service.replies, 1)
	assert.Contains(t, service.replies[0].text, "vid555")
}

func TestFetchMentions_SkipsMentionsWithoutVideo(t *testing.T) {
	repo := newFakeSettingsRepository()
	service := &fakeGetXAPIService{pages: []*domain.TweetSearchResponse{{
		Tweets: []domain.Tweet{
			mention("300", "user2", "@BaixadorDeVideo hello there"),
		},
	}}}

	uc := NewFetchMentions(service, defaultVideoApi(), repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	assert.Empty(t, service.replies)
	assert.Equal(t, "300", repo.settings[watermarkKey()].Value)
}

func TestFetchMentions_RetryableErrorStopsAndKeepsWatermark(t *testing.T) {
	repo := newFakeSettingsRepository()
	repo.settings[watermarkKey()] = &domain.Settings{KeySetting: string(watermarkKey()), Value: "100"}

	service := &fakeGetXAPIService{
		pages: []*domain.TweetSearchResponse{{
			Tweets: []domain.Tweet{
				mention("300", "user2", "@BaixadorDeVideo https://x.com/someone/status/888"),
				mention("200", "user1", "@BaixadorDeVideo https://x.com/someone/status/999"),
			},
		}},
		createErr: map[string]error{
			"200": getxapi.ErrPostingThrottled,
		},
	}

	uc := NewFetchMentions(service, defaultVideoApi(), repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	assert.Empty(t, service.replies)
	// Watermark must not advance past the failed mention.
	assert.Equal(t, "100", repo.settings[watermarkKey()].Value)
}

func TestFetchMentions_PermanentErrorSkipsAndAdvances(t *testing.T) {
	repo := newFakeSettingsRepository()

	service := &fakeGetXAPIService{
		pages: []*domain.TweetSearchResponse{{
			Tweets: []domain.Tweet{
				mention("300", "user2", "@BaixadorDeVideo https://x.com/someone/status/888"),
				mention("200", "user1", "@BaixadorDeVideo https://x.com/someone/status/999"),
			},
		}},
		createErr: map[string]error{
			"200": getxapi.ErrDuplicateTweet,
		},
	}

	uc := NewFetchMentions(service, defaultVideoApi(), repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	require.Len(t, service.replies, 1)
	assert.Equal(t, "300", service.replies[0].replyTo)
	assert.Equal(t, "300", repo.settings[watermarkKey()].Value)
}

func TestFetchMentions_ContinuesWithSecondBotAfterFirstFailure(t *testing.T) {
	repo := newFakeSettingsRepository()

	service := &fakeGetXAPIService{
		searchErr: getxapi.ErrUnavailable,
	}

	uc := NewFetchMentions(service, defaultVideoApi(), repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{
		{Username: botUsername, Language: "pt", AuthToken: botToken},
		{Username: "descargarvid", Language: "es", AuthToken: botToken},
	}})

	require.Error(t, err)
	assert.Equal(t, 2, service.searchCalls)
}

func TestFetchMentions_SkipsMentionWhenParentTweetIsGone(t *testing.T) {
	repo := newFakeSettingsRepository()

	reply := mention("300", "user2", "@BaixadorDeVideo")
	reply.InReplyToID = "555"

	service := &fakeGetXAPIService{
		pages: []*domain.TweetSearchResponse{{Tweets: []domain.Tweet{reply}}},
		// The parent tweet was deleted or is no longer visible to the bot.
		detailErr: map[string]error{"555": getxapi.ErrTweetNotFound},
	}

	uc := NewFetchMentions(service, defaultVideoApi(), repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	assert.Empty(t, service.replies)
	// A dead parent must not stall the queue.
	assert.Equal(t, "300", repo.settings[watermarkKey()].Value)
}

func TestFetchMentions_ParentServiceOutageIsRetried(t *testing.T) {
	repo := newFakeSettingsRepository()
	repo.settings[watermarkKey()] = &domain.Settings{KeySetting: string(watermarkKey()), Value: "100"}

	reply := mention("300", "user2", "@BaixadorDeVideo")
	reply.InReplyToID = "555"

	service := &fakeGetXAPIService{
		pages:     []*domain.TweetSearchResponse{{Tweets: []domain.Tweet{reply}}},
		detailErr: map[string]error{"555": getxapi.ErrUnavailable},
	}

	uc := NewFetchMentions(service, defaultVideoApi(), repo)

	err := uc.Execute(FetchMentionsRequest{Bots: []BotConfig{{Username: botUsername, Language: "pt", AuthToken: botToken}}})

	require.NoError(t, err)
	assert.Empty(t, service.replies)
	// A service outage is not permanent, so the watermark must not advance.
	assert.Equal(t, "100", repo.settings[watermarkKey()].Value)
}

func TestSkipReason(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected string
	}{
		{name: "no video", err: errNoVideo, expected: "no_video"},
		{name: "parent unavailable", err: fmt.Errorf("%w: gone", errParentUnavailable), expected: "parent_unavailable"},
		{name: "download link failed", err: fmt.Errorf("%w: 500", errDownloadLinkFailed), expected: "download_link_failed"},
		{name: "duplicate", err: getxapi.ErrDuplicateTweet, expected: "duplicate_reply"},
		{name: "restricted", err: getxapi.ErrReplyRestricted, expected: "reply_restricted"},
		{name: "throttled", err: getxapi.ErrPostingThrottled, expected: "posting_throttled"},
		{name: "unavailable", err: getxapi.ErrUnavailable, expected: "service_unavailable"},
		{name: "unknown", err: errors.New("boom"), expected: "error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, skipReason(tc.err))
		})
	}
}

func TestRunStats_LogSeverity(t *testing.T) {
	// Deferring work is the only case worth a warning.
	assert.False(t, runStats{replied: 3, skipped: 1}.needsAttention())
	assert.True(t, runStats{replied: 3, throttled: 1}.needsAttention())
	assert.True(t, runStats{failed: true}.needsAttention())
}
