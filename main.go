package main

import (
	"flag"
	"fmt"

	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Context contains objects which shared between http handlers
type Context struct {
	logLevel zerolog.Level
	storage  *Storage
}

func (context *Context) httpHandleFuncHelper(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		if context.logLevel == zerolog.DebugLevel {
			dumpRequest(r)
		}

		handler(w, r)
		if r != nil && r.Body != nil {
			r.Body.Close()
		}
	}

	http.HandleFunc(pattern, wrappedHandler)
}

func (context *Context) setZeroLogLevel(logLevel string) {

	context.logLevel = zerolog.DebugLevel

	switch logLevel {
	case "debug":
		context.logLevel = zerolog.DebugLevel
	case "info":
		context.logLevel = zerolog.InfoLevel
	case "warn":
		context.logLevel = zerolog.WarnLevel
	case "error":
		context.logLevel = zerolog.ErrorLevel
	case "fatal":
		context.logLevel = zerolog.FatalLevel
	case "panic":
		context.logLevel = zerolog.PanicLevel
	}
	fmt.Println("set log level:", context.logLevel)
	zerolog.SetGlobalLevel(context.logLevel)
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

	context := &Context{}
	context.setZeroLogLevel(logLevel)

	exps := ExpectationsFromString(initExpectations)

	context.storage = ControllerCreateStorage()
	for _, exp := range exps {
		context.storage.AddExpectation(exp.Key, exp)
	}

	http.HandleFunc("/gozzmock/status", context.HandlerStatus)
	context.httpHandleFuncHelper("/gozzmock/add_expectation", context.HandlerAddExpectation)
	context.httpHandleFuncHelper("/gozzmock/remove_expectation", context.HandlerRemoveExpectation)
	context.httpHandleFuncHelper("/gozzmock/get_expectations", context.HandlerGetExpectations)
	context.httpHandleFuncHelper("/", context.HandlerDefault)
	http.ListenAndServe(":8080", nil)
}
