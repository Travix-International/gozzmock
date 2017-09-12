package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/rs/zerolog/log"
)

// HandlerAddExpectation handler parses request and adds expectation to global expectations list
func HandlerAddExpectation(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("function", "HandlerAddExpectation").Logger()

	if r.Method != "POST" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		return
	}

	exp := ExpectationFromReadCloser(r.Body)

	var exps = ControllerAddExpectation(exp.Key, exp, nil)

	expsjson, err := json.Marshal(exps)
	if err != nil {
		fLog.Panic().Err(err)
		return
	}
	w.Write(expsjson)
}

// HandlerRemoveExpectation handler parses request and deletes expectation from global expectations list
func HandlerRemoveExpectation(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("function", "HandlerRemoveExpectation").Logger()

	if r.Method != "POST" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		return
	}

	requestBody := ExpectationRemove{}
	bodyDecoder := json.NewDecoder(r.Body)
	err := bodyDecoder.Decode(&requestBody)
	if err != nil {
		fLog.Panic().Err(err)
		return
	}
	defer r.Body.Close()

	var exps = ControllerRemoveExpectation(requestBody.Key, nil)
	expsjson, err := json.Marshal(exps)
	if err != nil {
		fLog.Panic().Err(err)
		return
	}
	w.Write(expsjson)
}

// HandlerGetExpectations handler parses request and returns global expectations list
func HandlerGetExpectations(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("function", "HandlerGetExpectations").Logger()

	if r.Method != "GET" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		return
	}

	var exps = ControllerGetExpectations(nil)
	expsjson, err := json.Marshal(exps)
	if err != nil {
		fLog.Panic().Err(err)
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
	generateResponseToResponseWriter(w, ControllerTranslateRequestToExpectation(r))
}

func uploadResponseToResponseWriter(w http.ResponseWriter, resp *ExpectationResponse) {
	w.WriteHeader(resp.HTTPCode)
	w.Write([]byte(resp.Body))
	if resp.Headers != nil {
		for name, value := range *resp.Headers {
			w.Header().Set(name, value)
		}
	}
}

func generateResponseToResponseWriter(w http.ResponseWriter, req *ExpectationRequest) {
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
			uploadResponseToResponseWriter(w, exp.Response)
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

func readResponseBody(resp *http.Response) ([]byte, error) {
	var body []byte
	var err error
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		body, err = ioutil.ReadAll(reader)
	} else {
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
	}
	if err != nil {
		return nil, err
	}
	return body, err
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

func doHTTPRequest(w http.ResponseWriter, httpReq *http.Request) {
	fLog := log.With().Str("function", "doHTTPRequest").Logger()

	if httpReq == nil {
		fLog.Panic().Msg("http.Request is nil")
		return
	}

	httpClient := &http.Client{}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		fLog.Panic().Err(err)
		return
	}

	body, err := readResponseBody(resp)
	if err != nil {
		fLog.Panic().Err(err)
		return
	}

	fLog.Debug().Str("messagetype", "ResponseBody").Msg(string(body))

	w.WriteHeader(resp.StatusCode)
	w.Write(body)

	headers := *ControllerTranslateHTTPHeadersToExpHeaders(resp.Header)
	for name, value := range headers {
		w.Header().Set(name, value)
	}
}
