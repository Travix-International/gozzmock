package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

// ControllerCreateStorage creates storage for expectations
func ControllerCreateStorage() *Storage {
	return &Storage{expectations: make(Expectations)}
}

// GetExpectationsJSON returns list with expectations in json format
func (storage *Storage) GetExpectationsJSON() ([]byte, error) {
	storage.mu.RLock()
	defer storage.mu.RUnlock()
	return json.Marshal(storage.expectations)
}

// GetExpectationsStructure returns map with int keys sorted by priority DESC.
// 0-indexed element has the highest priority
func (storage *Storage) GetExpectationsStructure() Expectations {
	copyExps := Expectations{}
	storage.mu.RLock()
	for _, exp := range storage.expectations {
		copyExps[exp.Key] = exp
	}
	storage.mu.RUnlock()
	return copyExps
}

// AddExpectation adds new expectation to list. If expectation with same key exists, updates it
func (storage *Storage) AddExpectation(key string, exp Expectation) {
	storage.mu.Lock()
	storage.expectations[key] = exp
	storage.mu.Unlock()
}

// RemoveExpectation removes expectation with particular key
func (storage *Storage) RemoveExpectation(key string) {
	storage.mu.RLock()
	_, ok := storage.expectations[key]
	storage.mu.RUnlock()

	if ok {
		storage.mu.Lock()
		delete(storage.expectations, key)
		storage.mu.Unlock()
	}
}

// GetExpectationsOrderedByPriority returns map with int keys sorted by priority DESC.
// 0-indexed element has the highest priority
func (storage *Storage) GetExpectationsOrderedByPriority() ExpectationsInt {
	listForSorting := ExpectationsInt{}
	i := 0
	storage.mu.RLock()
	for _, exp := range storage.expectations {
		listForSorting[i] = exp
		i++
	}
	storage.mu.RUnlock()
	sort.Sort(listForSorting)
	return listForSorting
}

// ControllerTranslateHTTPHeadersToExpHeaders translates http headers into custom headers map
func ControllerTranslateHTTPHeadersToExpHeaders(httpHeader http.Header) *Headers {
	headers := Headers{}
	for name, headerLine := range httpHeader {
		headers[name] = strings.Join(headerLine, ",")
	}
	return &headers
}

// ControllerTranslateRequestToExpectation Translates http request to expectation request
func ControllerTranslateRequestToExpectation(r *http.Request) *ExpectationRequest {
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
		expRequest.Headers = ControllerTranslateHTTPHeadersToExpHeaders(r.Header)
	}

	return &expRequest
}

// ControllerStringPassesFilter validates whether the input string has filter string as substring or as a regex
func ControllerStringPassesFilter(str string, filter string) bool {
	r, error := regexp.Compile("(?s)" + filter)
	if error != nil {
		return strings.Contains(str, filter)
	}
	return r.MatchString(str)
}

// ControllerRequestPassesFilter validates whether the incoming request passes particular filter
func ControllerRequestPassesFilter(req *ExpectationRequest, storedExpectation *ExpectationRequest) bool {
	fLog := log.With().Str("message_type", "ControllerRequestPassesFilter").Logger()

	if storedExpectation == nil {
		fLog.Debug().Msg("Stored expectation.request is nil")
		return true
	}

	if len(storedExpectation.Method) > 0 && storedExpectation.Method != req.Method {
		fLog.Debug().Msgf("method %s should be %s", req.Method, storedExpectation.Method)
		return false
	}

	if len(storedExpectation.Path) > 0 && !ControllerStringPassesFilter(req.Path, storedExpectation.Path) {
		fLog.Debug().Msgf("path %s doesn't pass filter %s", req.Path, storedExpectation.Path)
		return false
	}

	if len(storedExpectation.Body) > 0 && !ControllerStringPassesFilter(req.Body, storedExpectation.Body) {
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
			if !ControllerStringPassesFilter(value, storedHeaderValue) {
				fLog.Debug().Msgf("header %s:%s has been rejected. Expected header value %s", storedHeaderName, value, storedHeaderValue)
				return false
			}
		}
	}

	return true
}

// ControllerCreateHTTPRequest creates an http request based on incoming request and forward rules
func ControllerCreateHTTPRequest(req *ExpectationRequest, fwd *ExpectationForward) *http.Request {
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
