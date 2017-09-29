package main

import (
	"flag"
	"fmt"

	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func httpHandleFuncWithLogs(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		LogRequest(r)
		handler(w, r)
	}

	http.HandleFunc(pattern, wrappedHandler)
}

func setZeroLogLevel(logLevel string) {

	selectedLevel := zerolog.DebugLevel

	switch logLevel {
	case "debug":
		selectedLevel = zerolog.DebugLevel
	case "info":
		selectedLevel = zerolog.InfoLevel
	case "warn":
		selectedLevel = zerolog.WarnLevel
	case "error":
		selectedLevel = zerolog.ErrorLevel
	case "fatal":
		selectedLevel = zerolog.FatalLevel
	case "panic":
		selectedLevel = zerolog.PanicLevel
	}
	fmt.Println("set log level:", selectedLevel)
	zerolog.SetGlobalLevel(selectedLevel)
	zerolog.TimestampFieldName = "timestamp"
	log.Logger = log.With().Str("type", "gozzmock").Logger()

}

func main() {
	var initExpectations string
	flag.StringVar(&initExpectations, "expectations", "[]", "set initial expectations")
	var logLevel string
	flag.StringVar(&logLevel, "loglevel", "debug", "set log level: debug, info, warn, error, fatal, panic")
	flag.Parse()

	fmt.Println("initial expectations:", initExpectations)
	fmt.Println("loglevel:", logLevel)
	fmt.Println("tail:", flag.Args())

	setZeroLogLevel(logLevel)

	exps := ExpectationsFromString(initExpectations)

	storage := ControllerCreateStorage()
	for _, exp := range exps {
		storage.AddExpectation(exp.Key, exp)
	}

	http.HandleFunc("/gozzmock/status", storage.HandlerStatus)
	httpHandleFuncWithLogs("/gozzmock/add_expectation", storage.HandlerAddExpectation)
	httpHandleFuncWithLogs("/gozzmock/remove_expectation", storage.HandlerRemoveExpectation)
	httpHandleFuncWithLogs("/gozzmock/get_expectations", storage.HandlerGetExpectations)
	httpHandleFuncWithLogs("/", storage.HandlerDefault)
	http.ListenAndServe(":8080", nil)
}
