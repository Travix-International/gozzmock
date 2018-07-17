package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/stretchr/testify/assert"
)

func TestGzServerTranslateRequestToExpectation_SimpleRequest_AllFieldsTranslated(t *testing.T) {
	request, err := http.NewRequest("POST", "https://www.host.com/a/b?foo=bar#fr", strings.NewReader("body text"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("h1", "hv1")

	exp := translateRequestToExpectation(request)

	assert.NotNil(t, exp)
	assert.Equal(t, "POST", exp.Method)
	assert.Equal(t, "/a/b?foo=bar#fr", exp.Path)
	assert.Equal(t, "body text", exp.Body)
	assert.NotNil(t, exp.Headers)
	assert.Equal(t, 1, len(*exp.Headers))
	assert.Equal(t, "hv1", (*exp.Headers)["H1"])
}

func TestGzServerTranslateHTTPHeadersToExpHeaders_TwoHeaders_HeadersTranslated(t *testing.T) {
	header := http.Header{}
	header.Add("h1", "hv1")
	header.Add("h1", "hv2")

	expHeaders := translateHTTPHeadersToExpHeaders(header)
	assert.NotNil(t, expHeaders)
	assert.Equal(t, 1, len(*expHeaders))
	assert.Equal(t, "hv1,hv2", (*expHeaders)["H1"])
}

func TestGzServerControllerStringPassesFilter_EmptyFilter_True(t *testing.T) {
	assert.True(t, controllerStringPassesFilter("abc", ""))
}

func TestGzServerControllerStringPassesFilter_ExistingSubstring_True(t *testing.T) {
	assert.True(t, controllerStringPassesFilter("abc", "ab"))
}

func TestGzServerControllerStringPassesFilter_ExistingRegex_True(t *testing.T) {
	assert.True(t, controllerStringPassesFilter("abc", ".b."))
}

func TestGzServerControllerStringPassesFilter_NotExistingSubstring_False(t *testing.T) {
	assert.False(t, controllerStringPassesFilter("abc", "zz"))
}

func TestGzServerControllerStringPassesFilter_NotExistingRegex_False(t *testing.T) {
	assert.False(t, controllerStringPassesFilter("abc", ".z."))
}

func TestGzServerControllerStringPassesFilter_MultilineBody_True(t *testing.T) {
	assert.True(t, controllerStringPassesFilter("a\nb", "a.b"))
}

func TestGzServerControllerRequestPassFilter_EmptyRequestEmptyFilter_True(t *testing.T) {
	assert.True(t, controllerRequestPassesFilter(
		&ExpectationRequest{},
		&ExpectationRequest{}))
}

func TestGzServerControllerRequestPassFilter_MethodsAreEq_True(t *testing.T) {
	assert.True(t, controllerRequestPassesFilter(
		&ExpectationRequest{Method: "POST"},
		&ExpectationRequest{Method: "POST"}))
}

func TestGzServerControllerRequestPassFilter_PathsAreEq_True(t *testing.T) {
	assert.True(t, controllerRequestPassesFilter(
		&ExpectationRequest{Path: "/path"},
		&ExpectationRequest{Path: "/path"}))
}

func TestGzServerControllerRequestPassFilter_MethodsNotEqAndPathsAreEq_False(t *testing.T) {
	assert.False(t, controllerRequestPassesFilter(
		&ExpectationRequest{Method: "GET", Path: "/path"},
		&ExpectationRequest{Method: "POST", Path: "/path"}))
}

func TestGzServerControllerRequestPassFilter_HeadersAreEq_True(t *testing.T) {
	assert.True(t, controllerRequestPassesFilter(
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}},
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}}))
}

func TestGzServerControllerRequestPassFilter_HeaderNotEq_False(t *testing.T) {
	result := controllerRequestPassesFilter(
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}},
		&ExpectationRequest{Headers: &Headers{"h2": "hv2"}})
	assert.False(t, result)
}

func TestGzServerControllerRequestPassFilter_HeaderValueNotEq_False(t *testing.T) {
	assert.False(t, controllerRequestPassesFilter(
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}},
		&ExpectationRequest{Headers: &Headers{"h1": "hv2"}}))
}

