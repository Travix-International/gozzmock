package main

import (
	"net/http"
)

type httpClient interface {
	do(req *http.Request) (*http.Response, error)
}

type gzHTTPClient struct {
	httpClient *http.Client
}

func newGzHTTPClient() *gzHTTPClient {
	return &gzHTTPClient{
		httpClient: &http.Client{}}
}

func (gzClient *gzHTTPClient) do(req *http.Request) (*http.Response, error) {
	return gzClient.httpClient.Do(req)
}
