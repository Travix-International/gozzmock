package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Headers are HTTP headers
type Headers map[string]string

// ExpectationRequest is filter for incoming requests
type ExpectationRequest struct {
	Method  string  `json:"method"`
	Path    string  `json:"path"`
	Body    string  `json:"body"`
	Headers Headers `json:"headers,omitempty"`
}

// ExpectationForward is forward action if request passes filter
type ExpectationForward struct {
	Scheme  string  `json:"scheme"`
	Host    string  `json:"host"`
	Headers Headers `json:"headers,omitempty"`
}

// ExpectationResponse is response action if request passes filter
type ExpectationResponse struct {
	HTTPCode   int     `json:"httpcode"`
	Body       string  `json:"body"`
	Headers    Headers `json:"headers,omitempty"`
	JsTemplate string  `json:"jstemplate,omitempty"`
}

// Expectation is single set of rules: expected request and prepared action
type Expectation struct {
	Key      string               `json:"key"`
	Request  *ExpectationRequest  `json:"request,omitempty"`
	Forward  *ExpectationForward  `json:"forward,omitempty"`
	Response *ExpectationResponse `json:"response,omitempty"`
	Delay    time.Duration        `json:"delay,omitempty"`
	Priority int                  `json:"priority,omitempty"`
}

// ExpectationRemove removes action from list by key
type ExpectationRemove struct {
	Key string `json:"key"`
}

// Expectations is a map for expectations
type Expectations map[string]Expectation

// ObjectFromJSON decodes json to object
func ObjectFromJSON(reader io.Reader, v interface{}) error {
	bodyDecoder := json.NewDecoder(reader)
	return bodyDecoder.Decode(v)
}

// ExpectationsFromString decodes string with array of expectations to array of expectation objects
func ExpectationsFromString(str string) []Expectation {

	var exps []Expectation

	err := ObjectFromJSON(strings.NewReader(str), &exps)
	if err != nil {
		panic(err)
	}
	for _, exp := range exps {
		exp.setDefaultValues()
	}
	return exps
}

// ExpectationsFromJSONFile decodes json file content to expectations
func ExpectationsFromJSONFile(file string) []Expectation {
	var exps []Expectation

	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	err = ObjectFromJSON(bytes.NewReader(data), &exps)
	if err != nil {
		panic(err)
	}
	for _, exp := range exps {
		exp.setDefaultValues()
	}
	return exps
}

// expectationSetDefaultValues sets default values after deserialization
func (exp *Expectation) setDefaultValues() {
	if exp.Forward != nil && exp.Forward.Scheme == "" {
		exp.Forward.Scheme = "http"
	}
}

func httpHandleWrapper(zloglevel zerolog.Level, pattern string, handler func(http.ResponseWriter, *http.Request)) {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		if zloglevel == zerolog.DebugLevel {
			dumpRequest(r)
		}

		handler(w, r)
		if r != nil && r.Body != nil {
			r.Body.Close()
		}
	}

	http.HandleFunc(pattern, wrappedHandler)
}

func toZeroLogLevel(logLevel string) zerolog.Level {

	zlogLevel := zerolog.DebugLevel

	switch logLevel {
	case "debug":
		zlogLevel = zerolog.DebugLevel
	case "info":
		zlogLevel = zerolog.InfoLevel
	case "warn":
		zlogLevel = zerolog.WarnLevel
	case "error":
		zlogLevel = zerolog.ErrorLevel
	case "fatal":
		zlogLevel = zerolog.FatalLevel
	case "panic":
		zlogLevel = zerolog.PanicLevel
	}
	fmt.Println("set log level:", zlogLevel)
	zerolog.SetGlobalLevel(zlogLevel)
	log.Logger = log.Output(V3FormatWriter{Out: os.Stderr}).
		With().Str("app_name", "gozzmock").Logger()
	return zlogLevel
}

func main() {
	// JSON data with expectation
	initExpectations := os.Getenv("EXPECTATIONS")
	
	// set path to file with expectations in JSON
	initExpectationJSONFile := os.Getenv("EXPECTATIONSFile")
	
	// set log level: debug, info, warn, error, fatal, panic
	logLevel := os.Getenv("LOGLEVEL")
	if len(logLevel) == 0 {
		logLevel = "debug"
	}

	//default port to run: 8080
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}


	fmt.Println("Arguments:")
	fmt.Println("initial expectations:", initExpectations)
	fmt.Println("initial expectations from json file:", initExpectationJSONFile)
	fmt.Println("loglevel:", logLevel)
	fmt.Println("port:", port)
	
	server := newGzServer()
	server.logLevel = toZeroLogLevel(logLevel)

	var exps []Expectation
	if len(initExpectations) > 2 {
		exps = ExpectationsFromString(initExpectations)
		fmt.Println("loaded expecations from string:", len(exps))
	}
	if len(initExpectationJSONFile) > 0 {
		expsFromFile := ExpectationsFromJSONFile(initExpectationJSONFile)
		fmt.Println("loaded expecations from file:", len(expsFromFile))
		exps = append(exps, expsFromFile...)
	}

	for _, exp := range exps {
		server.storage.add(exp.Key, exp)
	}

	http.HandleFunc("/gozzmock/status", server.status)
	http.Handle("/metrics", promhttp.Handler())
	httpHandleWrapper(server.logLevel, "/gozzmock/add_expectation", server.add)
	httpHandleWrapper(server.logLevel, "/gozzmock/remove_expectation", server.remove)
	httpHandleWrapper(server.logLevel, "/gozzmock/get_expectations", server.get)
	httpHandleWrapper(server.logLevel, "/", server.root)
	http.ListenAndServe(":"+port, nil)
}
