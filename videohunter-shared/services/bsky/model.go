package bsky

import "time"

type Root struct {
	Thread Thread `json:"thread"`
}

type Thread struct {
	Type    string  `json:"$type"`
	Post    Post    `json:"post"`
	Replies []Reply `json:"replies"`
}

type Post struct {
	URI         string    `json:"uri"`
	Cid         string    `json:"cid"`
	Author      Author    `json:"author"`
	Record      Record    `json:"record"`
	Embed       Embed     `json:"embed"`
	ReplyCount  int       `json:"replyCount"`
	RepostCount int       `json:"repostCount"`
	LikeCount   int       `json:"likeCount"`
	QuoteCount  int       `json:"quoteCount"`
	IndexedAt   time.Time `json:"indexedAt"`
	Labels      []any     `json:"labels"`
}

type Author struct {
	Did         string     `json:"did"`
	Handle      string     `json:"handle"`
	DisplayName string     `json:"displayName"`
	Avatar      string     `json:"avatar"`
	Associated  Associated `json:"associated"`
	Labels      []Label    `json:"labels"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type Associated struct {
	Chat Chat `json:"chat"`
}

type Chat struct {
	AllowIncoming string `json:"allowIncoming"`
}

type Label struct {
	Src string    `json:"src"`
	URI string    `json:"uri"`
	Cid string    `json:"cid"`
	Val string    `json:"val"`
	Cts time.Time `json:"cts"`
}

type Record struct {
	Type      string      `json:"$type"`
	CreatedAt time.Time   `json:"createdAt"`
	Embed     RecordEmbed `json:"embed"`
	Facets    []Facet     `json:"facets"`
	Langs     []string    `json:"langs"`
	Text      string      `json:"text"`
	Embeds    []Embed     `json:"embeds"`
}

type RecordEmbed struct {
	Type        string      `json:"$type"`
	AspectRatio AspectRatio `json:"aspectRatio"`
	Video       Video       `json:"video"`
}

type AspectRatio struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

type Video struct {
	Type     string `json:"$type"`
	Ref      Ref    `json:"ref"`
	MimeType string `json:"mimeType"`
	Size     int    `json:"size"`
}

type Ref struct {
	Link string `json:"$link"`
}

type Facet struct {
	Features []Feature `json:"features"`
	Index    Index     `json:"index"`
}

type Feature struct {
	Type string `json:"$type"`
	Tag  string `json:"tag"`
}

type Index struct {
	ByteEnd   int `json:"byteEnd"`
	ByteStart int `json:"byteStart"`
}

type Embed struct {
	Type        string      `json:"$type"`
	Cid         string      `json:"cid"`
	Playlist    string      `json:"playlist"`
	Thumbnail   string      `json:"thumbnail"`
	AspectRatio AspectRatio `json:"aspectRatio"`
	Media       Media       `json:"media"`
	Record      Record      `json:"record"`
}

type Media struct {
	Cid         string      `json:"cid"`
	Playlist    string      `json:"playlist"`
	Thumbnail   string      `json:"thumbnail"`
	AspectRatio AspectRatio `json:"aspectRatio"`
}

type Reply struct {
	Type    string `json:"$type"`
	Post    Post   `json:"post"`
	Replies []any  `json:"replies"`
}
