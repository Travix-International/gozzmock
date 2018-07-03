package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestGzServer_NoExpectations_ReturnEmptyList(t *testing.T) {
	//Arrange
	server := gzServer{logLevel: zerolog.DebugLevel}
	server.storage.init()

	r := httpNewRequestMust("GET", "/gozzmock/get_expectations", nil)
	w := httptest.NewRecorder()

	//Act
	server.get(w, r)

	//Assert
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Equal(t, "{}", w.Body.String())
}

func jsonMarshalMust(v interface{}) []byte {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return encoded
}

/*
func (context *gzServerMock) getExpectations(t *testing.T) *bytes.Buffer {
	handlerGetExpectations := http.HandlerFunc(context.HandlerGetExpectations)

	handlerGetExpectations.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)

	return httpTestResponseRecorder.Body
}

func (context *gzServerMock) addExpectation(t *testing.T, exp Expectation) *bytes.Buffer {
	handlerAddExpectation := http.HandlerFunc(context.HandlerAddExpectation)

	expJSON := jsonMarshalMust(exp)

	req := httpNewRequestMust("POST", "/gozzmock/add_expectation", bytes.NewBuffer(expJSON))

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerAddExpectation.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)
	assert.Equal(t, fmt.Sprintf("Expectation with key '%s' was added", exp.Key), httpTestResponseRecorder.Body.String())

	return context.getExpectations(t)
}

func (context *gzServerMock) removeExpectation(t *testing.T, expKey string) *bytes.Buffer {
	handlerRemoveExpectation := http.HandlerFunc(context.HandlerRemoveExpectation)

	expRemoveJSON := jsonMarshalMust(ExpectationRemove{Key: expKey})

	req := httpNewRequestMust("POST", "/gozzmock/remove_expectation", bytes.NewBuffer(expRemoveJSON))

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerRemoveExpectation.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)
	assert.Equal(t, fmt.Sprintf("Expectation with key '%s' was removed", expKey), httpTestResponseRecorder.Body.String())

	return context.getExpectations(t)
}
*/

func TestHandlerNoExpectations(t *testing.T) {
	// Arrange
	server := gzServer{logLevel: zerolog.DebugLevel}
	server.storage.init()
	handlerDefault := http.HandlerFunc(server.root)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response from test server"))
	}))
	defer testServer.Close()

	// do request for response
	req := httpNewRequestMust("GET", "/request", nil)

	httpTestResponseRecorder := httptest.NewRecorder()

	// Act
	handlerDefault.ServeHTTP(httpTestResponseRecorder, req)

	// Assert
	assert.Equal(t, http.StatusNotImplemented, httpTestResponseRecorder.Code)
	assert.Equal(t, "No expectations in gozzmock for request!", httpTestResponseRecorder.Body.String())
}

func TestHandlerAddAndRemoveExpectation(t *testing.T) {
	// Arrange
	context := gzServer{
		logLevel: zerolog.DebugLevel}
	context.storage.init()
	expectedExp := Expectation{Key: "k"}
	expectedExps := Expectations{expectedExp.Key: expectedExp}
	expsjson := jsonMarshalMust(expectedExps)

	// Act
	bodyAddExpectation := context.addExpectation(t, expectedExp)
	bodyRemoveExpectation := context.removeExpectation(t, expectedExp.Key)

	// Assert
	assert.Equal(t, string(expsjson), bodyAddExpectation.String())
	assert.Equal(t, "{}", bodyRemoveExpectation.String())
}

func TestHandlerAddTwoExpectations(t *testing.T) {
	// Arrange
	context := gzServer{logLevel: zerolog.DebugLevel}
	context.storage.init()
	handlerDefault := http.HandlerFunc(context.HandlerDefault)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dumpRequest(r)
		w.Write([]byte("response from test server"))
	}))
	defer testServer.Close()
	testServerURL, err := url.Parse(testServer.URL)
	if err != nil {
		panic(err)
	}

	context.addExpectation(t, Expectation{
		Key:      "response",
		Request:  &ExpectationRequest{Path: "/response"},
		Response: &ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1})

	context.addExpectation(t, Expectation{
		Key:      "forward",
		Forward:  &ExpectationForward{Scheme: testServerURL.Scheme, Host: testServerURL.Host},
		Priority: 0})

	// do request for response
	reqToRespnse := httpNewRequestMust("POST", "/response", bytes.NewBuffer([]byte("request body")))
	reqToForward := httpNewRequestMust("POST", "/forward", bytes.NewBuffer([]byte("forward body")))

	httpTestResponseRecorderToResponse := httptest.NewRecorder()
	httpTestResponseRecorderToForward := httptest.NewRecorder()

	// Act
	handlerDefault.ServeHTTP(httpTestResponseRecorderToResponse, reqToRespnse)
	handlerDefault.ServeHTTP(httpTestResponseRecorderToForward, reqToForward)

	// Assert
	assert.Equal(t, http.StatusOK, httpTestResponseRecorderToResponse.Code)
	assert.Equal(t, "response body", httpTestResponseRecorderToResponse.Body.String())

	assert.Equal(t, http.StatusOK, httpTestResponseRecorderToForward.Code)
	assert.Equal(t, "response from test server", httpTestResponseRecorderToForward.Body.String())
}

