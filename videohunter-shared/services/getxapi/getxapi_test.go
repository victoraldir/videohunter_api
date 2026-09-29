package getxapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (s stubClient) Do(req *http.Request) (*http.Response, error) {
	return s.do(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestGetXAPIService_SearchMentions(t *testing.T) {
	var captured *http.Request

	client := stubClient{do: func(req *http.Request) (*http.Response, error) {
		captured = req
		return jsonResponse(http.StatusOK, `{
			"query": "@BaixadorDeVideo",
			"tweet_count": 1,
			"has_more": true,
			"next_cursor": "CURSOR",
			"tweets": [{
				"type": "tweet",
				"id": "2015410770834591992",
				"url": "https://x.com/someuser/status/2015410770834591992",
				"text": "@BaixadorDeVideo https://x.com/other/status/123",
				"isReply": true,
				"inReplyToId": "100",
				"author": {"userName": "someuser", "id": "42"}
			}]
		}`), nil
	}}

	service := NewGetXAPIService(client, "API_KEY")

	response, err := service.SearchMentions("@BaixadorDeVideo", "")

	require.NoError(t, err)
	require.NotNil(t, response)

	assert.Equal(t, "Bearer API_KEY", captured.Header.Get("Authorization"))
	assert.Equal(t, "https", captured.URL.Scheme)
	assert.Equal(t, "api.getxapi.com", captured.URL.Host)
	assert.Equal(t, "/twitter/tweet/advanced_search", captured.URL.Path)
	assert.Equal(t, "@BaixadorDeVideo", captured.URL.Query().Get("q"))
	assert.Equal(t, "Latest", captured.URL.Query().Get("product"))
	assert.Empty(t, captured.URL.Query().Get("cursor"))

	assert.Len(t, response.Tweets, 1)
	assert.Equal(t, "2015410770834591992", response.Tweets[0].ID)
	assert.Equal(t, "someuser", response.Tweets[0].Author.UserName)
	assert.True(t, response.Tweets[0].IsReply)
	assert.Equal(t, "100", response.Tweets[0].InReplyToID)
	assert.True(t, response.HasMore)
}

func TestGetXAPIService_SearchMentions_WithCursor(t *testing.T) {
	var captured *http.Request

	client := stubClient{do: func(req *http.Request) (*http.Response, error) {
		captured = req
		return jsonResponse(http.StatusOK, `{"tweets": [], "has_more": false}`), nil
	}}

	service := NewGetXAPIService(client, "API_KEY")

	_, err := service.SearchMentions("BaixadorDeVideo", "NEXT")

	require.NoError(t, err)
	assert.Equal(t, "NEXT", captured.URL.Query().Get("cursor"))
}

func TestGetXAPIService_GetTweetDetail(t *testing.T) {
	client := stubClient{do: func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "2019264360682778716", req.URL.Query().Get("id"))
		return jsonResponse(http.StatusOK, `{
			"status": "success",
			"data": {
				"id": "2019264360682778716",
				"url": "https://x.com/elonmusk/status/2019264360682778716",
				"text": "video tweet",
				"author": {"userName": "elonmusk", "id": "44196397"}
			}
		}`), nil
	}}

	service := NewGetXAPIService(client, "API_KEY")

	tweet, err := service.GetTweetDetail("2019264360682778716")

	require.NoError(t, err)
	require.NotNil(t, tweet)
	assert.Equal(t, "https://x.com/elonmusk/status/2019264360682778716", tweet.URL)
	assert.Equal(t, "elonmusk", tweet.Author.UserName)
}

func TestGetXAPIService_CreateReply(t *testing.T) {
	var captured *http.Request
	var body map[string]string

	client := stubClient{do: func(req *http.Request) (*http.Response, error) {
		captured = req
		raw, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(raw, &body)
		return jsonResponse(http.StatusOK, `{
			"status": "success",
			"msg": "Tweet created successfully",
			"data": {"id": "2019384131067818211", "text": "hello"}
		}`), nil
	}}

	service := NewGetXAPIService(client, "API_KEY")

	id, err := service.CreateReply("AUTH_TOKEN", "123", "here is your video")

	require.NoError(t, err)
	assert.Equal(t, "2019384131067818211", id)
	assert.Equal(t, "Bearer API_KEY", captured.Header.Get("Authorization"))
	assert.Equal(t, "application/json", captured.Header.Get("Content-Type"))
	assert.Equal(t, "AUTH_TOKEN", body["auth_token"])
	assert.Equal(t, "123", body["reply_to_tweet_id"])
	assert.Equal(t, "here is your video", body["text"])
}

func TestGetXAPIService_GetTweetDetail_NotFound(t *testing.T) {
	client := stubClient{do: func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, `{"error": "Tweet not found: 123"}`), nil
	}}

	service := NewGetXAPIService(client, "API_KEY")

	tweet, err := service.GetTweetDetail("123")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTweetNotFound), "expected ErrTweetNotFound, got %v", err)
	assert.Nil(t, tweet)
}

func TestReason(t *testing.T) {
	assert.Equal(t, "tweet_not_found", Reason(ErrTweetNotFound))
	assert.Equal(t, "duplicate_reply", Reason(ErrDuplicateTweet))
	assert.Equal(t, "posting_throttled", Reason(ErrPostingThrottled))
	assert.Equal(t, "error", Reason(errors.New("boom")))
	assert.Empty(t, Reason(nil))
}

func TestGetXAPIService_CreateReply_Errors(t *testing.T) {
	testCases := []struct {
		name        string
		status      int
		body        string
		expectedErr error
	}{
		{
			name:        "duplicate tweet",
			status:      http.StatusConflict,
			body:        `{"error": "Status is a duplicate.", "twitter_error_code": 187}`,
			expectedErr: ErrDuplicateTweet,
		},
		{
			name:        "reply restricted",
			status:      http.StatusForbidden,
			body:        `{"error": "restricted", "twitter_error_code": 433}`,
			expectedErr: ErrReplyRestricted,
		},
		{
			name:        "posting throttled",
			status:      http.StatusTooManyRequests,
			body:        `{"error": "Posting temporarily limited", "twitter_error_code": 344}`,
			expectedErr: ErrPostingThrottled,
		},
		{
			name:        "server error",
			status:      http.StatusBadGateway,
			body:        `{"error": "bad gateway"}`,
			expectedErr: ErrUnavailable,
		},
		{
			name:        "tweet not found",
			status:      http.StatusNotFound,
			body:        `{"error": "Tweet not found: 123"}`,
			expectedErr: ErrTweetNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := stubClient{do: func(req *http.Request) (*http.Response, error) {
				return jsonResponse(tc.status, tc.body), nil
			}}

			service := NewGetXAPIService(client, "API_KEY")

			_, err := service.CreateReply("AUTH_TOKEN", "123", "text")

			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.expectedErr), "expected %v, got %v", tc.expectedErr, err)
		})
	}
}
