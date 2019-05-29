package main

import (
	"fmt"
	"io"
	"os"

	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

func main() {
	// JSON data with expectation
	initExpectations := os.Getenv("GOZ_EXPECTATIONS")

	// set path to file with expectations in JSON
	initExpectationJSONFile := os.Getenv("GOZ_EXPECTATIONS_FILE")

	// set log level: debug, info, warn, error, fatal, panic
	logLevel := os.Getenv("GOZ_LOGLEVEL")
	if len(logLevel) == 0 {
		logLevel = "debug"
	}

	closer := initJaeger()
	defer closer.Close()

	//default port to run: 8080
	port := os.Getenv("GOZ_PORT")
	if len(port) == 0 {
		port = "8080"
	}

	fmt.Println("Arguments:")
	fmt.Println("initial expectations:", initExpectations)
	fmt.Println("initial expectations from json file:", initExpectationJSONFile)
	fmt.Println("loglevel:", logLevel)
	fmt.Println("port:", port)

	server := newGzServer(logLevel)
	if len(initExpectations) > 2 {
		err := server.filter.AddFromString(initExpectations)
		if err != nil {
			panic(err)
		}
	}
	if len(initExpectationJSONFile) > 0 {
		err := server.filter.AddFromJSON(initExpectationJSONFile)
		if err != nil {
			panic(err)
		}
	}

	server.serve(port)
}

// initJaeger returns an instance of Jaeger Tracer that can be configured with environment variables
// https://github.com/jaegertracing/jaeger-client-go#environment-variables
func initJaeger() io.Closer {

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		panic(err)
	}

	if os.Getenv("JAEGER_AGENT_HOST") != "" {
		// get remote config from jaeger-agent running as daemonset
		if cfg != nil && cfg.Sampler != nil && os.Getenv("JAEGER_SAMPLER_MANAGER_HOST_PORT") == "" {
			cfg.Sampler.SamplingServerURL = fmt.Sprintf("http://%v:5778/sampling", os.Getenv("JAEGER_AGENT_HOST"))
		}
	}

	closer, err := cfg.InitGlobalTracer(cfg.ServiceName, jaegercfg.Logger(jaeger.StdLogger))

	if err != nil {
		panic(err)
	}

	return closer
}
