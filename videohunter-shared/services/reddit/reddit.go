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
	"regexp"
	"strings"
	"sync"
	"time"

	shared_domain "github.com/victoraldir/myvideohuntershared/domain"
)

const (
	// userAgent identifies the service to Reddit. Reddit answers the default
	// Go user agent with an error status.
	userAgent = "VideoHunter/1.0 (+https://www.myvideohunter.com)"
	// browserUserAgent is used for the share link pages only: the challenge
	// flow mirrors what a browser does with those pages.
	browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
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

// BlockedShareLinkError reports that Reddit answered the share link fetch
// with its block page instead of the share page. Reddit serves the share
// pages of /s/ links to residential IPs but blocks this service's IP range on
// every host, so a share link has to be opened in a browser to find the post
// it leads to. Retrying will not help.
type BlockedShareLinkError struct {
	StatusCode int
	Err        error
}

func (b *BlockedShareLinkError) Error() string {
	return b.Err.Error()
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
		// Share links stopped redirecting upstream: they now serve an
		// anti-bot page that a browser solves and submits. ResolveShareUrl
		// performs those steps and returns the post permalink.
		resolved, err := r.ResolveShareUrl(url)
		if err != nil {
			return "", err
		}

		url = resolved
	}

	splitUrlQuery := strings.Split(url, "?")

	url = splitUrlQuery[0]

	if url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	urlWithExtension := url + ".json"

	return urlWithExtension, nil
}

// --- Share links ---------------------------------------------------------

const (
	// challengeMarker identifies the anti-bot page served for /s/ share
	// links instead of a redirect.
	challengeMarker = `name="js_challenge"`
	// wwwHost is where share pages and post pages live; only data API reads
	// move to the OAuth host.
	wwwHost = "https://www.reddit.com"
)

var (
	challengeScriptRe   = regexp.MustCompile(`(?s)<script[^>]*>(.*?)</script>`)
	challengeSolutionRe = regexp.MustCompile(`\(\s*async\s*(\w+)\s*=>\s*(\w+)\s*\+\s*(\w+)\s*\)\s*\(\s*"([0-9a-f]+)"\s*\)`)
	shareFormActionRe   = regexp.MustCompile(`<form[^>]*\saction="([^"]*)"`)
	shareFormTokenRe    = regexp.MustCompile(`name="jsc_token"[^>]*value="([^"]*)"`)
	shareFormOrigRRe    = regexp.MustCompile(`name="jsc_orig_r"[^>]*value="([^"]*)"`)
	sharePermalinkRe    = regexp.MustCompile(`(/r/[A-Za-z0-9_%]+/comments/([a-z0-9]+)/[A-Za-z0-9_%]*)`)
)

// ResolveShareUrl turns a /s/ share link into the post permalink.
//
// Reddit stopped redirecting share links: they now serve an anti-bot page
// whose script solves a small challenge and submits a hidden form, and only
// then is the post served. The same steps work from here: fetch the page,
// submit the solved form with the page's cookies, and read the post permalink
// out of the response. An unknown or expired share id lands on the subreddit
// feed instead, which the permalink extraction treats as an unresolved link.
//
// The www host answers the Lambda's IP with a block page, so with credentials
// the share pages are fetched from the OAuth host, like the data API reads.
func (r *redditDownloaderRepository) ResolveShareUrl(shareUrl string) (string, error) {

	authorization, err := r.GetAuthToken()
	if err != nil {
		return "", err
	}

	if authorization != "" {
		shareUrl = withOauthHost(shareUrl)
	}

	page, err := r.fetchSharePage(shareUrl, "", "", authorization)
	if err != nil {
		return "", err
	}

	if page.location != "" {
		return absoluteUrl(page.location, shareUrl), nil
	}

	if page.statusCode == http.StatusOK && strings.Contains(page.body, challengeMarker) {
		submitUrl, err := solveShareChallenge(shareUrl, page)
		if err != nil {
			return "", err
		}

		page, err = r.fetchSharePage(submitUrl, shareUrl, strings.Join(page.cookies, "; "), authorization)
		if err != nil {
			return "", err
		}

		if page.location != "" {
			return absoluteUrl(page.location, shareUrl), nil
		}
	}

	if page.statusCode != http.StatusOK {
		switch page.statusCode {
		case http.StatusNotFound:
			return "", &InvalidPostError{StatusCode: page.statusCode, Err: fmt.Errorf("share link not found")}
		case http.StatusForbidden, http.StatusUnauthorized:
			return "", &BlockedShareLinkError{StatusCode: page.statusCode, Err: fmt.Errorf("share link fetch returned status %d", page.statusCode)}
		default:
			return "", &UpstreamError{StatusCode: page.statusCode, Err: fmt.Errorf("share link fetch returned status %d", page.statusCode)}
		}
	}

	return extractSharePermalink(page.body)
}

