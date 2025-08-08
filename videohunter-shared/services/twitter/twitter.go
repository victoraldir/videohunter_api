package twitter

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/victoraldir/myvideohunterapi/utils"
	shared_domain "github.com/victoraldir/myvideohuntershared/domain"
)

const (
	crawlerUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	// Alternative scraping endpoints
	fxTwitterUrl = "https://api.fxtwitter.com/%s/status/%s"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type twitterDownloaderRepository struct {
	client HttpClient
}

// FxTwitterResponse represents the response from fxtwitter API
type FxTwitterResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Tweet   struct {
		URL    string `json:"url"`
		ID     string `json:"id"`
		Text   string `json:"text"`
		Author struct {
			Name     string `json:"name"`
			Username string `json:"screen_name"`
		} `json:"author"`
		Media struct {
			Videos []struct {
				URL       string `json:"url"`
				Type      string `json:"type"`
				Width     int    `json:"width"`
				Height    int    `json:"height"`
				Thumbnail string `json:"thumbnail"`
			} `json:"videos"`
			Photos []struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"photos"`
		} `json:"media"`
	} `json:"tweet"`
}

func NewTwitterDownloaderRepository(client HttpClient) *twitterDownloaderRepository {
	return &twitterDownloaderRepository{
		client: client,
	}
}

func (t *twitterDownloaderRepository) DownloadVideo(url string, authToken ...string) (videoDownload *shared_domain.Video, token *string, err error) {
	videoId := utils.GetVideoId(url)
	username := t.extractUsername(url)

	if username == "" {
		return nil, nil, fmt.Errorf("could not extract username from URL")
	}

	// Try fxtwitter API first
	video, err := t.tryFxTwitter(username, videoId)
	if err == nil && video != nil {
		return video, nil, nil
	}

	slog.Debug("fxtwitter failed, trying direct scraping", "error", err)

	// Fallback to direct scraping
	video, err = t.tryDirectScraping(url)
	if err != nil {
		return nil, nil, fmt.Errorf("all methods failed: %v", err)
	}

	return video, nil, nil
}

func (t *twitterDownloaderRepository) extractUsername(url string) string {
	// Extract username from URL like https://x.com/username/status/123
	re := regexp.MustCompile(`https?://(?:twitter\.com|x\.com)/([^/]+)/status/`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (t *twitterDownloaderRepository) tryFxTwitter(username, videoId string) (*shared_domain.Video, error) {
	apiUrl := fmt.Sprintf(fxTwitterUrl, username, videoId)

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", crawlerUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fxtwitter API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fxResp FxTwitterResponse
	err = json.Unmarshal(body, &fxResp)
	if err != nil {
		return nil, err
	}

	if fxResp.Code != 200 || len(fxResp.Tweet.Media.Videos) == 0 {
		return nil, fmt.Errorf("no video found in fxtwitter response")
	}

	// Convert to our domain model
	video := &shared_domain.Video{
		OriginalId:   videoId,
		Text:         fxResp.Tweet.Text,
		ThumbnailUrl: fxResp.Tweet.Media.Videos[0].Thumbnail,
		ExtendedEntities: shared_domain.ExtendedEntities{
			Media: []shared_domain.Media{
				{
					MediaUrl: fxResp.Tweet.Media.Videos[0].URL,
					Type:     "video",
					VideoInfo: shared_domain.VideoInfo{
						Variants: []shared_domain.Variants{
							{
								URL:         fxResp.Tweet.Media.Videos[0].URL,
								ContentType: "video/mp4",
							},
						},
					},
				},
			},
		},
	}

	return video, nil
}

func (t *twitterDownloaderRepository) tryDirectScraping(url string) (*shared_domain.Video, error) {
	// Convert x.com to twitter.com for better scraping compatibility
	scrapingUrl := strings.Replace(url, "x.com", "twitter.com", 1)

	req, err := http.NewRequest("GET", scrapingUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", crawlerUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("direct scraping returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	html := string(body)

	// Extract video URL using regex patterns
	videoUrl := t.extractVideoUrl(html)
	if videoUrl == "" {
		return nil, fmt.Errorf("no video URL found in HTML")
	}

	// Extract text content
	text := t.extractTweetText(html)

	// Extract thumbnail
	thumbnail := t.extractThumbnail(html)

	videoId := utils.GetVideoId(url)

	video := &shared_domain.Video{
		OriginalId:   videoId,
		Text:         text,
		ThumbnailUrl: thumbnail,
		ExtendedEntities: shared_domain.ExtendedEntities{
			Media: []shared_domain.Media{
				{
					MediaUrl: videoUrl,
					Type:     "video",
					VideoInfo: shared_domain.VideoInfo{
						Variants: []shared_domain.Variants{
							{
								URL:         videoUrl,
								ContentType: "video/mp4",
							},
						},
					},
				},
			},
		},
	}

	return video, nil
}

func (t *twitterDownloaderRepository) extractVideoUrl(html string) string {
	// Look for video URLs in various meta tags and script tags
	patterns := []string{
		`<meta property="og:video" content="([^"]+)"`,
		`<meta property="og:video:url" content="([^"]+)"`,
		`<meta name="twitter:player:stream" content="([^"]+)"`,
		`"video_url":"([^"]+)"`,
		`"playback_url":"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

func (t *twitterDownloaderRepository) extractTweetText(html string) string {
	// Extract tweet text from meta tags
	patterns := []string{
		`<meta property="og:description" content="([^"]+)"`,
		`<meta name="twitter:description" content="([^"]+)"`,
		`<meta name="description" content="([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

func (t *twitterDownloaderRepository) extractThumbnail(html string) string {
	// Extract thumbnail from meta tags
	patterns := []string{
		`<meta property="og:image" content="([^"]+)"`,
		`<meta name="twitter:image" content="([^"]+)"`,
		`"poster":"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}
