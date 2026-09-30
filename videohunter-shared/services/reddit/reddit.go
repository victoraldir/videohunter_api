package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
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

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type redditDownloaderRepository struct {
	client HttpClient
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

	req, err := http.NewRequest("GET", jsonUrl, nil)
	if err != nil {
		slog.Error("Error creating request", "error", err)
		return nil, nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	basicAuth, err := r.GetAuthToken()
	if err != nil {
		slog.Error("Error getting auth token", "error", err)
		return nil, nil, err
	}

	if basicAuth != "" {
		req.Header.Set("Authorization", basicAuth)
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

// GetAuthToken returns the Authorization header for Reddit. Public posts do not
// need one, and placeholder credentials make Reddit reject the request, so the
// header is only produced when real credentials are configured.
func (r *redditDownloaderRepository) GetAuthToken() (authToken string, err error) {

	redditClientId := os.Getenv("REDDIT_CLIENT_ID")
	redditClientSecret := os.Getenv("REDDIT_CLIENT_SECRET")

	if isPlaceholder(redditClientId) || isPlaceholder(redditClientSecret) {
		return "", nil
	}

	return "Basic " + redditClientId + ":" + redditClientSecret, nil
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

		// Set Authorization header
		basicAuth, _ := r.GetAuthToken()

		if basicAuth != "" {
			req.Header.Set("Authorization", basicAuth)
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
