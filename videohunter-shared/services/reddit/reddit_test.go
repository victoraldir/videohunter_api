package reddit

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const videoListingFixture = `[
  {
    "kind": "Listing",
    "data": {
      "children": [
        {
          "kind": "t3",
          "data": {
            "id": "1kxtryc",
            "title": "boa criacao e de berco",
            "is_video": true,
            "thumbnail": "https://b.thumbs.redditmedia.com/x.jpg",
            "secure_media": {
              "reddit_video": {
                "bitrate_kbps": 2400,
                "hls_url": "https://v.redd.it/zg80stewgl3f1/HLSPlaylist.m3u8?a=1&v=1"
              }
            }
          }
        }
      ]
    }
  }
]`

const tokenFixture = `{"access_token": "token-abc", "token_type": "bearer", "expires_in": 3600}`

type stubClient struct {
	responses []*http.Response
	errs      []error
	requests  []*http.Request
}

func (c *stubClient) Do(req *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, req)

	i := len(c.requests) - 1

	if i < len(c.errs) && c.errs[i] != nil {
		return nil, c.errs[i]
	}

	if i < len(c.responses) {
		return c.responses[i], nil
	}

	if len(c.responses) > 0 {
		return c.responses[len(c.responses)-1], nil
	}

	return nil, errors.New("stub: no response configured")
}

func newResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func withFastRetries(t *testing.T) {
	t.Helper()
	original := retryBackoff
	retryBackoff = 0
	t.Cleanup(func() { retryBackoff = original })
}

func TestDownloadVideo_ParsesVideo(t *testing.T) {

	client := &stubClient{responses: []*http.Response{newResponse(http.StatusOK, videoListingFixture)}}
	repository := NewRedditDownloaderRepository(client)

	postUrl := "https://www.reddit.com/r/botecodoreddit/comments/1kxtryc/boa_criacao/"

	video, _, err := repository.DownloadVideo(postUrl)

	assert.NoError(t, err)
	assert.NotNil(t, video)
	assert.Equal(t, "1kxtryc", video.OriginalId)
	assert.Equal(t, "boa criacao e de berco", video.Text)
	// The source link must stay the post the user pasted, not the JSON endpoint.
	assert.Equal(t, postUrl, video.OriginalVideoUrl)
	assert.Len(t, video.ExtendedEntities.Media, 1)
	assert.Equal(t, "https://v.redd.it/zg80stewgl3f1/HLSPlaylist.m3u8?a=1&v=1",
		video.ExtendedEntities.Media[0].VideoInfo.Variants[0].URL)
}

func TestDownloadVideo_MapsNotFoundToInvalidPostError(t *testing.T) {

	client := &stubClient{responses: []*http.Response{newResponse(http.StatusNotFound, "")}}
	repository := NewRedditDownloaderRepository(client)

	_, _, err := repository.DownloadVideo("https://www.reddit.com/r/videos/comments/1abcxyz/test/")

	var invalidPost *InvalidPostError
	assert.ErrorAs(t, err, &invalidPost)
	assert.Equal(t, 404, invalidPost.StatusCode)
	// A 404 is a final answer: no retry.
	assert.Len(t, client.requests, 1)
}

func TestDownloadVideo_MapsBlockedToUpstreamError(t *testing.T) {

	client := &stubClient{responses: []*http.Response{newResponse(http.StatusForbidden, "")}}
	repository := NewRedditDownloaderRepository(client)

	_, _, err := repository.DownloadVideo("https://www.reddit.com/r/videos/comments/1abcxyz/test/")

	var upstream *UpstreamError
	assert.ErrorAs(t, err, &upstream)
	assert.Equal(t, http.StatusForbidden, upstream.StatusCode)
	assert.Len(t, client.requests, 1)
}

func TestDownloadVideo_RetriesRateLimitThenSucceeds(t *testing.T) {

	withFastRetries(t)

	client := &stubClient{responses: []*http.Response{
		newResponse(http.StatusTooManyRequests, ""),
		newResponse(http.StatusOK, videoListingFixture),
	}}
	repository := NewRedditDownloaderRepository(client)

	video, _, err := repository.DownloadVideo("https://www.reddit.com/r/videos/comments/1abcxyz/test/")

	assert.NoError(t, err)
	assert.NotNil(t, video)
	assert.Len(t, client.requests, 2)
}

func TestDownloadVideo_GivesUpAfterRetries(t *testing.T) {

	withFastRetries(t)

	client := &stubClient{responses: []*http.Response{newResponse(http.StatusInternalServerError, "")}}
	repository := NewRedditDownloaderRepository(client)

	_, _, err := repository.DownloadVideo("https://www.reddit.com/r/videos/comments/1abcxyz/test/")

	var upstream *UpstreamError
	assert.ErrorAs(t, err, &upstream)
	assert.Len(t, client.requests, requestAttempts)
}

func TestDownloadVideo_RejectsPostWithoutVideo(t *testing.T) {

	client := &stubClient{responses: []*http.Response{
		newResponse(http.StatusOK, `[{"kind":"Listing","data":{"children":[{"kind":"t3","data":{"id":"abc","title":"text post","is_video":false}}]}}]`),
	}}
	repository := NewRedditDownloaderRepository(client)

	_, _, err := repository.DownloadVideo("https://www.reddit.com/r/videos/comments/1abcxyz/test/")

	var invalidPost *InvalidPostError
	assert.ErrorAs(t, err, &invalidPost)
}

