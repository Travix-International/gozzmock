package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Travix-International/gozzmock/expectations"
	"github.com/rs/zerolog"

	"github.com/stretchr/testify/assert"
)

func httpNewRequestMust(method, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	return req
}

type mockedRoundTripper struct{}

func (rt *mockedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	response := &http.Response{StatusCode: 200, Header: http.Header{}}

	var sb strings.Builder
	sb.WriteString(req.URL.String())
	sb.WriteString(" Host:")
	sb.WriteString(req.Host)
	for k, v := range req.Header {
		sb.WriteString(" ")
		sb.WriteString(k)
		sb.WriteString(":")
		sv := strings.Join(v, ",")
		sb.WriteString(sv)
		response.Header.Add(k, sv)
	}
	response.Body = ioutil.NopCloser(strings.NewReader(sb.String()))
	return response, nil
}

func newMockedGzServer() *gzServer {
	server := &gzServer{logLevel: zerolog.DebugLevel}
	server.filter = expectations.NewGzFilter(&mockedRoundTripper{}, expectations.NewGzStorage())
	return server
}

func TestHandlerGet_NoExpectations_ReturnEmptyList(t *testing.T) {
	server := newMockedGzServer()

	r := httpNewRequestMust("GET", "/gozzmock/get_expectations", nil)
	w := httptest.NewRecorder()

	// Act
	server.get(w, r)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "{}", w.Body.String())
}

func jsonMarshalMust(v interface{}) []byte {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return encoded
}

func TestHandlerRoot_NoExpectations(t *testing.T) {
	server := newMockedGzServer()

	r := httpNewRequestMust("GET", "/request", nil)
	w := httptest.NewRecorder()

	// Act
	server.root(w, r)

	// Assert
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Equal(t, "No expectations in gozzmock for request!", w.Body.String())
}

func TestHandlerAddRemove_AddAndRemoveExpectation(t *testing.T) {
	server := newMockedGzServer()
	exp := expectations.Expectation{Key: "k"}

	expJSON := jsonMarshalMust(exp)
	expRemoveJSON := jsonMarshalMust(expectations.ExpectationRemove{Key: exp.Key})

	rAdd := httpNewRequestMust("POST", "/add", bytes.NewBuffer(expJSON))
	rRemove := httpNewRequestMust("POST", "/remove", bytes.NewBuffer(expRemoveJSON))

	wAdd := httptest.NewRecorder()
	wRemove := httptest.NewRecorder()

	// Act
	server.add(wAdd, rAdd)
	server.remove(wRemove, rRemove)

	// Assert
	assert.Equal(t, http.StatusOK, wAdd.Code)
	assert.Equal(t, "Expectation with key 'k' was added", wAdd.Body.String())
	assert.Equal(t, http.StatusOK, wRemove.Code)
	assert.Equal(t, "Expectation with key 'k' was removed", wRemove.Body.String())
}

func TestHandlerRoot_TwoOverlapingExpectations(t *testing.T) {
	server := newMockedGzServer()
	exp1 := expectations.Expectation{
		Key:      "response",
		Request:  &expectations.ExpectationRequest{Path: "/response"},
		Response: &expectations.ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1}

	exp2 := expectations.Expectation{
		Key:      "forward",
		Forward:  &expectations.ExpectationForward{Scheme: "https", Host: "local.com"},
		Priority: 0}

	server.filter.Add(exp1)
	server.filter.Add(exp2)

	reqToRespnse := httpNewRequestMust("POST", "/response", bytes.NewBuffer([]byte("request body")))
	reqToForward := httpNewRequestMust("POST", "/forward", bytes.NewBuffer([]byte("forward body")))

	wR := httptest.NewRecorder()
	wF := httptest.NewRecorder()

	// Act
	server.root(wR, reqToRespnse)
	server.root(wF, reqToForward)

	// Assert
	assert.Equal(t, http.StatusOK, wR.Code)
	assert.Equal(t, "response body", wR.Body.String())

	assert.Equal(t, http.StatusOK, wF.Code)
	assert.Equal(t, "https://local.com/forward Host:local.com", wF.Body.String())
}

