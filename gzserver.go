package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Travix-International/gozzmock/expectations"
	"github.com/Travix-International/gozzmock/httpclient"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Context contains objects which shared between http handlers
type gzServer struct {
	logLevel zerolog.Level
	filter   expectations.Filter
}

func newGzServer(logLevel string) *gzServer {

	return &gzServer{
		logLevel: toZeroLogLevel(logLevel),
		filter:   expectations.NewGzFilter(httpclient.NewRoundTripper(), expectations.NewGzStorage()),
	}
}

func reportError(w http.ResponseWriter) {
	http.Error(w, "Gozzmock. Something went wrong", http.StatusInternalServerError)
}

func (s *gzServer) add(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("messagetype", "HandlerAddExpectation").Logger()

	if r.Method != "POST" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}

	exp, err := expectations.HttpRequestToExpectation(r)
	if err != nil {
		fLog.Panic().Err(err).Msg("Error with assembling the expectation")
		reportError(w)
		return
	}

	s.filter.Add(*exp)
	fmt.Fprintf(w, "Expectation with key '%s' was added", exp.Key)
}

// HandlerRemoveExpectation handler parses request and deletes expectation from global expectations list
func (s *gzServer) remove(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("messagetype", "HandlerRemoveExpectation").Logger()

	if r.Method != "POST" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}
	expRemove, err := expectations.HttpRequestToExpectationRemove(r)
	if err != nil {
		fLog.Panic().Err(err).Msg("")
		reportError(w)
		return
	}

	s.filter.Remove(expRemove.Key)
	fmt.Fprintf(w, "Expectation with key '%s' was removed", expRemove.Key)
}

// HandlerGetExpectations handler parses request and returns global expectations list
func (s *gzServer) get(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("messagetype", "HandlerGetExpectations").Logger()

	if r.Method != "GET" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}

	expsJSON, err := json.Marshal(s.filter.GetOrdered())
	if err != nil {
		fLog.Panic().Err(err).Msg("Error getting expectations")
		reportError(w)
		return
	}
	w.Write(expsJSON)
}

// HandlerStatus handler returns applications status
func (s *gzServer) status(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "gozzmock status is OK")
}

// HandlerDefault handler is an entry point for all incoming requests
func (s *gzServer) root(w http.ResponseWriter, r *http.Request) {

	// NOTE
	// Changing the header map after a call to WriteHeader (or
	// Write) has no effect unless the modified headers are
	// trailers.

	resp := s.filter.Apply(r)

	for name, value := range resp.Headers {
		w.Header().Set(name, value)
	}
	w.WriteHeader(resp.HTTPCode)
	w.Write(resp.Body)
}

func (s *gzServer) handle(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		if s.logLevel == zerolog.DebugLevel {
			httpclient.DumpRequest(
				log.With().Str("messagetype", "request").Logger(),
				r)
		}

		handler(w, r)
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
		With().Str("appname", "gozzmock").Logger()
	return zlogLevel
}

func (s *gzServer) serve(port string) {
	http.HandleFunc("/gozzmock/status", s.status)
	http.Handle("/metrics", promhttp.Handler())
	s.handle("/gozzmock/add_expectation", s.add)
	s.handle("/gozzmock/remove_expectation", s.remove)
	s.handle("/gozzmock/get_expectations", s.get)
	s.handle("/", s.root)
	http.ListenAndServe(":"+port, nil)
}