// sharePage is one fetched page of the share link resolution.
type sharePage struct {
	body       string
	location   string
	statusCode int
	cookies    []string
}

// fetchSharePage fetches one page of the share link resolution. A Referer is
// only sent for the challenge submission, which is the request a browser
// makes from the challenge page; cookies are passed through so the challenge
// is validated by the edge that issued it; the authorization carries the
// app's token on the OAuth host.
func (r *redditDownloaderRepository) fetchSharePage(target, referer, cookie, authorization string) (*sharePage, error) {

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html")

	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	page := &sharePage{
		body:       string(content),
		location:   resp.Header.Get("Location"),
		statusCode: resp.StatusCode,
	}

	for _, c := range resp.Cookies() {
		page.cookies = append(page.cookies, c.Name+"="+c.Value)
	}

	return page, nil
}

// solveShareChallenge builds the form submission the challenge page's script
// performs in a browser: the solution is the challenge token concatenated
// with itself, and the form carries its own hidden fields plus the share
// link's query parameters.
func solveShareChallenge(shareUrl string, page *sharePage) (string, error) {

	action := shareFormActionRe.FindStringSubmatch(page.body)
	if action == nil {
		return "", &UpstreamError{Err: errors.New("no share challenge form found")}
	}

	solution, ok := solveChallengeSolution(page.body)
	if !ok {
		return "", &UpstreamError{Err: errors.New("unsupported reddit share challenge")}
	}

	var token, origR string
	if m := shareFormTokenRe.FindStringSubmatch(page.body); m != nil {
		token = m[1]
	}
	if m := shareFormOrigRRe.FindStringSubmatch(page.body); m != nil {
		origR = m[1]
	}

	submitUrl, err := url.Parse(absoluteUrl(action[1], shareUrl))
	if err != nil {
		return "", err
	}

	query := submitUrl.Query()
	query.Set("solution", solution)
	query.Set("js_challenge", "1")
	query.Set("jsc_token", token)
	query.Set("jsc_orig_r", origR)

	// The served script forwards the share link's own query parameters
	// (share_id and friends) into the submission.
	if shared, err := url.Parse(shareUrl); err == nil {
		for key, values := range shared.Query() {
			for _, value := range values {
				query.Set(key, value)
			}
		}
	}

	submitUrl.RawQuery = query.Encode()

	return submitUrl.String(), nil
}

// solveChallengeSolution reads the challenge out of the served script. Only
// the self concatenation the script performs is understood; a different
// challenge is a Reddit change, not a problem with the link.
func solveChallengeSolution(body string) (string, bool) {

	for _, script := range challengeScriptRe.FindAllStringSubmatch(body, -1) {
		m := challengeSolutionRe.FindStringSubmatch(script[1])
		if m == nil {
			continue
		}

		if m[1] == m[2] && m[2] == m[3] {
			return m[4] + m[4], true
		}
	}

	return "", false
}

// extractSharePermalink finds the post permalink on a resolved share page. A
// resolved page is a post: every permalink on it is the same post. An
// unresolved share id lands on the subreddit feed, whose permalinks are many
// different posts, so anything other than exactly one post id is an
// unresolved link.
func extractSharePermalink(body string) (string, error) {

	permalinks := make(map[string]string)
	for _, m := range sharePermalinkRe.FindAllStringSubmatch(body, -1) {
		// The post permalink is what a page renders first; a comment
		// permalink to the same post must not replace it.
		if _, seen := permalinks[m[2]]; !seen {
			permalinks[m[2]] = m[1]
		}
	}

	if len(permalinks) != 1 {
		return "", &InvalidPostError{StatusCode: http.StatusNotFound, Err: fmt.Errorf("share link does not lead to a post")}
	}

	var permalink string
	for _, p := range permalinks {
		permalink = p
	}

	return wwwHost + permalink, nil
}

// absoluteUrl resolves a Location header value against the URL it came from.
func absoluteUrl(location, base string) string {

	parsed, err := url.Parse(location)
	if err != nil || parsed.IsAbs() {
		return location
	}

	baseUrl, err := url.Parse(base)
	if err != nil {
		return location
	}

	return baseUrl.ResolveReference(parsed).String()
}
