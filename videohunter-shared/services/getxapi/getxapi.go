package getxapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/victoraldir/myvideohuntershared/domain"
	"github.com/victoraldir/myvideohuntershared/services"
)

const (
	scheme = "https"
	host   = "api.getxapi.com"
)

// Error sentinels returned by the service so callers can decide whether a
// mention should be retried later or permanently skipped.
var (
	ErrDuplicateTweet   = errors.New("getxapi: status is a duplicate")
	ErrReplyRestricted  = errors.New("getxapi: reply restricted by author")
	ErrPostingThrottled = errors.New("getxapi: posting temporarily limited")
	ErrUnavailable      = errors.New("getxapi: service unavailable")
	ErrTweetNotFound    = errors.New("getxapi: tweet not found")
)

// Reason returns a short, stable phrase describing an API failure, suitable for
// logs and metrics. Error strings from the API can vary, so key off this.
func Reason(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrDuplicateTweet):
		return "duplicate_reply"
	case errors.Is(err, ErrReplyRestricted):
		return "reply_restricted"
	case errors.Is(err, ErrPostingThrottled):
		return "posting_throttled"
	case errors.Is(err, ErrUnavailable):
		return "service_unavailable"
	case errors.Is(err, ErrTweetNotFound):
		return "tweet_not_found"
	default:
		return "error"
	}
}

// GetXAPIService is the subset of the GetXAPI Twitter/X API used by the bots.
type GetXAPIService interface {
	SearchMentions(botUsername, cursor string) (*domain.TweetSearchResponse, error)
	GetTweetDetail(id string) (*domain.Tweet, error)
	CreateReply(authToken, replyToTweetID, text string) (string, error)
}

type getXAPIService struct {
	client services.HttpClient
	apiKey string
}

func NewGetXAPIService(client services.HttpClient, apiKey string) *getXAPIService {
	return &getXAPIService{
		client: client,
		apiKey: apiKey,
	}
}

// SearchMentions returns the latest tweets mentioning botUsername. Results are
// returned newest-first, ~20 per page. Pass an empty cursor for the first page.
func (g *getXAPIService) SearchMentions(botUsername, cursor string) (*domain.TweetSearchResponse, error) {
	query := fmt.Sprintf("@%s", strings.TrimPrefix(botUsername, "@"))

	values := url.Values{
		"q":       []string{query},
		"product": []string{"Latest"},
	}
	if cursor != "" {
		values.Set("cursor", cursor)
	}

	req, err := g.newRequest(http.MethodGet, "/twitter/tweet/advanced_search", values, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		slog.Debug("Error searching mentions", slog.Any("error", err))
		return nil, fmt.Errorf("searching mentions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, g.decodeError(resp)
	}

	searchResponse := domain.TweetSearchResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		slog.Debug("Error decoding search response", slog.Any("error", err))
		return nil, err
	}

	return &searchResponse, nil
}

// GetTweetDetail returns a single tweet with its canonical URL and author.
func (g *getXAPIService) GetTweetDetail(id string) (*domain.Tweet, error) {
	req, err := g.newRequest(http.MethodGet, "/twitter/tweet/detail", url.Values{"id": []string{id}}, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		slog.Debug("Error getting tweet detail", slog.Any("error", err))
		return nil, fmt.Errorf("getting tweet detail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, g.decodeError(resp)
	}

	detail := tweetDetailResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		slog.Debug("Error decoding tweet detail", slog.Any("error", err))
		return nil, err
	}

	return &detail.Data, nil
}

// CreateReply posts a reply from the account owning authToken and returns the
// id of the created tweet.
func (g *getXAPIService) CreateReply(authToken, replyToTweetID, text string) (string, error) {
	body, err := json.Marshal(map[string]string{
		"auth_token":        authToken,
		"text":              text,
		"reply_to_tweet_id": replyToTweetID,
	})
	if err != nil {
		return "", err
	}

	req, err := g.newRequest(http.MethodPost, "/twitter/tweet/create", nil, body)
	if err != nil {
		return "", err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		slog.Debug("Error creating reply", slog.Any("error", err))
		return "", fmt.Errorf("creating reply: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", g.decodeError(resp)
	}

	createResponse := createTweetResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&createResponse); err != nil {
		slog.Debug("Error decoding create tweet response", slog.Any("error", err))
		return "", err
	}

	return createResponse.Data.ID, nil
}

func (g *getXAPIService) newRequest(method, path string, values url.Values, body []byte) (*http.Request, error) {
	target := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}
	if len(values) > 0 {
		target.RawQuery = values.Encode()
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, target.String(), reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.apiKey))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// decodeError reads the GetXAPI error body and maps it to a sentinel error when
// the failure is one the bot knows how to handle.
func (g *getXAPIService) decodeError(resp *http.Response) error {
	apiErr := apiError{}
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		_ = json.Unmarshal(body, &apiErr)
	}
	if apiErr.Message == "" {
		apiErr.Message = strings.TrimSpace(string(body))
	}

	switch {
	case resp.StatusCode == http.StatusConflict || apiErr.TwitterErrorCode == 187:
		return fmt.Errorf("%w: %s", ErrDuplicateTweet, apiErr.Message)
	case apiErr.TwitterErrorCode == 433:
		return fmt.Errorf("%w: %s", ErrReplyRestricted, apiErr.Message)
	case resp.StatusCode == http.StatusTooManyRequests || apiErr.TwitterErrorCode == 344:
		return fmt.Errorf("%w: %s", ErrPostingThrottled, apiErr.Message)
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrTweetNotFound, apiErr.Message)
	case resp.StatusCode >= http.StatusInternalServerError:
		return fmt.Errorf("%w: %s", ErrUnavailable, apiErr.Message)
	default:
		return fmt.Errorf("getxapi returned status %d: %s", resp.StatusCode, apiErr.Message)
	}
}