func TestGzServerControllerRequestPassFilter_NoHeaderinReq_False(t *testing.T) {
	assert.False(t, controllerRequestPassesFilter(
		&ExpectationRequest{},
		&ExpectationRequest{Headers: &Headers{"h2": "hv2"}}))
}

func TestGzServerControllerRequestPassFilter_NoHeaderInFilter_True(t *testing.T) {
	assert.True(t, controllerRequestPassesFilter(
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}},
		&ExpectationRequest{}))
}

func TestGzServerControllerRequestPassFilter_BodysEq_True(t *testing.T) {
	assert.True(t, controllerRequestPassesFilter(
		&ExpectationRequest{Body: "body"},
		&ExpectationRequest{Body: "body"}))
}

func TestGzServerControllerCreateHTTPRequestWithHeaders(t *testing.T) {
	expReq := &ExpectationRequest{Method: "GET", Path: "/request", Headers: &Headers{"h_req": "hv_req"}}
	expFwd := &ExpectationForward{Scheme: "https", Host: "localhost_fwd", Headers: &Headers{"h_req": "hv_fwd", "h_fwd": "hv_fwd"}}
	httpReq := controllerCreateHTTPRequest(expReq, expFwd)
	assert.NotNil(t, httpReq)
	assert.Equal(t, expReq.Method, httpReq.Method)
	assert.Equal(t, expFwd.Host, httpReq.Host)
	assert.Equal(t, fmt.Sprintf("%s://%s%s", expFwd.Scheme, expFwd.Host, expReq.Path), httpReq.URL.String())
	assert.Equal(t, "hv_fwd", httpReq.Header.Get("h_req"))
	assert.Equal(t, "hv_fwd", httpReq.Header.Get("h_fwd"))
}

func TestGzServerControllerCreateHTTPRequestHostRewrite(t *testing.T) {
	expReq := &ExpectationRequest{Method: "GET", Path: "/request"}
	expFwd := &ExpectationForward{Scheme: "https", Host: "localhost_fwd", Headers: &Headers{"Host": "fwd_host"}}
	httpReq := controllerCreateHTTPRequest(expReq, expFwd)
	assert.NotNil(t, httpReq)
	assert.Equal(t, "fwd_host", httpReq.Host)
	assert.Equal(t, fmt.Sprintf("%s://%s%s", expFwd.Scheme, expFwd.Host, expReq.Path), httpReq.URL.String())
}

// Integration tests

func httpNewRequestMust(method, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	return req
}

type gzMockedHTTPClient struct {
}

func (gzClient *gzMockedHTTPClient) do(req *http.Request) (*http.Response, error) {
	bodyString := req.URL.String() + " Host:" + req.Host
	for k, v := range req.Header {
		bodyString += " " + k + ":" + strings.Join(v, ",")
	}
	body := ioutil.NopCloser(strings.NewReader(bodyString))
	response := &http.Response{StatusCode: 200, Body: body}
	return response, nil
}

func newMockedGzServer() *gzServer {
	server := &gzServer{logLevel: zerolog.DebugLevel}
	server.storage = newGzStorage()
	server.httpClient = &gzMockedHTTPClient{}
	return server
}

func TestHandlerGet_NoExpectations_ReturnEmptyList(t *testing.T) {
	//Arrange
	server := newMockedGzServer()

	r := httpNewRequestMust("GET", "/gozzmock/get_expectations", nil)
	w := httptest.NewRecorder()

	//Act
	server.get(w, r)

	//Assert
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
	//Arrange
	server := newMockedGzServer()

	r := httpNewRequestMust("GET", "/request", nil)
	w := httptest.NewRecorder()

	//Act
	server.root(w, r)

	//Assert
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Equal(t, "No expectations in gozzmock for request!", w.Body.String())
}

