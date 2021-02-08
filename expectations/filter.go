package expectations

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Travix-International/gozzmock/httpclient"
	"github.com/robertkrimen/otto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Filter interface {
	Storer
	Apply(r *http.Request) *HttpResponse
}

type GzFilter struct {
	storage      Storer
	roundTripper http.RoundTripper
	logLevel     zerolog.Level
}

type HttpResponse struct {
	HTTPCode int     `json:"httpcode"`
	Body     []byte  `json:"body"`
	Headers  Headers `json:"headers,omitempty"`
}

func NewGzFilter(rt http.RoundTripper, storage Storer) *GzFilter {
	return &GzFilter{
		storage:      storage,
		roundTripper: rt,
	}
}

func (f *GzFilter) Add(exp Expectation) {
	f.storage.Add(exp)
}

func (f *GzFilter) AddFromJSON(file string) error {
	return f.storage.AddFromJSON(file)
}

func (f *GzFilter) AddFromString(str string) error {
	return f.storage.AddFromString(str)
}

func (f *GzFilter) Remove(key string) {
	f.storage.Remove(key)
}

func (f *GzFilter) GetOrdered() OrderedExpectations {
	return f.storage.GetOrdered()
}

func (f *GzFilter) Apply(r *http.Request) *HttpResponse {
	fLog := log.With().Str("messagetype", "generateResponseToResponseWriter").Logger()
	req, err := HttpRequestToExpectationRequest(r)
	if err != nil {
		return reportError()
	}

	orderedStoredExpectations := f.storage.GetOrdered()
	for i := 0; i < len(orderedStoredExpectations); i++ {
		exp := orderedStoredExpectations[i]

		if !expectationsMatch(req, exp.Request) {
			continue
		}

		return f.applyExpectation(exp, req)
	}

	fLog.Error().Msg("No expectations in gozzmock for request!")

	return &HttpResponse{
		HTTPCode: http.StatusNotImplemented,
		Body:     []byte("No expectations in gozzmock for request!"),
	}
}

func (f *GzFilter) applyExpectation(exp Expectation, req *ExpectationRequest) *HttpResponse {
	fLog := log.With().Str("messagetype", "applyExpectation").Str("key", exp.Key).Logger()

	if exp.Delay > 0 {
		fLog.Info().Msg(fmt.Sprintf("Delay %v sec", exp.Delay))
		time.Sleep(time.Second * exp.Delay)
	}

	if exp.Response != nil {
		fLog.Info().Msg("Apply response expectation")
		return responseFromExpectation(exp.Response, req)
	}

	if exp.Forward != nil {
		fLog.Debug().Msg("Apply forward expectation")
		return f.responseFromHTTPForward(req, exp.Forward)
	}

	return nil
}

func reportError() *HttpResponse {
	return &HttpResponse{
		HTTPCode: http.StatusInternalServerError,
		Body:     []byte("Gozzmock. Something went wrong"),
	}
}

func responseFromExpectation(exp *ExpectationResponse, req *ExpectationRequest) *HttpResponse {
	// NOTE
	// Changing the header map after a call to WriteHeader (or
	// Write) has no effect unless the modified headers are
	// trailers.
	fLog := log.With().Str("messagetype", "responseFromExpectation").Logger()

	resp := HttpResponse{HTTPCode: exp.HTTPCode}
	if exp.Headers != nil {
		for name, value := range exp.Headers {
			resp.Headers[name] = value
		}
	}

	resposneBody := exp.Body
	if len(exp.JsTemplate) > 0 {
		var err error
		resposneBody, err = runJsTemplate(exp.JsTemplate, req)
		if err != nil {
			resp.HTTPCode = http.StatusInternalServerError
			resp.Body = []byte(err.Error())
			fLog.Error().Err(err).Msg("")
			return &resp
		}
	}
	resp.Body = []byte(resposneBody)

	return &resp
}

// runJsTemplate creates response body as string based on template and incoming request
func runJsTemplate(encodedTmpl string, req *ExpectationRequest) (string, error) {
	if len(encodedTmpl) == 0 {
		return "", nil
	}

	decodedTmpl, err := base64.StdEncoding.DecodeString(encodedTmpl)
	if err != nil {
		return "", fmt.Errorf("Error decoding from base64 template %s \n %s", encodedTmpl, err.Error())
	}
	stringTmpl := string(decodedTmpl)

	vm := otto.New()
	vm.Set("request", req)
	value, err := vm.Run(stringTmpl)
	if err != nil {
		return "", fmt.Errorf("Error running template %s \n %s", stringTmpl, err.Error())
	}

	return value.String(), nil
}