func TestDownloadVideo_IdentifiesItselfAndSkipsPlaceholderCredentials(t *testing.T) {

	t.Setenv("REDDIT_CLIENT_ID", "dummy")
	t.Setenv("REDDIT_CLIENT_SECRET", "dummy")

	client := &stubClient{responses: []*http.Response{newResponse(http.StatusOK, videoListingFixture)}}
	repository := NewRedditDownloaderRepository(client)

	_, _, err := repository.DownloadVideo("https://www.reddit.com/r/videos/comments/1abcxyz/test/")

	assert.NoError(t, err)
	assert.Len(t, client.requests, 1)
	assert.Equal(t, userAgent, client.requests[0].Header.Get("User-Agent"))
	assert.Empty(t, client.requests[0].Header.Get("Authorization"))
	// Without credentials the read stays on the pasted host, unauthenticated.
	assert.Equal(t, "www.reddit.com", client.requests[0].URL.Host)
}

func TestDownloadVideo_UsesCredentialsWhenConfigured(t *testing.T) {

	t.Setenv("REDDIT_CLIENT_ID", "real-id")
	t.Setenv("REDDIT_CLIENT_SECRET", "real-secret")

	client := &stubClient{responses: []*http.Response{
		newResponse(http.StatusOK, tokenFixture),
		newResponse(http.StatusOK, videoListingFixture),
	}}
	repository := NewRedditDownloaderRepository(client)

	_, _, err := repository.DownloadVideo("https://www.reddit.com/r/videos/comments/1abcxyz/test/")

	assert.NoError(t, err)
	assert.Len(t, client.requests, 2)

	// First the app credentials are exchanged for a bearer token.
	tokenReq := client.requests[0]
	assert.Equal(t, http.MethodPost, tokenReq.Method)
	assert.Equal(t, tokenUrl, tokenReq.URL.String())
	assert.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("real-id:real-secret")),
		tokenReq.Header.Get("Authorization"))

	grantBody, readErr := io.ReadAll(tokenReq.Body)
	assert.NoError(t, readErr)
	assert.Equal(t, "grant_type=client_credentials", string(grantBody))

	// Then the post is read from the OAuth host with the bearer token.
	readReq := client.requests[1]
	assert.Equal(t, http.MethodGet, readReq.Method)
	assert.Equal(t, oauthHost, readReq.URL.Host)
	assert.Equal(t, "/r/videos/comments/1abcxyz/test.json", readReq.URL.Path)
	assert.Equal(t, "Bearer token-abc", readReq.Header.Get("Authorization"))
	assert.Equal(t, userAgent, readReq.Header.Get("User-Agent"))
}

func TestDownloadVideo_CachesTokenBetweenReads(t *testing.T) {

	t.Setenv("REDDIT_CLIENT_ID", "real-id")
	t.Setenv("REDDIT_CLIENT_SECRET", "real-secret")

	client := &stubClient{responses: []*http.Response{
		newResponse(http.StatusOK, tokenFixture),
		newResponse(http.StatusOK, videoListingFixture),
		newResponse(http.StatusOK, videoListingFixture),
	}}
	repository := NewRedditDownloaderRepository(client)

	postUrl := "https://www.reddit.com/r/videos/comments/1abcxyz/test/"

	_, _, err := repository.DownloadVideo(postUrl)
	assert.NoError(t, err)

	_, _, err = repository.DownloadVideo(postUrl)
	assert.NoError(t, err)

	// The second read reuses the cached token: no second token exchange.
	assert.Len(t, client.requests, 3)
	assert.Equal(t, "Bearer token-abc", client.requests[2].Header.Get("Authorization"))
	assert.Equal(t, oauthHost, client.requests[2].URL.Host)
}

func TestDownloadVideo_PropagatesTokenEndpointFailure(t *testing.T) {

	t.Setenv("REDDIT_CLIENT_ID", "real-id")
	t.Setenv("REDDIT_CLIENT_SECRET", "real-secret")

	client := &stubClient{responses: []*http.Response{newResponse(http.StatusBadRequest, "")}}
	repository := NewRedditDownloaderRepository(client)

	_, _, err := repository.DownloadVideo("https://www.reddit.com/r/videos/comments/1abcxyz/test/")

	assert.Error(t, err)
	assert.Len(t, client.requests, 1)
}

func TestGetJsonUrl(t *testing.T) {

	repository := NewRedditDownloaderRepository(&stubClient{})

	t.Run("builds the json endpoint and drops query and trailing slash", func(t *testing.T) {
		got, err := repository.GetJsonUrl("https://www.reddit.com/r/videos/comments/1abcxyz/test/?utm_source=share")
		assert.NoError(t, err)
		assert.Equal(t, "https://www.reddit.com/r/videos/comments/1abcxyz/test.json", got)
	})

	t.Run("rejects a url that is not a post instead of panicking", func(t *testing.T) {
		_, err := repository.GetJsonUrl("https://www.reddit.com/r/videos")
		assert.Error(t, err)
	})
}
