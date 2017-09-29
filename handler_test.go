package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"net/url"

	"github.com/stretchr/testify/assert"
)

func (storage *Storage) addExpectation(t *testing.T, exp Expectation) *bytes.Buffer {
	handlerAddExpectation := http.HandlerFunc(storage.HandlerAddExpectation)

	expJSON, err := json.Marshal(exp)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", "/gozzmock/add_expectation", bytes.NewBuffer(expJSON))
	if err != nil {
		t.Fatal(err)
	}

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerAddExpectation.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)

	return httpTestResponseRecorder.Body
}

func (storage *Storage) removeExpectation(t *testing.T, expKey string) *bytes.Buffer {
	handlerAddExpectation := http.HandlerFunc(storage.HandlerRemoveExpectation)

	expRemoveJSON, err := json.Marshal(ExpectationRemove{Key: expKey})
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "/gozzmock/remove_expectation", bytes.NewBuffer(expRemoveJSON))
	if err != nil {
		t.Fatal(err)
	}

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerAddExpectation.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)

	return httpTestResponseRecorder.Body
}

func TestHandlerNoExpectations(t *testing.T) {
	storage := ControllerCreateStorage()
	handlerDefault := http.HandlerFunc(storage.HandlerDefault)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response from test server"))
	}))
	defer testServer.Close()

	// do request for response
	req, err := http.NewRequest("GET", "/request", nil)
	if err != nil {
		t.Fatal(err)
	}

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerDefault.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusNotImplemented, httpTestResponseRecorder.Code)
	assert.Equal(t, "No expectations in gozzmock for request!", httpTestResponseRecorder.Body.String())
}

func TestHandlerAddAndRemoveExpectation(t *testing.T) {
	storage := ControllerCreateStorage()
	handlerRemoveExpectation := http.HandlerFunc(storage.HandlerRemoveExpectation)
	expectedExp := Expectation{Key: "k"}
	expectedExps := Expectations{expectedExp.Key: expectedExp}

	body := storage.addExpectation(t, expectedExp)
	expsjson, err := json.Marshal(expectedExps)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, string(expsjson), body.String())

	// remove expectation
	expRemoveJSON, err := json.Marshal(ExpectationRemove{Key: expectedExp.Key})
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", "/gozzmock/remove_expectation", bytes.NewBuffer(expRemoveJSON))
	if err != nil {
		t.Fatal(err)
	}

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerRemoveExpectation.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)

	assert.Equal(t, "{}", httpTestResponseRecorder.Body.String())
}

func TestHandlerAddTwoExpectations(t *testing.T) {
	storage := ControllerCreateStorage()
	handlerDefault := http.HandlerFunc(storage.HandlerDefault)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		LogRequest(r)
		w.Write([]byte("response from test server"))
	}))
	defer testServer.Close()
	testServerURL, err := url.Parse(testServer.URL)
	if err != nil {
		panic(err)
	}

	storage.addExpectation(t, Expectation{
		Key:      "response",
		Request:  &ExpectationRequest{Path: "/response"},
		Response: &ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1})

	storage.addExpectation(t, Expectation{
		Key:      "forward",
		Forward:  &ExpectationForward{Scheme: testServerURL.Scheme, Host: testServerURL.Host},
		Priority: 0})

	// do request for response
	req, err := http.NewRequest("POST", "/response", bytes.NewBuffer([]byte("request body")))
	if err != nil {
		t.Fatal(err)
	}

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerDefault.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)

	assert.Equal(t, "response body", httpTestResponseRecorder.Body.String())

	// do request for forward
	req, err = http.NewRequest("POST", "/forward", bytes.NewBuffer([]byte("forward body")))
	if err != nil {
		t.Fatal(err)
	}

	httpTestResponseRecorder2 := httptest.NewRecorder()
	handlerDefault.ServeHTTP(httpTestResponseRecorder2, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder2.Code)

	assert.Equal(t, "response from test server", httpTestResponseRecorder2.Body.String())
}

func TestHandlerGetExpectations(t *testing.T) {
	storage := ControllerCreateStorage()
	handlerGetExpectations := http.HandlerFunc(storage.HandlerGetExpectations)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response from test server"))
	}))
	defer testServer.Close()

	expectation := Expectation{
		Key:      "response",
		Request:  &ExpectationRequest{Path: "/response"},
		Response: &ExpectationResponse{HTTPCode: http.StatusOK, Body: "response body"},
		Priority: 1}
	storage.addExpectation(t, expectation)

	// do request for response
	req, err := http.NewRequest("GET", "/gozzmock/get_expectations", nil)
	if err != nil {
		t.Fatal(err)
	}

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerGetExpectations.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)

	expectedResponse, err := json.Marshal(expectation)
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, httpTestResponseRecorder.Body.String(), string(expectedResponse))
}

func TestHandlerStatus(t *testing.T) {
	storage := ControllerCreateStorage()
	handlerStatus := http.HandlerFunc(storage.HandlerStatus)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response from test server"))
	}))
	defer testServer.Close()

	// do request for response
	req, err := http.NewRequest("GET", "/gozzmock/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerStatus.ServeHTTP(httpTestResponseRecorder, req)
	assert.Equal(t, http.StatusOK, httpTestResponseRecorder.Code)
	assert.Contains(t, "gozzmock status is OK", httpTestResponseRecorder.Body.String())
}

func TestHandlerForwardValidatrHeaders(t *testing.T) {
	storage := ControllerCreateStorage()
	handlerDefault := http.HandlerFunc(storage.HandlerDefault)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprint(r.Host)))
	}))
	defer testServer.Close()
	testServerURL, err := url.Parse(testServer.URL)
	if err != nil {
		panic(err)
	}

	storage.addExpectation(t, Expectation{
		Key:      "forward",
		Forward:  &ExpectationForward{Scheme: testServerURL.Scheme, Host: testServerURL.Host, Headers: &Headers{"Host": "fwd_host"}},
		Priority: 0})

	// do request for forward
	req, err := http.NewRequest("POST", "/forward", bytes.NewBuffer([]byte("forward body")))
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "reqest_host"

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerDefault.ServeHTTP(httpTestResponseRecorder, req)
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
	storage := ControllerCreateStorage()
	handlerDefault := http.HandlerFunc(storage.HandlerDefault)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCompressedMessage(w, []byte("response from test server"))
	}))
	defer testServer.Close()
	testServerURL, err := url.Parse(testServer.URL)
	if err != nil {
		panic(err)
	}

	storage.addExpectation(t, Expectation{
		Key:      "forward",
		Forward:  &ExpectationForward{Scheme: testServerURL.Scheme, Host: testServerURL.Host},
		Priority: 0})

	// do request for forward
	req, err := http.NewRequest("POST", "/forward", bytes.NewBuffer([]byte("forward body")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Accept-Encoding", "gzip")

	httpTestResponseRecorder := httptest.NewRecorder()
	handlerDefault.ServeHTTP(httpTestResponseRecorder, req)
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