func TestHandlerGetExpectations(t *testing.T) {
	// Arrange
	context := gzServer{logLevel: zerolog.DebugLevel}
	context.storage.init()
	handlerGetExpectations := http.HandlerFunc(context.HandlerGetExpectations)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response from test server"))
	}))
	defer testServer.Close()

	expectation := Expectation{
		Key:      "response",
		Request:  &ExpectationRequest{Path: "/response"},
		Response: &ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1}
	context.addExpectation(t, expectation)

	expectedResponse := jsonMarshalMust(expectation)

	// do request for response
	req := httpNewRequestMust("GET", "/gozzmock/get_expectations", nil)

	httpTestResponseRecorder := httptest.NewRecorder()

	// Act
	handlerGetExpectations.ServeHTTP(httpTestResponseRecorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)
	assert.Contains(t, httpTestResponseRecorder.Body.String(), string(expectedResponse))
}

func TestHandlerStatus(t *testing.T) {
	// Arrange
	context := gzServer{logLevel: zerolog.DebugLevel}
	context.storage.init()
	handlerStatus := http.HandlerFunc(context.HandlerStatus)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response from test server"))
	}))
	defer testServer.Close()

	req := httpNewRequestMust("GET", "/gozzmock/status", nil)

	httpTestResponseRecorder := httptest.NewRecorder()

	// Act
	handlerStatus.ServeHTTP(httpTestResponseRecorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)
	assert.Contains(t, "gozzmock status is OK", httpTestResponseRecorder.Body.String())
}

func TestHandlerForwardValidatrHeaders(t *testing.T) {
	// Arrange
	context := gzServer{logLevel: zerolog.DebugLevel}
	context.storage.init()
	handlerDefault := http.HandlerFunc(context.HandlerDefault)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprint(r.Host)))
	}))
	defer testServer.Close()
	testServerURL, err := url.Parse(testServer.URL)
	if err != nil {
		panic(err)
	}

	context.addExpectation(t, Expectation{
		Key:      "forward",
		Forward:  &ExpectationForward{Scheme: testServerURL.Scheme, Host: testServerURL.Host, Headers: &Headers{"Host": "fwd_host"}},
		Priority: 0})

	// do request for forward
	req := httpNewRequestMust("POST", "/forward", bytes.NewBuffer([]byte("forward body")))

	req.Host = "reqest_host"

	httpTestResponseRecorder := httptest.NewRecorder()

	// Act
	handlerDefault.ServeHTTP(httpTestResponseRecorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)
	assert.Equal(t, "fwd_host", httpTestResponseRecorder.Body.String())
}

func writeCompressedMessage(w http.ResponseWriter, message []byte) {
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	gz.Write(message)
}

func TestHandlerForwardReturnsGzip(t *testing.T) {
	// Arrange
	context := gzServer{logLevel: zerolog.DebugLevel}
	context.storage.init()
	handlerDefault := http.HandlerFunc(context.HandlerDefault)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCompressedMessage(w, []byte("response from test server"))
	}))
	defer testServer.Close()
	testServerURL, err := url.Parse(testServer.URL)
	if err != nil {
		panic(err)
	}

	context.addExpectation(t, Expectation{
		Key:      "forward",
		Forward:  &ExpectationForward{Scheme: testServerURL.Scheme, Host: testServerURL.Host},
		Priority: 0})

	// do request for forward
	req := httpNewRequestMust("POST", "/forward", bytes.NewBuffer([]byte("forward body")))

	req.Header.Add("Accept-Encoding", "gzip")

	httpTestResponseRecorder := httptest.NewRecorder()
	// Act
	handlerDefault.ServeHTTP(httpTestResponseRecorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)

	reader, err := gzip.NewReader(httpTestResponseRecorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "response from test server", string(body))
}

func TestHandlerRespondsWithJsTemplate(t *testing.T) {
	// Arrange
	context := gzServer{logLevel: zerolog.DebugLevel}
	context.storage.init()

	handlerDefault := http.HandlerFunc(context.HandlerDefault)
	jsTemplate := `"123".length`

	expectation := Expectation{
		Key: "template",
		Response: &ExpectationResponse{HTTPCode: http.StatusOK,
			JsTemplate: base64.StdEncoding.EncodeToString([]byte(jsTemplate))},
		Priority: 1}
	context.addExpectation(t, expectation)

	req := httpNewRequestMust("GET", "/", nil)

	httpTestResponseRecorder := httptest.NewRecorder()

	// Act
	handlerDefault.ServeHTTP(httpTestResponseRecorder, req)

	// Assert
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)
	assert.Contains(t, httpTestResponseRecorder.Body.String(), "3")
}
