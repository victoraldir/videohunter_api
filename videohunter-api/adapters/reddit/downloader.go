package reddit

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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

	url, err = r.GetJsonUrl(url)

	if err != nil {
		log.Println("Error getting json url", "error", err)
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url, nil)

	basicAuth, err := r.GetAuthToken()

	if err != nil {
		log.Println("Error getting auth token", "error", err)
		return nil, nil, err
	}

	req.Header.Set("Authorization", basicAuth)

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

func (r *redditDownloaderRepository) GetAuthToken() (authToken string, err error) {

	redditClientId := os.Getenv("REDDIT_CLIENT_ID")
	redditClientSecret := os.Getenv("REDDIT_CLIENT_SECRET")

	if redditClientId == "" || redditClientSecret == "" {
		return "", fmt.Errorf("REDDIT_CLIENT_ID or REDDIT_CLIENT_SECRET not found")
	}

	return "Basic " + redditClientId + ":" + redditClientSecret, nil
}

func (r *redditDownloaderRepository) GetJsonUrl(url string) (string, error) {

	splitUrl := strings.Split(url, "/")

	// Check if url is short url
	if splitUrl[5] == "s" {
		// Get the Location header. We need to configure the client to not follow redirects
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, err := client.Get(url)

		if err != nil {
			return "", err
		}

		defer resp.Body.Close()

		url = resp.Header.Get("Location")

		if url == "" {
			return "", fmt.Errorf("error getting location header")
		}
	}

	splitUrlQuery := strings.Split(url, "?")

	url = splitUrlQuery[0]

	if url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	urlWithExtension := url + ".json"

	return urlWithExtension, nil
}
