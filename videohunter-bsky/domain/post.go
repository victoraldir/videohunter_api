package domain

type Posts struct {
	Posts []Post `json:"posts"`
}

type Post struct {
	Uri       string    `json:"uri"`
	Cid       string    `json:"cid"`
	Author    Author    `json:"author"`
	Record    Record    `json:"record"`
	Embed     Embed     `json:"embed"`
	RootVideo RootVideo // Enriched video
}

type RootVideo struct {
	Cid        string     `json:"cid"`
	Thumbnail  string     `json:"thumbnail"`
	Playlist   string     `json:"playlist"`
	AspecRatio AspecRatio `json:"aspectRatio"`
}

type Author struct {
	DisplayName string `json:"displayName"`
	Did         string `json:"did"`
}

type Record struct {
	CreatedAt string   `json:"createdAt"`
	Langs     []string `json:"langs"`
	Reply     Reply    `json:"reply"`
	Text      string   `json:"text"`
	Embed     Embed    `json:"embed"`
}

type Reply struct {
	Parent PostItem `json:"parent"`
	Root   PostItem `json:"root"`
}

type PostItem struct {
	Cid string `json:"cid"`
	Uri string `json:"uri"`
}

type Embed struct {
	Type       string     `json:"$type"`
	Cid        string     `json:"cid"`
	AspecRatio AspecRatio `json:"aspectRatio"`
	Video      Video      `json:"video"`
	Playlist   string     `json:"playlist"`
	Thumbnail  string     `json:"thumbnail"`
}

type AspecRatio struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

type Video struct {
	Type     string `json:"$type"`
	MimeType string `json:"mimeType"`
	Size     int    `json:"size"`
}
