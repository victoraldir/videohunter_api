package reddit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	shared_domain "github.com/victoraldir/myvideohuntershared/domain"
)

const (
	// userAgent identifies the service to Reddit. Reddit answers the default
	// Go user agent with an error status.
	userAgent = "VideoHunter/1.0 (+https://www.myvideohunter.com)"
	// requestAttempts is how many times a rate limited or failed request is
	// retried before giving up.
	requestAttempts = 3
	// tokenUrl exchanges the app credentials for a bearer token. It is the
	// only request that keeps going to www.reddit.com once OAuth is in use.
	tokenUrl = "https://www.reddit.com/api/v1/access_token"
	// oauthHost is the host that serves the data API to authenticated apps.
	// Reddit answers unauthenticated JSON reads on the www host with 403,
	// whatever user agent or IP asks.
	oauthHost = "oauth.reddit.com"
	// tokenExpiryMargin is subtracted from the token lifetime so a read never
	// races the real expiry.
	tokenExpiryMargin = time.Minute
)

// retryBackoff is multiplied by the attempt number to space out retries. It is
// a variable so tests can run without waiting.
var retryBackoff = 500 * time.Millisecond

// InvalidPostError reports that Reddit answered, but the post cannot be
// downloaded: it does not exist, or it is not a video. Retrying will not help.
type InvalidPostError struct {
	StatusCode int
	Err        error
}

func (r *InvalidPostError) Error() string {
	return r.Err.Error()
}

// UpstreamError reports that Reddit could not be reached, or answered with an
// error of its own. The link may be perfectly valid, so this is worth retrying
// and must not be reported to the user as an invalid link.
type UpstreamError struct {
	StatusCode int
	Err        error
}

func (u *UpstreamError) Error() string {
	return u.Err.Error()
}

// tokenResponse is the payload of the client credentials grant.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type redditDownloaderRepository struct {
	client HttpClient

	// accessToken caches the client credentials token until shortly before
	// it expires, so a read costs one extra request only when the cache is
	// empty. A Lambda container serves many requests, so the cache pays off.
	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewRedditDownloaderRepository(client HttpClient) *redditDownloaderRepository {
	return &redditDownloaderRepository{
		client: client,
	}
}

func (r *redditDownloaderRepository) DownloadVideo(url string, authToken ...string) (videoDownload *shared_domain.Video, currentToken *string, err error) {

	originalUrl := url

	jsonUrl, err := r.GetJsonUrl(url)
	if err != nil {
		slog.Error("Error getting json url", "error", err)
		return nil, nil, err
	}

	authorization, err := r.GetAuthToken()
	if err != nil {
		slog.Error("Error getting auth token", "error", err)
		return nil, nil, err
	}

	if authorization != "" {
		// Authenticated reads go through the OAuth host; the www host
		// answers them with 403.
		jsonUrl = withOauthHost(jsonUrl)
	}

	req, err := http.NewRequest("GET", jsonUrl, nil)
	if err != nil {
		slog.Error("Error creating request", "error", err)
		return nil, nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	statusCode, content, err := r.fetch(req)
	if err != nil {
		slog.Error("Error making request", "status", statusCode, "error", err)
		return nil, nil, &UpstreamError{StatusCode: statusCode, Err: err}
	}

	if statusCode != http.StatusOK {
		if statusCode == http.StatusNotFound {
			return nil, nil, &InvalidPostError{StatusCode: statusCode, Err: fmt.Errorf("post not found")}
		}

		// Anything else (403 from a blocked IP, 429 rate limit, 5xx) says
		// nothing about the link the user pasted.
		return nil, nil, &UpstreamError{
			StatusCode: statusCode,
			Err:        fmt.Errorf("reddit returned status %d", statusCode),
		}
	}

	var posts []Post
	err = json.Unmarshal(content, &posts)
	if err != nil {
		slog.Error("Error unmarshalling json", "error", err)
		return nil, nil, err
	}

	var t3 ChildData

	for _, post := range posts {
		for _, c := range post.Data.Children {
			if c.Kind == "t3" {
				t3 = c.Data
				break
			}
		}
	}

	if t3.ID == "" {
		return nil, nil, &InvalidPostError{StatusCode: 404, Err: fmt.Errorf("invalid post")}
	}

	if !t3.IsVideo {
		return nil, nil, &InvalidPostError{StatusCode: 404, Err: fmt.Errorf("post is not a video")}
	}

	var redditMedia RedditVideo

	redditMedia = t3.SecureMedia.RedditVideo

	if redditMedia.HlsURL == "" {
		redditMedia = t3.Preview.RedditVideoPreview
	}

	if redditMedia.HlsURL == "" && len(t3.CrosspostParentList) > 0 {
		redditMedia = t3.CrosspostParentList[0].SecureMedia.RedditVideo
	}

	if redditMedia.HlsURL == "" {
		return nil, nil, &InvalidPostError{StatusCode: 404, Err: fmt.Errorf("no video found")}
	}

	tumb := strings.ReplaceAll(t3.Thumbnail, "&amp;", "&")

	video := shared_domain.Video{
		// Keep the link the user pasted: the JSON endpoint is an internal
		// detail and reads as raw JSON if it is shown as the source link.
		OriginalVideoUrl: originalUrl,
		OriginalId:       t3.ID,
		ThumbnailUrl:     tumb,
		CreatedAt:        time.Now().String(),
		Text:             t3.Title,
		ExtendedEntities: shared_domain.ExtendedEntities{
			Media: []shared_domain.Media{
				{
					MediaUrl: redditMedia.HlsURL,
					Type:     "video",
					VideoInfo: shared_domain.VideoInfo{
						Variants: []shared_domain.Variants{
							{
								Bitrate:     redditMedia.BitrateKbps,
								URL:         redditMedia.HlsURL,
								ContentType: "video/mp4",
							},
						},
					},
				},
			},
		},
	}

	return &video, nil, nil
}

// fetch performs the request, retrying rate limits, server errors and
// transport failures. It returns the last status code it saw together with the
// response body, or an error when no attempt produced a response.
func (r *redditDownloaderRepository) fetch(req *http.Request) (statusCode int, body []byte, err error) {

	var lastErr error

	for attempt := 1; attempt <= requestAttempts; attempt++ {

		if attempt > 1 {
			time.Sleep(retryBackoff * time.Duration(attempt-1))
		}

		resp, err := r.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		content, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return resp.StatusCode, content, nil
		}

		if readErr != nil {
			lastErr = readErr
		} else {
			lastErr = fmt.Errorf("reddit returned status %d", resp.StatusCode)
		}

		// Only rate limits and server errors are worth another attempt: a 403
		// or a 404 will answer the same way for this invocation.
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return resp.StatusCode, content, nil
		}
	}

	return 0, nil, lastErr
}

