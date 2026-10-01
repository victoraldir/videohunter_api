package domain

// Folder groups the videos a signed in user has saved. A library is stored as
// one row per folder plus one row per saved video, all under the same user id,
// so listing it is a single query.
type Folder struct {
	Id        string       `json:"id"`
	Name      string       `json:"name"`
	CreatedAt string       `json:"created_at"`
	Videos    []SavedVideo `json:"videos"`
}

// SavedVideo is a reference to a video that already exists in the video table.
// Saving never copies the video, only the id.
//
// The thumbnail and the description are filled in when a library is listed, so
// the page can draw a card per video without a request per video. They are
// never stored.
type SavedVideo struct {
	VideoId      string `json:"video_id"`
	SavedAt      string `json:"saved_at"`
	ThumbnailUrl string `json:"thumbnail_url,omitempty"`
	Description  string `json:"description,omitempty"`
}
