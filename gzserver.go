package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type mockServer interface {
	status(w http.ResponseWriter, r *http.Request)
	add(w http.ResponseWriter, r *http.Request)
	remove(w http.ResponseWriter, r *http.Request)
	get(w http.ResponseWriter, r *http.Request)
	root(w http.ResponseWriter, r *http.Request)
}

// Context contains objects which shared between http handlers
type gzServer struct {
	logLevel zerolog.Level
	storage  mockStorage
}

func (server *gzServer) add(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("message_type", "HandlerAddExpectation").Logger()

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

	server.storage.add(exp.Key, exp)
	fmt.Fprintf(w, "Expectation with key '%s' was added", exp.Key)
}

// HandlerRemoveExpectation handler parses request and deletes expectation from global expectations list
func (server *gzServer) remove(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("message_type", "HandlerRemoveExpectation").Logger()

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

	server.storage.remove(requestBody.Key)
	fmt.Fprintf(w, "Expectation with key '%s' was removed", requestBody.Key)
}

func writeExpectationsToResponse(storage mockStorage, w http.ResponseWriter) {
	fLog := log.With().Str("message_type", "writeExpectationsToResponse").Logger()
	expsJSON, err := json.Marshal(storage.getOrdered())
	if err != nil {
		fLog.Panic().Err(err)
		reportError(w)
		return
	}
	w.Write(expsJSON)
}

// HandlerGetExpectations handler parses request and returns global expectations list
func (server *gzServer) get(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("message_type", "HandlerGetExpectations").Logger()

	if r.Method != "GET" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}

	writeExpectationsToResponse(server.storage, w)
}

// HandlerStatus handler returns applications status
func (server *gzServer) status(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "gozzmock status is OK")
}

// controllerStringPassesFilter validates whether the input string has filter string as substring or as a regex
func controllerStringPassesFilter(str string, filter string) bool {
	r, error := regexp.Compile("(?s)" + filter)
	if error != nil {
		return strings.Contains(str, filter)
	}
	return r.MatchString(str)
}

// controllerRequestPassesFilter validates whether the incoming request passes particular filter
func controllerRequestPassesFilter(req *ExpectationRequest, storedExpectation *ExpectationRequest) bool {
	fLog := log.With().Str("message_type", "controllerRequestPassesFilter").Logger()

	if storedExpectation == nil {
		fLog.Debug().Msg("Stored expectation.request is nil")
		return true
	}

	if len(storedExpectation.Method) > 0 && storedExpectation.Method != req.Method {
		fLog.Debug().Msgf("method %s should be %s", req.Method, storedExpectation.Method)
		return false
	}

	if len(storedExpectation.Path) > 0 && !controllerStringPassesFilter(req.Path, storedExpectation.Path) {
		fLog.Debug().Msgf("path %s doesn't pass filter %s", req.Path, storedExpectation.Path)
		return false
	}

	if len(storedExpectation.Body) > 0 && !controllerStringPassesFilter(req.Body, storedExpectation.Body) {
		fLog.Debug().Msgf("body %s doesn't pass filter %s", req.Body, storedExpectation.Body)
		return false
	}

	if storedExpectation.Headers != nil {
		if req.Headers == nil {
			fLog.Debug().Msgf("Request is expected to contain headers")
			return false
		}
		for storedHeaderName, storedHeaderValue := range *storedExpectation.Headers {
			value, ok := (*req.Headers)[storedHeaderName]
			if !ok {
				fLog.Debug().Msgf("No header %s in the request headers %v", storedHeaderName, req.Headers)
				return false
			}
			if !controllerStringPassesFilter(value, storedHeaderValue) {
				fLog.Debug().Msgf("header %s:%s has been rejected. Expected header value %s", storedHeaderName, value, storedHeaderValue)
				return false
			}
		}
	}

	return true
}