func TestHandlerGet_GetExpectations(t *testing.T) {
	server := newMockedGzServer()
	exp1 := expectations.Expectation{
		Key:      "response",
		Request:  &expectations.ExpectationRequest{Path: "/response"},
		Response: &expectations.ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1}

	server.filter.Add(exp1)

	expectedResponse := jsonMarshalMust(exp1)

	rGet := httpNewRequestMust("GET", "/gozzmock/get_expectations", nil)
	wGet := httptest.NewRecorder()

	// Act
	server.get(wGet, rGet)

	// Assert
	assert.Equal(t, http.StatusOK, wGet.Code)
	assert.Contains(t, wGet.Body.String(), string(expectedResponse))
}

func TestHandler_Status(t *testing.T) {
	server := newMockedGzServer()
	exp1 := expectations.Expectation{
		Key:      "response",
		Request:  &expectations.ExpectationRequest{Path: "/response"},
		Response: &expectations.ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1}

	server.filter.Add(exp1)

	r := httpNewRequestMust("GET", "/gozzmock/status", nil)
	w := httptest.NewRecorder()

	// Act
	server.status(w, r)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, "gozzmock status is OK", w.Body.String())
}

func TestHandlerRoot_ForwardValidatrHeaders(t *testing.T) {
	server := newMockedGzServer()
	exp1 := expectations.Expectation{
		Key: "forward",
		Forward: &expectations.ExpectationForward{
			Scheme:  "https",
			Host:    "local.xx",
			Headers: expectations.Headers{"Host": "fwd_host"}},
		Priority: 0}

	server.filter.Add(exp1)

	r := httpNewRequestMust("POST", "/forward", bytes.NewBuffer([]byte("forward body")))
	r.Host = "reqest_host"
	w := httptest.NewRecorder()

	// Act
	server.root(w, r)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `https://local.xx/forward Host:fwd_host`, w.Body.String())
}

type mockedGzipRoundTripper struct{}

func (rt *mockedGzipRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := http.Response{
		Header: make(http.Header)}

	resp.StatusCode = http.StatusOK
	resp.Header.Add("Content-Encoding", "gzip")

	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Write([]byte(req.URL.String()))
	gz.Close()

	resp.Body = ioutil.NopCloser(bufio.NewReader(&b))

	return &resp, nil
}

func newMockedGzipServer() *gzServer {
	server := &gzServer{logLevel: zerolog.DebugLevel}
	server.filter = expectations.NewGzFilter(&mockedGzipRoundTripper{}, expectations.NewGzStorage())
	return server
}

func TestHandlerRoot_ForwardReturnsGzip(t *testing.T) {
	server := newMockedGzipServer()

	exp1 := expectations.Expectation{
		Key:      "forward",
		Forward:  &expectations.ExpectationForward{Scheme: "https", Host: "local.xx"},
		Priority: 0}

	server.filter.Add(exp1)

	r := httpNewRequestMust("POST", "/forward", bytes.NewBuffer([]byte("forward body")))
	r.Header.Add("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()

	// Act
	server.root(w, r)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, `https://local.xx/forward`, string(body))
}

func TestHandlerRoot_RespondsWithJsTemplate(t *testing.T) {
	server := newMockedGzServer()

	jsTemplate := base64.StdEncoding.EncodeToString([]byte(`"123".length`))

	exp := expectations.Expectation{
		Key: "template",
		Response: &expectations.ExpectationResponse{HTTPCode: http.StatusOK,
			JsTemplate: jsTemplate},
		Priority: 1}
	server.filter.Add(exp)

	r := httpNewRequestMust("GET", "/", nil)
	w := httptest.NewRecorder()

	// Act
	server.root(w, r)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "3", w.Body.String())
}
