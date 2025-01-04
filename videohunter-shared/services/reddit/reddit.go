package reddit

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	shared_domain "github.com/victoraldir/myvideohuntershared/domain"
)

type InvalidPostError struct {
	StatusCode int
	Err        error
}

func (r *InvalidPostError) Error() string {
	return r.Err.Error()
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type redditDownloaderRepository struct {
	client HttpClient
}

func NewRedditDownloaderRepository(client HttpClient) *redditDownloaderRepository {
	return &redditDownloaderRepository{
		client: client,
	}
}

func (r *redditDownloaderRepository) DownloadVideo(url string, authToken ...string) (videoDownload *shared_domain.Video, currentToken *string, err error) {

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

	var posts []Post

	err = json.NewDecoder(resp.Body).Decode(&posts)

	if err != nil {
		return nil, nil, &InvalidPostError{StatusCode: 400, Err: err}
	}

	var t3 ChildData

	for _, post := range posts {
		for _, c := range post.Data.Children {
			if c.Kind == "t3" {
				t3 = c.Data
				break
			}
		}
	}

	var redditMedia RedditVideo

	redditMedia = t3.SecureMedia.RedditVideo

	if redditMedia.HlsURL == "" {
		redditMedia = t3.Preview.RedditVideoPreview
	}

	video := shared_domain.Video{
		IdDB:             t3.ID,
		OriginalVideoUrl: url,
		ThumbnailUrl:     t3.Thumbnail,
		CreatedAt:        time.Now().String(),
		Text:             t3.Title,
		ExtendedEntities: shared_domain.ExtendedEntities{
			Media: []shared_domain.Media{
				{
					MediaUrl: redditMedia.HlsURL,
					Type:     "video",
					VideoInfo: shared_domain.VideoInfo{
						Variants: []shared_domain.Variants{
							{
								Bitrate:     redditMedia.BitrateKbps,
								URL:         redditMedia.HlsURL,
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

	// if redditClientId == "" || redditClientSecret == "" {
	// 	return "", fmt.Errorf("REDDIT_CLIENT_ID or REDDIT_CLIENT_SECRET not found")
	// }

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

		// Set Authorization header
		basicAuth, _ := r.GetAuthToken()

		req, err := http.NewRequest("GET", url, nil)

		if err != nil {
			return "", err
		}

		req.Header.Set("Authorization", basicAuth)

		resp, err := client.Do(req)

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