// controllerCreateHTTPRequest creates an http request based on incoming request and forward rules
func controllerCreateHTTPRequest(req *ExpectationRequest, fwd *ExpectationForward) *http.Request {
	fLog := log.With().Str("message_type", "ControllerCreateHTTPRequest").Logger()

	fwdURL, err := url.Parse(fmt.Sprintf("%s://%s%s", fwd.Scheme, fwd.Host, req.Path))
	if err != nil {
		fLog.Panic().Err(err)
		return nil
	}
	fLog.Info().Msgf("Send request to %s", fwdURL)
	httpReq, err := http.NewRequest(req.Method, fwdURL.String(), bytes.NewBuffer([]byte(req.Body)))
	if err != nil {
		fLog.Panic().Err(err)
		return nil
	}

	if req.Headers != nil {
		for name, value := range *req.Headers {
			httpReq.Header.Set(name, value)
		}
	}

	if fwd.Headers != nil {
		for name, value := range *fwd.Headers {
			if name == "Host" {
				fLog.Debug().Msgf("Set host to %s in request", value)
				httpReq.Host = value
			} else {
				httpReq.Header.Set(name, value)
			}
		}
	}

	return httpReq
}

// translateHTTPHeadersToExpHeaders translates http headers into custom headers map
func translateHTTPHeadersToExpHeaders(httpHeader http.Header) *Headers {
	headers := Headers{}
	for name, headerLine := range httpHeader {
		headers[name] = strings.Join(headerLine, ",")
	}
	return &headers
}

// translateRequestToExpectation Translates http request to expectation request
func translateRequestToExpectation(r *http.Request) *ExpectationRequest {
	var expRequest = ExpectationRequest{}
	expRequest.Method = r.Method
	expRequest.Path = r.URL.RequestURI()

	if len(r.URL.Fragment) > 0 {
		expRequest.Path += "#" + r.URL.Fragment
	}

	// Buffer the body
	if r.Body != nil {
		bodyBuffer, error := ioutil.ReadAll(r.Body)
		if error == nil {
			expRequest.Body = string(bodyBuffer)
		}
	}

	if len(r.Header) > 0 {
		expRequest.Headers = translateHTTPHeadersToExpHeaders(r.Header)
	}

	return &expRequest
}

// HandlerDefault handler is an entry point for all incoming requests
func (server *gzServer) root(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("message_type", "generateResponseToResponseWriter").Logger()
	req := translateRequestToExpectation(r)

	orderedStoredExpectations := server.storage.getOrdered()
	for i := 0; i < len(orderedStoredExpectations); i++ {
		exp := orderedStoredExpectations[i]

		if !controllerRequestPassesFilter(req, exp.Request) {
			continue
		}

		server.applyExpectation(exp, w, req)
		return
	}
	fLog.Error().Msg("No expectations in gozzmock for request!")

	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("No expectations in gozzmock for request!"))
}

func (server *gzServer) applyExpectation(exp Expectation, w http.ResponseWriter, req *ExpectationRequest) {
	fLog := log.With().Str("message_type", "applyExpectation").Str("key", exp.Key).Logger()

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
		httpReq := controllerCreateHTTPRequest(req, exp.Forward)
		server.doHTTPRequest(w, httpReq)
		return
	}
}

func (server *gzServer) doHTTPRequest(w http.ResponseWriter, httpReq *http.Request) {
	fLog := log.With().Str("message_type", "doHTTPRequest").Logger()

	if httpReq == nil {
		fLog.Panic().Msg("http.Request is nil")
		reportError(w)
		return
	}

	if server.logLevel == zerolog.DebugLevel {
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

	if server.logLevel == zerolog.DebugLevel {
		dumpResponse(resp)
	}

	// NOTE
	// Changing the header map after a call to WriteHeader (or
	// Write) has no effect unless the modified headers are
	// trailers.
	headers := *translateHTTPHeadersToExpHeaders(resp.Header)
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

func reportError(w http.ResponseWriter) {
	http.Error(w, "Gozzmock. Something went wrong", http.StatusInternalServerError)
}

func createResponseFromExpectation(w http.ResponseWriter, resp *ExpectationResponse, req *ExpectationRequest) {
	// NOTE
	// Changing the header map after a call to WriteHeader (or
	// Write) has no effect unless the modified headers are
	// trailers.
	fLog := log.With().Str("message_type", "createResponseFromExpectation").Logger()

	if resp.Headers != nil {
		for name, value := range *resp.Headers {
			w.Header().Set(name, value)
		}
	}

	resposneBody := resp.Body
	if len(resp.JsTemplate) > 0 {
		var err error
		resposneBody, err = JsTemplateCreateResponseBody(resp.JsTemplate, req)
		if err != nil {
			fLog.Error().Err(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(resp.HTTPCode)
	w.Write([]byte(resposneBody))
}
