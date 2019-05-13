package main

import (
	"fmt"
	"os"
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
