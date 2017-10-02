package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

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
func (storage *Storage) HandlerAddExpectation(w http.ResponseWriter, r *http.Request) {
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

	storage.AddExpectation(exp.Key, exp)
	storage.writeExpectationsToResponse(w)
}

// HandlerRemoveExpectation handler parses request and deletes expectation from global expectations list
func (storage *Storage) HandlerRemoveExpectation(w http.ResponseWriter, r *http.Request) {
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

	storage.RemoveExpectation(requestBody.Key)
	storage.writeExpectationsToResponse(w)
}

// HandlerGetExpectations handler parses request and returns global expectations list
func (storage *Storage) HandlerGetExpectations(w http.ResponseWriter, r *http.Request) {
	fLog := log.With().Str("function", "HandlerGetExpectations").Logger()

	if r.Method != "GET" {
		fLog.Panic().Msgf("Wrong method %s", r.Method)
		reportError(w)
		return
	}

	storage.writeExpectationsToResponse(w)
}

// HandlerStatus handler returns applications status
func (storage *Storage) HandlerStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "gozzmock status is OK")
}

// HandlerDefault handler is an entry point for all incoming requests
func (storage *Storage) HandlerDefault(w http.ResponseWriter, r *http.Request) {

	storage.generateResponse(w, ControllerTranslateRequestToExpectation(r))
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

func applyExpectation(exp Expectation, w http.ResponseWriter, req *ExpectationRequest) {
	fLog := log.With().Str("function", "applyExpectation").Str("key", exp.Key).Logger()

	if exp.Delay > 0 {
		fLog.Info().Msg(fmt.Sprintf("Delay %v sec", exp.Delay))
		time.Sleep(time.Second * exp.Delay)
	}

	if exp.Response != nil {
		fLog.Info().Msg("Apply response expectation")
		createResponseFromExpectation(w, exp.Response)
		return
	}

	if exp.Forward != nil {
		fLog.Debug().Msg("Apply forward expectation")
		httpReq := ControllerCreateHTTPRequest(req, exp.Forward)
		doHTTPRequest(w, httpReq)
		return
	}
}

func (storage *Storage) generateResponse(w http.ResponseWriter, req *ExpectationRequest) {
	fLog := log.With().Str("function", "generateResponseToResponseWriter").Logger()

	orderedStoredExpectations := storage.GetExpectationsOrderedByPriority()
	for i := 0; i < len(orderedStoredExpectations); i++ {
		exp := orderedStoredExpectations[i]

		if !ControllerRequestPassesFilter(req, exp.Request) {
			continue
		}

		applyExpectation(exp, w, req)
		return
	}
	fLog.Error().Msg("No expectations in gozzmock for request!")

	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("No expectations in gozzmock for request!"))
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

// drainBody is copied from dump.go
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

// dumpCompressedResponse is copied from dump.go
func dumpCompressedResponse(resp *http.Response, body bool) ([]byte, error) {
	var b bytes.Buffer
	var err error
	// emptyBody is an instance of empty reader.
	var emptyBody = ioutil.NopCloser(strings.NewReader(""))
	// errNoBody is a sentinel error value used by failureToReadBody so we
	// can detect that the lack of body was intentional.
	var errNoBody = errors.New("sentinel error value")
	save := resp.Body
	savecl := resp.ContentLength

	if !body {
		resp.Body = emptyBody
	} else if resp.Body == nil {
		resp.Body = emptyBody
	} else {
		save, resp.Body, err = drainBody(resp.Body)
		if err != nil {
			return nil, err
		}
	}
	err = resp.Write(&b)
	if err == errNoBody {
		err = nil
	}
	resp.Body = save
	resp.ContentLength = savecl
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// logResponse dumps http response and writes content to log
func logResponse(resp *http.Response) {
	fLog := log.With().Str("function", "LogResponse").Logger()
	var respDumped []byte
	var err error
	if resp.Header.Get("Content-Encoding") == "gzip" {
		respDumped, err = dumpCompressedResponse(resp, true)
	} else {
		respDumped, err = httputil.DumpResponse(resp, true)
	}
	if err != nil {
		fLog.Panic().Err(err)
		return
	}

	fLog.Debug().Str("messagetype", "DumpedResponse").Msg(string(respDumped))
}

func reportError(w http.ResponseWriter) {
	http.Error(w, "Gozzmock. Something went wrong", http.StatusInternalServerError)
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

	defer resp.Body.Close()

	logResponse(resp)

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
