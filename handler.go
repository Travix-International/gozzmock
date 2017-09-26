package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// HandlerAddExpectation handler parses request and adds expectation to global expectations list
func HandlerAddExpectation(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("function", "HandlerAddExpectation").Logger()

	if r.Method != "POST" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}

	exp := ExpectationFromReadCloser(r.Body)

	var exps = ControllerAddExpectation(exp.Key, exp, nil)

	expsjson, err := json.Marshal(exps)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}
	w.Write(expsjson)
}

// HandlerRemoveExpectation handler parses request and deletes expectation from global expectations list
func HandlerRemoveExpectation(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("function", "HandlerRemoveExpectation").Logger()

	if r.Method != "POST" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}

	requestBody := ExpectationRemove{}
	bodyDecoder := json.NewDecoder(r.Body)
	err := bodyDecoder.Decode(&requestBody)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}
	defer r.Body.Close()

	var exps = ControllerRemoveExpectation(requestBody.Key, nil)
	expsjson, err := json.Marshal(exps)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}
	w.Write(expsjson)
}

// HandlerGetExpectations handler parses request and returns global expectations list
func HandlerGetExpectations(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("function", "HandlerGetExpectations").Logger()

	if r.Method != "GET" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}

	exps := ControllerGetExpectations(nil)
	expsjson, err := json.Marshal(exps)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}
	fmt.Fprint(w, string(expsjson))
}

// HandlerStatus handler returns applications status
func HandlerStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "gozzmock status is OK")
}

// HandlerDefault handler is an entry point for all incoming requests
func HandlerDefault(w http.ResponseWriter, r *http.Request) {
	generateResponse(w, ControllerTranslateRequestToExpectation(r))
}

func createResponseFromExpectation(w http.ResponseWriter, resp *ExpectationResponse) {
	// NOTE
	// Changing the header map after a call to WriteHeader (or
	// Write) has no effect unless the modified headers are
	// trailers.
	if resp.Headers != nil {
		for name, value := range *resp.Headers {
			w.Header().Set(name, value)
		}
	}
	w.WriteHeader(resp.HTTPCode)
	w.Write([]byte(resp.Body))
}

func generateResponse(w http.ResponseWriter, req *ExpectationRequest) {
	fLog := log.With().Str("function", "generateResponseToResponseWriter").Logger()

	storedExpectations := ControllerGetExpectations(nil)
	orderedStoredExpectations := ControllerSortExpectationsByPriority(storedExpectations)
	for i := 0; i < len(orderedStoredExpectations); i++ {
		exp := orderedStoredExpectations[i]

		if !ControllerRequestPassesFilter(req, exp.Request) {
			continue
		}

		time.Sleep(time.Second * exp.Delay)

		if exp.Response != nil {
			fLog.Info().Str("key", exp.Key).Msg("Apply response expectation")
			createResponseFromExpectation(w, exp.Response)
			return
		}

		if exp.Forward != nil {
			fLog.Info().Str("key", exp.Key).Msg("Apply forward expectation")
			httpReq := ControllerCreateHTTPRequest(req, exp.Forward)
			doHTTPRequest(w, httpReq)
			return
		}
	}
	fLog.Error().Msg("No expectations in gozzmock for request!")

	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("No expectations in gozzmock for request!"))
}

func readCompressed(body []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return ioutil.ReadAll(reader)
}

func logResponseBody(responseHeader http.Header, body []byte, fLog zerolog.Logger) error {
	if responseHeader.Get("Content-Encoding") == "gzip" {
		var err error
		body, err = readCompressed(body)
		if err != nil {
			return err
		}
	}

	fLog.Debug().Str("messagetype", "ResponseBody").Msg(string(body))
	return nil
}

// LogRequest dumps http request and writes content to log
func LogRequest(req *http.Request) {
	fLog := log.With().Str("function", "LogRequest").Logger()
	reqDumped, err := httputil.DumpRequest(req, true)
	if err != nil {
		fLog.Panic().Err(err)
		return
	}
	fLog.Debug().Str("messagetype", "Request").Msg(string(reqDumped))
}

func reportError(w http.ResponseWriter) {
	http.Error(w, "Something went wrong", http.StatusInternalServerError)
}

func doHTTPRequest(w http.ResponseWriter, httpReq *http.Request) {
	fLog := log.With().Str("function", "doHTTPRequest").Logger()

	if httpReq == nil {
		fLog.Panic().Msg("http.Request is nil")
		reportError(w)
		return
	}

	LogRequest(httpReq)

	httpClient := &http.Client{}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}

	var body bytes.Buffer
	_, err = io.Copy(&body, resp.Body)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}

	err = logResponseBody(resp.Header, body.Bytes(), fLog)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}

	// NOTE
	// Changing the header map after a call to WriteHeader (or
	// Write) has no effect unless the modified headers are
	// trailers.
	headers := *ControllerTranslateHTTPHeadersToExpHeaders(resp.Header)
	for name, value := range headers {
		w.Header().Set(name, value)
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body.Bytes())
}
