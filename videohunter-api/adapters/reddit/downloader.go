package reddit

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/victoraldir/myvideohunterapi/adapters/httpclient"
	"github.com/victoraldir/myvideohunterapi/domain"
)

type Comment struct {
	Kind string `json:"kind"`
	Data Data   `json:"data"`
}

type Data struct {
	Children []Thread `json:"children"`
}

type Thread struct {
	Kind  string `json:"kind"`
	Media Media  `json:"data"`
}

type Media struct {
	SecureMedia SecureMedia `json:"secure_media"`
	Id          string      `json:"id"`
	Thumbnail   string      `json:"thumbnail"`
	Title       string      `json:"title"`
}

type SecureMedia struct {
	RedditVideo RedditVideo `json:"reddit_video"`
}

type RedditVideo struct {
	HlsUrl           string `json:"hls_url"`
	BitrateKbps      int    `json:"bitrate_kbps"`
	ScrubberMediaUrl string `json:"scrubber_media_url"`
}

type redditDownloaderRepository struct {
	client httpclient.HttpClient
}

func NewRedditDownloaderRepository(client httpclient.HttpClient) *redditDownloaderRepository {
	return &redditDownloaderRepository{
		client: client,
	}
}

func (r *redditDownloaderRepository) DownloadVideo(url string, authToken ...string) (videoDownload *domain.Video, currentToken *string, err error) {

	urlWithExtension := url + ".json"

	req, err := http.NewRequest("GET", urlWithExtension, nil)

	if err != nil {
		log.Println("Error creating request", "error", err)
		return nil, nil, err
	}

	resp, err := r.client.Do(req)

	if err != nil {
		log.Println("Error making request", "error", err)
		return nil, nil, err
	}

	defer resp.Body.Close()

	log.Println("Response status", "status", resp.Status)
	log.Println("Response headers", "headers", resp.Header)

	var comments []Comment

	err = json.NewDecoder(resp.Body).Decode(&comments)

	if err != nil {
		return nil, nil, err
	}

	comment := comments[0]

	var t3 Thread

	for _, c := range comment.Data.Children {
		if c.Kind == "t3" {
			t3 = c
			break
		}
	}

	fmt.Println(t3)

	video := domain.Video{
		IdDB:             t3.Media.Id,
		OriginalVideoUrl: url,
		ThumbnailUrl:     t3.Media.SecureMedia.RedditVideo.ScrubberMediaUrl,
		CreatedAt:        time.Now().String(),
		Text:             t3.Media.Title,
		ExtendedEntities: domain.ExtendedEntities{
			Media: []domain.Media{
				{
					MediaUrl: t3.Media.SecureMedia.RedditVideo.HlsUrl,
					Type:     "video",
					VideoInfo: domain.VideoInfo{
						Variants: []domain.Variants{
							{
								Bitrate:     t3.Media.SecureMedia.RedditVideo.BitrateKbps,
								URL:         t3.Media.SecureMedia.RedditVideo.HlsUrl,
								ContentType: "video/mp4",
							},
						},
					},
				},
			},
		},
	}

	return &video, nil, nil
}
