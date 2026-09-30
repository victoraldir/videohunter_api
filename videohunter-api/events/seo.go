package events

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// canonicalBaseURL is the canonical origin of the website. The live site
	// currently redirects the apex domain to www (CloudFront function), so www
	// is the canonical host. Change this single constant if that ever flips.
	canonicalBaseURL = "https://www.myvideohunter.com"
	// maxTitleText is the maximum amount of video text included in the page title.
	maxTitleText = 70
	// maxDescription is the maximum length of the meta description.
	maxDescription = 155
)

// Platform returns a human readable name of the platform the video comes from.
func (v *GetVideoResponse) Platform() string {

	host := hostFromURL(v.OriginalVideoUrl)
	lowerURL := strings.ToLower(v.OriginalVideoUrl)

	switch {
	case host == "x.com" || host == "twitter.com" ||
		strings.HasSuffix(host, ".x.com") || strings.HasSuffix(host, ".twitter.com"):
		return "X (Twitter)"
	case host == "v.redd.it" || strings.HasSuffix(host, "reddit.com"):
		return "Reddit"
	case strings.HasSuffix(host, "bsky.app") || strings.Contains(lowerURL, "bsky") ||
		strings.HasPrefix(lowerURL, "at://"):
		return "Bluesky"
	}

	return "social media"
}

// PageTitle returns a unique, keyword oriented title for the video page.
func (v *GetVideoResponse) PageTitle() string {

	text := collapseWhitespace(v.Text)

	if text == "" {
		return fmt.Sprintf("Download %s video online — Video Hunter", v.Platform())
	}

	return fmt.Sprintf("Download %s video: %s — Video Hunter", v.Platform(), truncateText(text, maxTitleText))
}

// PageDescription returns a unique meta description for the video page.
func (v *GetVideoResponse) PageDescription() string {

	const suffix = " Download it free in HD with Video Hunter."

	text := collapseWhitespace(v.Text)

	if text == "" {
		return fmt.Sprintf("Download this %s video online for free. "+
			"Paste any X (Twitter), Reddit or Bluesky link into Video Hunter and save the video in seconds.", v.Platform())
	}

	return truncateText(text, maxDescription-len(suffix)) + suffix
}

// CanonicalURL returns the canonical URL of this video page.
func (v *GetVideoResponse) CanonicalURL() string {
	return canonicalBaseURL + "/prod/url/" + v.Id
}

// PlatformPageURL returns the landing page for the platform the video comes
// from, so video pages can link to it.
func (v *GetVideoResponse) PlatformPageURL() string {

	switch v.Platform() {
	case "X (Twitter)":
		return "/x-video-downloader.html"
	case "Reddit":
		return "/reddit-video-downloader.html"
	case "Bluesky":
		return "/bluesky-video-downloader.html"
	}

	return "/"
}

// HostFromURL returns the lowercased host of the given URL, or an empty
// string when it cannot be determined.
func hostFromURL(rawURL string) string {

	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Host)
	}

	// Fallback for values without a scheme, e.g. "at://did:plc:...".
	for _, part := range strings.Split(rawURL, "/") {
		if strings.Contains(part, ".") && !strings.Contains(part, ":") {
			return strings.ToLower(part)
		}
	}

	return ""
}

func collapseWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func truncateText(value string, max int) string {

	runes := []rune(value)
	if len(runes) <= max {
		return value
	}

	cut := string(runes[:max])

	if idx := strings.LastIndex(cut, " "); idx > max/2 {
		cut = cut[:idx]
	}

	return strings.TrimSpace(cut) + "…"
}
