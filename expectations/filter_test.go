package expectations

import (
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunJsTemplate_SimpleJsonEncodedBase64(t *testing.T) {
	expReq := &ExpectationRequest{
		Body: `{"a": [
			{"b": "bv1"}
			]}`}
	tmpl := []byte(`
	var response = {"response": JSON.parse(request.Body)["a"][0]["b"]};
	JSON.stringify(response);`)

	tmplEncoded := base64.StdEncoding.EncodeToString(tmpl)

	expectedOutput := `{"response":"bv1"}`

	// Act
	res, err := runJsTemplate(tmplEncoded, expReq)

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, expectedOutput, string(res))
}

func TestRunJsTemplate_WrongEncoding(t *testing.T) {
	expReq := &ExpectationRequest{
		Body: `{"a": [
				{"b": "bv1"}
				]}`}

	tmpl := `"abc"`

	// Act
	res, err := runJsTemplate(tmpl, expReq)

	// Assert
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Error decoding from base64 template")
	assert.Equal(t, "", res)
}

type mockedRoundTripper struct{}

func (rt *mockedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	response := &http.Response{StatusCode: 200, Header: http.Header{}}

	var sb strings.Builder
	sb.WriteString(req.URL.String())
	sb.WriteString(" Host:")
	sb.WriteString(req.Host)
	for k, v := range req.Header {
		sb.WriteString(" ")
		sb.WriteString(k)
		sb.WriteString(":")
		sv := strings.Join(v, ",")
		sb.WriteString(sv)
		response.Header.Add(k, sv)
	}
	response.Body = ioutil.NopCloser(strings.NewReader(sb.String()))
	return response, nil
}

func NewMockedGzFilter() *GzFilter {
	return NewGzFilter(&mockedRoundTripper{}, NewGzStorage())
}

func TestStringsMatch_EmptyFilter_True(t *testing.T) {
	assert.True(t, stringsMatch("abc", ""))
}

func TestStringsMatch_ExistingSubstring_True(t *testing.T) {
	assert.True(t, stringsMatch("abc", "ab"))
}

func TestStringsMatch_ExistingRegex_True(t *testing.T) {
	assert.True(t, stringsMatch("abc", ".b."))
}

func TestStringsMatch_NotExistingSubstring_False(t *testing.T) {
	assert.False(t, stringsMatch("abc", "zz"))
}

func TestStringsMatch_NotExistingRegex_False(t *testing.T) {
	assert.False(t, stringsMatch("abc", ".z."))
}

func TestStringsMatch_MultilineBody_True(t *testing.T) {
	assert.True(t, stringsMatch("a\nb", "a.b"))
}

func TestExpectationsMatch_EmptyRequestEmptyFilter_True(t *testing.T) {
	assert.True(t, expectationsMatch(
		&ExpectationRequest{},
		&ExpectationRequest{}))
}

func TestExpectationsMatch_MethodsAreEq_True(t *testing.T) {
	assert.True(t, expectationsMatch(
		&ExpectationRequest{Method: "POST"},
		&ExpectationRequest{Method: "POST"}))
}

func TestExpectationsMatch_PathsAreEq_True(t *testing.T) {
	assert.True(t, expectationsMatch(
		&ExpectationRequest{Path: "/path"},
		&ExpectationRequest{Path: "/path"}))
}

func TestExpectationsMatch_MethodsNotEqAndPathsAreEq_False(t *testing.T) {
	assert.False(t, expectationsMatch(
		&ExpectationRequest{Method: "GET", Path: "/path"},
		&ExpectationRequest{Method: "POST", Path: "/path"}))
}

func TestExpectationsMatch_HeadersAreEq_True(t *testing.T) {
	assert.True(t, expectationsMatch(
		&ExpectationRequest{Headers: Headers{"h1": "hv1"}},
		&ExpectationRequest{Headers: Headers{"h1": "hv1"}}))
}

func TestExpectationsMatch_HeaderNotEq_False(t *testing.T) {
	result := expectationsMatch(
		&ExpectationRequest{Headers: Headers{"h1": "hv1"}},
		&ExpectationRequest{Headers: Headers{"h2": "hv2"}})
	assert.False(t, result)
}

func TestExpectationsMatch_HeaderValueNotEq_False(t *testing.T) {
	assert.False(t, expectationsMatch(
		&ExpectationRequest{Headers: Headers{"h1": "hv1"}},
		&ExpectationRequest{Headers: Headers{"h1": "hv2"}}))
}

func TestExpectationsMatch_NoHeaderinReq_False(t *testing.T) {
	assert.False(t, expectationsMatch(
		&ExpectationRequest{},
		&ExpectationRequest{Headers: Headers{"h2": "hv2"}}))
}

func TestExpectationsMatch_NoHeaderInFilter_True(t *testing.T) {
	assert.True(t, expectationsMatch(
		&ExpectationRequest{Headers: Headers{"h1": "hv1"}},
		&ExpectationRequest{}))
}

func TestExpectationsMatch_BodysEq_True(t *testing.T) {
	assert.True(t, expectationsMatch(
		&ExpectationRequest{Body: "body"},
		&ExpectationRequest{Body: "body"}))
}

func httpNewRequestMust(method, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}

	return req
}

func TestGzFilter_ApplyForward_HostAndHeadersAreUpdated(t *testing.T) {
	req := httpNewRequestMust("GET", "/request", nil)
	req.Header.Add("h_req", "hv_req")

	exp := Expectation{
		Key: "k",
		Forward: &ExpectationForward{
			Scheme:  "https",
			Host:    "localhost_fwd",
			Headers: Headers{"Host": "fwd_host", "h_req": "hv_fwd", "h_fwd": "hv_fwd"},
		},
	}

	filter := NewMockedGzFilter()
	filter.Add(exp)

	// Act
	resp := filter.Apply(req)

	// Assert
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.HTTPCode)
	respBody := string(resp.Body)
	assert.Contains(t, respBody, "https://localhost_fwd/request")
	assert.Contains(t, respBody, "Host:fwd_host")
	assert.Contains(t, respBody, "H_req:hv_fwd")
	assert.Contains(t, respBody, "H_fwd:hv_fwd")
	assert.Equal(t, "hv_fwd", resp.Headers["H_req"])
	assert.Equal(t, "hv_fwd", resp.Headers["H_fwd"])
}
