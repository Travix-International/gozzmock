package main

import (
	"net/http"
)

type httpClient interface {
	roundTrip(req *http.Request) (*http.Response, error)
}

type gzHTTPClient struct {
	roundTripper http.RoundTripper
}

func (gzClient *gzHTTPClient) roundTrip(req *http.Request) (*http.Response, error) {
	return gzClient.roundTripper.RoundTrip(req)
}

func newGzHTTPClient() *gzHTTPClient {
	return &gzHTTPClient{
		roundTripper: http.DefaultTransport}
}
