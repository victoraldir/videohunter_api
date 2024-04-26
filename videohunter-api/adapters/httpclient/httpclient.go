package httpclient

import "net/http"

//go:generate mockgen -destination=../httpclient/mocks/mockHttpClient.go -package=httpclient github.com/victoraldir/myvideohunterapi/adapters/httpclient HttpClient
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
