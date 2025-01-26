package domain

type Url struct {
	Id          string `json:"id"`
	Thumbnail   string `json:"thumbnail"`
	Description string `json:"description"`
	Uri         string `json:"uri"`
}

type Posts struct {
	Posts []Post `json:"posts"`
}

type Post struct {
	Uri    string `json:"uri"`
	Cid    string `json:"cid"`
	Author Author `json:"author"`
	Record Record `json:"record"`
	Embed  Embed  `json:"embed"`
	Url    *Url   // Enriched video
}

type PostReply struct {
	Record     RecordReply `json:"record"`
	Repo       string      `json:"repo"`
	Collection string      `json:"collection"`
}

type RecordReply struct {
	Text          string         `json:"text"`
	CreatedAt     string         `json:"createdAt"`
	Reply         Reply          `json:"reply"`
	EmbedExternal *EmbedExternal `json:"embed"`
	Facets        []Facet        `json:"facets"`
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

type Facet struct {
	Features []Feature `json:"features"`
	Index    Index     `json:"index"`
}

type Index struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

type Feature struct {
	Type string `json:"$type"`
	Uri  string `json:"uri"`
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
	Type        string     `json:"$type"`
	Uri         string     `json:"uri"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Cid         string     `json:"cid"`
	AspecRatio  AspecRatio `json:"aspectRatio"`
	Video       Video      `json:"video"`
	Playlist    string     `json:"playlist"`
	Thumbnail   string     `json:"thumbnail"`
}

type EmbedExternal struct {
	Type     string   `json:"$type"`
	External External `json:"external"`
}

type External struct {
	Uri         string `json:"uri"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AspecRatio struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

type Video struct {
	Type             string           `json:"$type"`
	MimeType         string           `json:"mimeType"`
	Size             int              `json:"size"`
	IdDB             string           `json:"id_db"`
	OriginalVideoUrl string           `json:"original_video_url"`
	ThumbnailUrl     string           `json:"media_url_https"`
	CreatedAt        string           `json:"created_at"`
	ExtendedEntities ExtendedEntities `json:"extended_entities"`
	Text             string           `json:"full_text"`
	QuotedStatus     Status           `json:"quoted_status"`
	RetweetedStatus  Status           `json:"retweeted_status"`
	Path             string           `json:"path"`
}

type Media struct {
	VideoInfo VideoInfo `json:"video_info"`
	MediaUrl  string    `json:"media_url_https"`
	Type      string    `json:"type"`
}

type Status struct {
	ExtendedEntities ExtendedEntities `json:"extended_entities"`
	Text             string           `json:"full_text"`
}

type ExtendedEntities struct {
	Media []Media `json:"media"`
}

type VideoInfo struct {
	Variants []Variants `json:"variants"`
}

type Variants struct {
	Bitrate     int    `json:"bitrate"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
}

func (v Video) GetText() string {

	if v.ExtendedEntities.Media != nil && v.ExtendedEntities.Media[0].Type == "video" {
		return v.Text
	}

	if v.QuotedStatus.ExtendedEntities.Media != nil && v.QuotedStatus.ExtendedEntities.Media[0].Type == "video" {
		return v.QuotedStatus.Text
	}

	if v.RetweetedStatus.ExtendedEntities.Media != nil && v.RetweetedStatus.ExtendedEntities.Media[0].Type == "video" {
		return v.RetweetedStatus.Text
	}

	return ""
}

func (v Video) GetMedia() *Media {

	if v.ExtendedEntities.Media != nil && v.ExtendedEntities.Media[0].Type == "video" {
		return &v.ExtendedEntities.Media[0]
	}

	if v.QuotedStatus.ExtendedEntities.Media != nil && v.QuotedStatus.ExtendedEntities.Media[0].Type == "video" {
		return &v.QuotedStatus.ExtendedEntities.Media[0]
	}

	if v.RetweetedStatus.ExtendedEntities.Media != nil && v.RetweetedStatus.ExtendedEntities.Media[0].Type == "video" {
		return &v.RetweetedStatus.ExtendedEntities.Media[0]
	}

	return nil
}