func (f *GzFilter) doHTTPRequest(httpReq *http.Request) *HttpResponse {
	fLog := log.With().Str("messagetype", "doHTTPRequest").Logger()

	if httpReq == nil {
		fLog.Panic().Msg("http.Request is nil")
		return reportError()
	}

	if f.logLevel == zerolog.DebugLevel {
		httpclient.DumpRequest(
			log.With().Str("messagetype", "externalRequest").Logger(),
			httpReq)
	}

	httpResp, err := f.roundTripper.RoundTrip(httpReq)
	if err != nil {
		fLog.Panic().Err(err)
		return reportError()
	}

	if httpResp == nil {
		fLog.Panic().Msg("response is nil")
		return nil
	}
	defer httpResp.Body.Close()

	if f.logLevel == zerolog.DebugLevel {
		httpclient.DumpResponse(
			log.With().Str("messagetype", "externalResponse").Logger(),
			httpResp)
	}

	resp, err := toCustomHttpResponse(httpResp)
	if err != nil {
		fLog.Panic().Err(err).Msg("")
		return reportError()
	}

	return resp
}

// stringsMatch validates whether the input string has filter string as substring or as a regex
func stringsMatch(req string, exp string) bool {
	if len(exp) == 0 {
		return true
	}

	r, err := regexp.Compile("(?s)" + exp)
	if err != nil {
		return strings.Contains(req, exp)
	}
	return r.MatchString(req)
}

// findInMapCaseInsensitive looks up the specified key in the map using case-insensitive lookup
func findInMapCaseInsensitive(m map[string]string, k string) (string, bool) {
	for name, value := range m {
		if strings.EqualFold(name, k) {
			return value, true
		}
	}

	return "", false
}

// headersMatch validates whether the input string has filter string as substring or as a regex
func headersMatch(req Headers, exp Headers) bool {
	if len(exp) == 0 {
		return true
	}

	for expName, expValue := range exp {
		reqValue, ok := findInMapCaseInsensitive(req, expName)
		if ok && stringsMatch(reqValue, expValue) {
			continue
		}
		return false
	}
	return true
}

// expectationsMatch validates whether the incoming request passes particular filter
func expectationsMatch(req *ExpectationRequest, exp *ExpectationRequest) bool {
	fLog := log.With().Str("messagetype", "controllerRequestPassesFilter").Logger()

	if exp == nil {
		fLog.Debug().Msg("Match. Expectation is nil")
		return true
	}

	if len(exp.Method) > 0 && exp.Method != req.Method {
		fLog.Debug().Msgf("No match. Request method %s != %s", req.Method, exp.Method)
		return false
	}

	if !stringsMatch(req.Path, exp.Path) {
		fLog.Debug().Msgf("No match. Request path %s doesn't match %s", req.Path, exp.Path)
		return false
	}

	if !stringsMatch(string(req.Body), string(exp.Body)) {
		fLog.Debug().Msgf("No match. Request body %s doesn't match %s", req.Body, exp.Body)
		return false
	}

	if !headersMatch(req.Headers, exp.Headers) {
		fLog.Debug().Msgf("No match. Request headers %v doesn't match %v", req.Headers, exp.Headers)
		return false
	}

	return true
}

// responseFromHTTPForward creates an http request based on incoming request and forward rules
func (f *GzFilter) responseFromHTTPForward(req *ExpectationRequest, fwd *ExpectationForward) *HttpResponse {
	fLog := log.With().Str("messagetype", "responseFromHTTPForward").Logger()

	fwdURL, err := url.Parse(fmt.Sprintf("%s://%s%s", fwd.Scheme, fwd.Host, req.Path))
	if err != nil {
		fLog.Panic().Err(err).Msg("")
		return nil
	}
	fLog.Info().Msgf("Send request to %s", fwdURL)
	httpReq, err := http.NewRequest(req.Method, fwdURL.String(), bytes.NewBuffer([]byte(req.Body)))
	if err != nil {
		fLog.Panic().Err(err)
		return nil
	}

	if len(req.Headers) > 0 {
		for name, value := range req.Headers {
			httpReq.Header.Set(name, value)
		}
	}

	if len(fwd.Headers) > 0 {
		for name, value := range fwd.Headers {
			if name == "Host" {
				fLog.Debug().Msgf("Set host to %s in request", value)
				httpReq.Host = value
			} else {
				httpReq.Header.Set(name, value)
			}
		}
	}

	return f.doHTTPRequest(httpReq)
}

func toCustomHttpResponse(httpResp *http.Response) (*HttpResponse, error) {

	resp := HttpResponse{
		HTTPCode: httpResp.StatusCode,
		Headers:  Headers{},
	}
	for name, headerLine := range httpResp.Header {
		resp.Headers[name] = strings.Join(headerLine, ",")
	}

	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = body

	return &resp, nil
}