func TestHandlerAddRemove_AddAndRemoveExpectation(t *testing.T) {
	//Arrange
	server := newMockedGzServer()
	exp := Expectation{Key: "k"}

	expJSON := jsonMarshalMust(exp)
	expRemoveJSON := jsonMarshalMust(ExpectationRemove{Key: exp.Key})

	rAdd := httpNewRequestMust("POST", "/add", bytes.NewBuffer(expJSON))
	rRemove := httpNewRequestMust("POST", "/remove", bytes.NewBuffer(expRemoveJSON))

	wAdd := httptest.NewRecorder()
	wRemove := httptest.NewRecorder()

	//Act
	server.add(wAdd, rAdd)
	server.remove(wRemove, rRemove)

	//Assert
	assert.Equal(t, http.StatusOK, wAdd.Code)
	assert.Equal(t, "Expectation with key 'k' was added", wAdd.Body.String())
	assert.Equal(t, http.StatusOK, wRemove.Code)
	assert.Equal(t, "Expectation with key 'k' was removed", wRemove.Body.String())
}

func TestHandlerRoot_TwoOverlapingExpectations(t *testing.T) {
	//Arrange
	server := newMockedGzServer()
	exp1 := Expectation{
		Key:      "response",
		Request:  &ExpectationRequest{Path: "/response"},
		Response: &ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1}

	exp2 := Expectation{
		Key:      "forward",
		Forward:  &ExpectationForward{Scheme: "https", Host: "local.com"},
		Priority: 0}

	server.storage.add(exp1.Key, exp1)
	server.storage.add(exp2.Key, exp2)

	reqToRespnse := httpNewRequestMust("POST", "/response", bytes.NewBuffer([]byte("request body")))
	reqToForward := httpNewRequestMust("POST", "/forward", bytes.NewBuffer([]byte("forward body")))

	wR := httptest.NewRecorder()
	wF := httptest.NewRecorder()

	//Act
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
	exp1 := Expectation{
		Key:      "response",
		Request:  &ExpectationRequest{Path: "/response"},
		Response: &ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1}

	server.storage.add(exp1.Key, exp1)

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
	// Arrange
	server := newMockedGzServer()
	exp1 := Expectation{
		Key:      "response",
		Request:  &ExpectationRequest{Path: "/response"},
		Response: &ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1}

	server.storage.add(exp1.Key, exp1)

	r := httpNewRequestMust("GET", "/gozzmock/status", nil)
	w := httptest.NewRecorder()

	// Act
	server.status(w, r)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, "gozzmock status is OK", w.Body.String())
}

func TestHandlerRoot_ForwardValidatrHeaders(t *testing.T) {
	// Arrange
	server := newMockedGzServer()
	exp1 := Expectation{
		Key:      "forward",
		Forward:  &ExpectationForward{Scheme: "https", Host: "local.xx", Headers: &Headers{"Host": "fwd_host"}},
		Priority: 0}

	server.storage.add(exp1.Key, exp1)

	r := httpNewRequestMust("POST", "/forward", bytes.NewBuffer([]byte("forward body")))
	r.Host = "reqest_host"
	w := httptest.NewRecorder()

	// Act
	server.root(w, r)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `https://local.xx/forward Host:fwd_host`, w.Body.String())
}

type gzMockedGzipHTTPClient struct {
}

func (gzClient *gzMockedGzipHTTPClient) do(req *http.Request) (*http.Response, error) {
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

func TestHandlerRoot_ForwardReturnsGzip(t *testing.T) {
	// Arrange
	server := newMockedGzServer()
	server.httpClient = &gzMockedGzipHTTPClient{}

	exp1 := Expectation{
		Key:      "forward",
		Forward:  &ExpectationForward{Scheme: "https", Host: "local.xx"},
		Priority: 0}

	server.storage.add(exp1.Key, exp1)

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
	// Arrange
	server := newMockedGzServer()
	jsTemplate := `"123".length`

	exp := Expectation{
		Key: "template",
		Response: &ExpectationResponse{HTTPCode: http.StatusOK,
			JsTemplate: base64.StdEncoding.EncodeToString([]byte(jsTemplate))},
		Priority: 1}
	server.storage.add(exp.Key, exp)

	r := httpNewRequestMust("GET", "/", nil)
	w := httptest.NewRecorder()

	// Act
	server.root(w, r)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "3")
}
