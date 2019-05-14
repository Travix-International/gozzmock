package expectations

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Headers are HTTP headers
type Headers map[string]string

// ExpectationRequest is filter for incoming requests
type ExpectationRequest struct {
	Method  string  `json:"method"`
	Path    string  `json:"path"`
	Body    string  `json:"body"`
	Headers Headers `json:"headers,omitempty"`
}

// ExpectationForward is forward action if request passes filter
type ExpectationForward struct {
	Scheme  string  `json:"scheme"`
	Host    string  `json:"host"`
	Headers Headers `json:"headers,omitempty"`
}

// ExpectationResponse is response action if request passes filter
type ExpectationResponse struct {
	HTTPCode   int     `json:"httpcode"`
	Body       string  `json:"body"`
	Headers    Headers `json:"headers,omitempty"`
	JsTemplate string  `json:"jstemplate,omitempty"`
}

// Expectation is single set of rules: expected request and prepared action
type Expectation struct {
	Key      string               `json:"key"`
	Request  *ExpectationRequest  `json:"request,omitempty"`
	Forward  *ExpectationForward  `json:"forward,omitempty"`
	Response *ExpectationResponse `json:"response,omitempty"`
	Delay    time.Duration        `json:"delay,omitempty"`
	Priority int                  `json:"priority,omitempty"`
}

// ExpectationRemove removes action from list by key
type ExpectationRemove struct {
	Key string `json:"key"`
}

// Expectations is a map for expectations
type Expectations map[string]Expectation

// setDefaultValues sets default values after deserialization
func (exp *Expectation) setDefaultValues() {
	if exp.Forward != nil && exp.Forward.Scheme == "" {
		exp.Forward.Scheme = "http"
	}
}

// Storer interface describes expectations storage functionality
type Storer interface {
	Add(exp Expectation)
	AddFromJSON(file string) error
	AddFromString(str string) error
	Remove(key string)
	GetOrdered() OrderedExpectations
}

// gzStorage is a structure with mutex to control access to expectations
type gzStorage struct {
	expectations Expectations
	mu           sync.RWMutex
}

// NewGzStorage is gzStorage constructor
func NewGzStorage() Storer {
	return &gzStorage{expectations: make(Expectations)}
}

// Add a new expectation to list. If expectation with same key exists, updates it
func (storage *gzStorage) Add(exp Expectation) {
	storage.mu.Lock()
	storage.expectations[exp.Key] = exp
	storage.mu.Unlock()
}

// AddFromJSON adds expectation from json file
func (storage *gzStorage) AddFromJSON(file string) error {
	var exps []Expectation

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	err = json.NewDecoder(bytes.NewReader(data)).Decode(&exps)
	if err != nil {
		return err
	}
	for _, exp := range exps {
		exp.setDefaultValues()
		storage.Add(exp)
	}
	return nil
}

// AddFromString adds expectation from string
func (storage *gzStorage) AddFromString(file string) error {
	var exps []Expectation

	err := json.NewDecoder(strings.NewReader(file)).Decode(&exps)
	if err != nil {
		return err
	}
	for _, exp := range exps {
		exp.setDefaultValues()
		storage.Add(exp)
	}
	return nil
}

// Remove removes expectation with particular key
func (storage *gzStorage) Remove(key string) {
	storage.mu.RLock()
	_, ok := storage.expectations[key]
	storage.mu.RUnlock()

	if ok {
		storage.mu.Lock()
		delete(storage.expectations, key)
		storage.mu.Unlock()
	}
}

// OrderedExpectations is for sorting expectations by priority. the lowest priority is 0
type OrderedExpectations map[int]Expectation

func (exps OrderedExpectations) Len() int           { return len(exps) }
func (exps OrderedExpectations) Swap(i, j int)      { exps[i], exps[j] = exps[j], exps[i] }
func (exps OrderedExpectations) Less(i, j int) bool { return exps[i].Priority > exps[j].Priority }

// GetOrdered returns map with int keys sorted by priority DESC.
// 0-indexed element has the highest priority
func (storage *gzStorage) GetOrdered() OrderedExpectations {
	listForSorting := OrderedExpectations{}
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

// HttpRequestToExpectationRequest Translates http request to expectation request
func HttpRequestToExpectationRequest(r *http.Request) (*ExpectationRequest, error) {
	var expRequest = ExpectationRequest{}
	expRequest.Method = r.Method
	expRequest.Path = r.URL.RequestURI()

	if len(r.URL.Fragment) > 0 {
		expRequest.Path += "#" + r.URL.Fragment
	}

	if r.Body != nil {
		bodyContent, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		expRequest.Body = string(bodyContent)
	}

	if len(r.Header) > 0 {
		expRequest.Headers = Headers{}
		for name, headerLine := range r.Header {
			expRequest.Headers[name] = strings.Join(headerLine, ",")
		}
	}

	return &expRequest, nil
}

// HttpRequestToExpectationRemove Translates http request to expectationRemove
func HttpRequestToExpectationRemove(r *http.Request) (*ExpectationRemove, error) {
	expRemove := ExpectationRemove{}

	err := json.NewDecoder(r.Body).Decode(&expRemove)
	if err != nil {
		return nil, err
	}

	return &expRemove, nil
}

// HttpRequestToExpectation Translates http request to expectation
func HttpRequestToExpectation(r *http.Request) (*Expectation, error) {
	exp := Expectation{}
	err := json.NewDecoder(r.Body).Decode(&exp)
	if err != nil {
		return nil, err
	}
	exp.setDefaultValues()

	return &exp, nil
}
