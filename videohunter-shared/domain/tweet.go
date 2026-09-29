package domain

import "encoding/json"

// TweetAuthor is the author object returned by the GetXAPI tweet endpoints.
type TweetAuthor struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	UserName string `json:"userName"`
	Name     string `json:"name"`
}

// Tweet is a tweet as returned by the GetXAPI advanced search and detail endpoints.
type Tweet struct {
	Type            string          `json:"type"`
	ID              string          `json:"id"`
	URL             string          `json:"url"`
	TwitterURL      string          `json:"twitterUrl"`
	Text            string          `json:"text"`
	CreatedAt       string          `json:"createdAt"`
	IsReply         bool            `json:"isReply"`
	InReplyToID     string          `json:"inReplyToId"`
	InReplyToUserID string          `json:"inReplyToUserId"`
	ConversationID  string          `json:"conversationId"`
	Media           json.RawMessage `json:"media"`
	Author          TweetAuthor     `json:"author"`
}

// HasMedia reports whether the tweet carries any media payload. The exact media
// shape is not needed, only whether the tweet owns a video/image.
func (t Tweet) HasMedia() bool {
	if len(t.Media) == 0 {
		return false
	}

	var media []json.RawMessage
	if err := json.Unmarshal(t.Media, &media); err != nil {
		return false
	}

	return len(media) > 0
}

// TweetSearchResponse is the response of the GetXAPI advanced search endpoint.
type TweetSearchResponse struct {
	Query      string  `json:"query"`
	TweetCount int     `json:"tweet_count"`
	HasMore    bool    `json:"has_more"`
	NextCursor string  `json:"next_cursor"`
	Tweets     []Tweet `json:"tweets"`
}
