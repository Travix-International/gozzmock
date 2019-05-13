package httpclient

import (
	"net/http"
)

type RoundTripper struct {
	http.RoundTripper
}

func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt.RoundTrip(req)
}

func NewRoundTripper() http.RoundTripper {
	return http.DefaultTransport
}
