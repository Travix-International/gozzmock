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
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func jsonMarshalMust(v interface{}) []byte {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return encoded
}

func httpNewRequestMust(method, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	return req
}

func (context *Context) getExpectations(t *testing.T) *bytes.Buffer {
	handlerGetExpectations := http.HandlerFunc(context.HandlerGetExpectations)
	req := httpNewRequestMust("GET", "/gozzmock/get_expectations", nil)

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerGetExpectations.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)

	return httpTestResponseRecorder.Body
}

func (context *Context) addExpectation(t *testing.T, exp Expectation) *bytes.Buffer {
	handlerAddExpectation := http.HandlerFunc(context.HandlerAddExpectation)

	expJSON := jsonMarshalMust(exp)

	req := httpNewRequestMust("POST", "/gozzmock/add_expectation", bytes.NewBuffer(expJSON))

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerAddExpectation.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)
	assert.Equal(t, fmt.Sprintf("Expectation with key '%s' was added", exp.Key), httpTestResponseRecorder.Body.String())

	return context.getExpectations(t)
}

func (context *Context) removeExpectation(t *testing.T, expKey string) *bytes.Buffer {
	handlerRemoveExpectation := http.HandlerFunc(context.HandlerRemoveExpectation)

	expRemoveJSON := jsonMarshalMust(ExpectationRemove{Key: expKey})

	req := httpNewRequestMust("POST", "/gozzmock/remove_expectation", bytes.NewBuffer(expRemoveJSON))

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerRemoveExpectation.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)
	assert.Equal(t, fmt.Sprintf("Expectation with key '%s' was removed", expKey), httpTestResponseRecorder.Body.String())

	return context.getExpectations(t)
}

func TestHandlerNoExpectations(t *testing.T) {
	// Arrange
	context := Context{
		logLevel: zerolog.DebugLevel,
		storage:  ControllerCreateStorage()}
	handlerDefault := http.HandlerFunc(context.HandlerDefault)
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
	context := Context{
		logLevel: zerolog.DebugLevel,
		storage:  ControllerCreateStorage()}
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
	context := Context{
		logLevel: zerolog.DebugLevel,
		storage:  ControllerCreateStorage()}
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
	context := Context{
		logLevel: zerolog.DebugLevel,
		storage:  ControllerCreateStorage()}
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
	context := Context{
		logLevel: zerolog.DebugLevel,
		storage:  ControllerCreateStorage()}
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
	context := Context{
		logLevel: zerolog.InfoLevel,
		storage:  ControllerCreateStorage()}
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
	context := Context{
		logLevel: zerolog.DebugLevel,
		storage:  ControllerCreateStorage()}
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
	context := Context{
		logLevel: zerolog.DebugLevel,
		storage:  ControllerCreateStorage()}

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
