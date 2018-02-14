package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/rs/zerolog/log"
)

func (storage *Storage) writeExpectationsToResponse(w http.ResponseWriter) {
	fLog := log.With().Str("function", "writeExpectationsToResponse").Logger()
	expsJSON, err := storage.GetExpectationsJSON()
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}
	w.Write(expsJSON)
}

// HandlerAddExpectation handler parses request and adds expectation to global expectations list
func (context *Context) HandlerAddExpectation(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("function", "HandlerAddExpectation").Logger()

	if r.Method != "POST" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}

	exp := Expectation{}
	err := ObjectFromJSON(r.Body, &exp)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}

	expectationSetDefaultValues(&exp)

	context.storage.AddExpectation(exp.Key, exp)
	fmt.Fprintf(w, "Expectation with key '%s' was added", exp.Key)
}

// HandlerRemoveExpectation handler parses request and deletes expectation from global expectations list
func (context *Context) HandlerRemoveExpectation(w http.ResponseWriter, r *http.Request) {
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

	context.storage.RemoveExpectation(requestBody.Key)
	fmt.Fprintf(w, "Expectation with key '%s' was removed", requestBody.Key)
}

// HandlerGetExpectations handler parses request and returns global expectations list
func (context *Context) HandlerGetExpectations(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("function", "HandlerGetExpectations").Logger()

	if r.Method != "GET" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}

	context.storage.writeExpectationsToResponse(w)
}

// HandlerStatus handler returns applications status
func (context *Context) HandlerStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "gozzmock status is OK")
}

// HandlerDefault handler is an entry point for all incoming requests
func (context *Context) HandlerDefault(w http.ResponseWriter, r *http.Request) {

	context.generateResponse(w, ControllerTranslateRequestToExpectation(r))
}

func createResponseFromExpectation(w http.ResponseWriter, resp *ExpectationResponse, req *ExpectationRequest) {
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
	if len(resp.JsTemplate) > 0 {
		w.Write([]byte(JsTemplateCreateResponseBody(resp.JsTemplate, req)))
		return
	}
	w.Write([]byte(resp.Body))
}

func (context *Context) applyExpectation(exp Expectation, w http.ResponseWriter, req *ExpectationRequest) {
	fLog := log.With().Str("function", "applyExpectation").Str("key", exp.Key).Logger()

	if exp.Delay > 0 {
		fLog.Info().Msg(fmt.Sprintf("Delay %v sec", exp.Delay))
		time.Sleep(time.Second * exp.Delay)
	}

	if exp.Response != nil {
		fLog.Info().Msg("Apply response expectation")
		createResponseFromExpectation(w, exp.Response, req)
		return
	}

	if exp.Forward != nil {
		fLog.Debug().Msg("Apply forward expectation")
		httpReq := ControllerCreateHTTPRequest(req, exp.Forward)
		context.doHTTPRequest(w, httpReq)
		return
	}
}

func (context *Context) generateResponse(w http.ResponseWriter, req *ExpectationRequest) {
	fLog := log.With().Str("function", "generateResponseToResponseWriter").Logger()

	orderedStoredExpectations := context.storage.GetExpectationsOrderedByPriority()
	for i := 0; i < len(orderedStoredExpectations); i++ {
		exp := orderedStoredExpectations[i]

		if !ControllerRequestPassesFilter(req, exp.Request) {
			continue
		}

		context.applyExpectation(exp, w, req)
		return
	}
	fLog.Error().Msg("No expectations in gozzmock for request!")

	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("No expectations in gozzmock for request!"))
}

func reportError(w http.ResponseWriter) {
	http.Error(w, "Gozzmock. Something went wrong", http.StatusInternalServerError)
}

func (context *Context) doHTTPRequest(w http.ResponseWriter, httpReq *http.Request) {
	fLog := log.With().Str("function", "doHTTPRequest").Logger()

	if httpReq == nil {
		fLog.Panic().Msg("http.Request is nil")
		reportError(w)
		return
	}

	if context.logLevel == zerolog.DebugLevel {
		dumpRequest(httpReq)
	}

	httpClient := &http.Client{}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}

	defer resp.Body.Close()

	if context.logLevel == zerolog.DebugLevel {
		dumpResponse(resp)
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

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}
}
