package domain

type VideoUrl struct {
	Id           string `json:"id"`
	ThumbnailUrl string `json:"thumbnail_url"`
	Description  string `json:"description"`
}

func (v VideoUrl) GetUrl() string {
	return "https://www.myvideohunter.com/prod/url/" + v.Id
}