// GetAuthToken returns the Authorization header for Reddit. Reddit answers
// unauthenticated data API reads with 403, so with real app credentials the
// header is a bearer token from the client credentials grant, cached until
// shortly before it expires. With placeholder credentials no header is
// produced: reads stay on the www host and fail honestly as an upstream
// problem instead of pretending to be authenticated.
func (r *redditDownloaderRepository) GetAuthToken() (authToken string, err error) {

	redditClientId := os.Getenv("REDDIT_CLIENT_ID")
	redditClientSecret := os.Getenv("REDDIT_CLIENT_SECRET")

	if isPlaceholder(redditClientId) || isPlaceholder(redditClientSecret) {
		return "", nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.accessToken != "" && time.Now().Before(r.tokenExpiry) {
		return "Bearer " + r.accessToken, nil
	}

	req, err := http.NewRequest("POST", tokenUrl, strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	req.SetBasicAuth(redditClientId, redditClientSecret)

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Reddit token endpoint failed", "status", resp.StatusCode)
		return "", fmt.Errorf("reddit token endpoint returned status %d", resp.StatusCode)
	}

	var token tokenResponse
	if err := json.Unmarshal(content, &token); err != nil {
		return "", err
	}

	if token.AccessToken == "" {
		return "", errors.New("reddit token endpoint returned no access token")
	}

	lifetime := time.Duration(token.ExpiresIn) * time.Second
	if lifetime > tokenExpiryMargin {
		lifetime -= tokenExpiryMargin
	} else {
		lifetime /= 2
	}

	r.accessToken = token.AccessToken
	r.tokenExpiry = time.Now().Add(lifetime)

	slog.Debug("Reddit access token refreshed", "expires_at", r.tokenExpiry)

	return "Bearer " + r.accessToken, nil
}

// withOauthHost moves a www.reddit.com JSON URL onto the OAuth host, which is
// the one that serves data API reads to authenticated apps. Reads without a
// token stay where they are and fail there honestly.
func withOauthHost(jsonUrl string) string {
	parsed, err := url.Parse(jsonUrl)
	if err != nil {
		return jsonUrl
	}

	parsed.Host = oauthHost

	return parsed.String()
}

func isPlaceholder(value string) bool {
	return value == "" || strings.EqualFold(value, "dummy")
}

func (r *redditDownloaderRepository) GetJsonUrl(url string) (string, error) {

	splitUrl := strings.Split(url, "/")

	if len(splitUrl) < 6 {
		return "", fmt.Errorf("not a reddit post url: %s", url)
	}

	// Check if url is short url
	if splitUrl[5] == "s" {
		// Get the Location header. We need to configure the client to not follow redirects
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		req, err := http.NewRequest("GET", url, nil)

		if err != nil {
			return "", err
		}

		req.Header.Set("User-Agent", userAgent)

		// Set Authorization header. The redirect endpoint does not need it,
		// but an authenticated short link resolution does no harm.
		authorization, _ := r.GetAuthToken()

		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}

		resp, err := client.Do(req)

		if err != nil {
			return "", err
		}

		defer resp.Body.Close()

		url = resp.Header.Get("Location")

		if url == "" {
			return "", fmt.Errorf("error getting location header")
		}
	}

	splitUrlQuery := strings.Split(url, "?")

	url = splitUrlQuery[0]

	if url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	urlWithExtension := url + ".json"

	return urlWithExtension, nil
}
