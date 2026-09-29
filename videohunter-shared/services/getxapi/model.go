package getxapi

import (
	"fmt"

	"github.com/victoraldir/myvideohuntershared/domain"
)

// createTweetResponse is the success response of POST /twitter/tweet/create.
type createTweetResponse struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
	Data   struct {
		ID string `json:"id"`
	} `json:"data"`
}

// tweetDetailResponse is the response of GET /twitter/tweet/detail.
type tweetDetailResponse struct {
	Status string       `json:"status"`
	Msg    string       `json:"msg"`
	Data   domain.Tweet `json:"data"`
}

// apiError is the error body returned by GetXAPI.
type apiError struct {
	Message          string `json:"error"`
	TwitterErrorCode int    `json:"twitter_error_code"`
}

func (e apiError) Error() string {
	if e.TwitterErrorCode != 0 {
		return fmt.Sprintf("%s (twitter_error_code=%d)", e.Message, e.TwitterErrorCode)
	}

	if e.Message == "" {
		return "unknown getxapi error"
	}

	return e.Message
}
