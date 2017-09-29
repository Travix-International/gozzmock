package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestControllerCreateStorage_NoExpectations_ReturnsPointer(t *testing.T) {
	storage := ControllerCreateStorage()
	assert.Equal(t, &Storage{expectations: make(Expectations)}, storage)
}

func TestControllerGetExpectations_NoExpectations_ReturnEmptyList(t *testing.T) {
	storage := ControllerCreateStorage()

	exps, err := storage.GetExpectationsJSON()
	assert.Nil(t, err)
	assert.Equal(t, "{}", string(exps))
}

func TestControllerGetExpectationsAsStructure_NoExpectations_ReturnEmptyStructure(t *testing.T) {
	storage := ControllerCreateStorage()

	exps := storage.GetExpectationsStructure()
	assert.Equal(t, Expectations{}, exps)
}

func TestControllerAddExpectations_NoExpectations_ReturnOneItem(t *testing.T) {
	var exp = Expectation{Key: "k"}

	storage := ControllerCreateStorage()

	storage.AddExpectation(exp.Key, exp)

	exps := storage.GetExpectationsStructure()
	assert.Contains(t, exps, exp.Key)
	assert.Equal(t, exp, exps[exp.Key])
}

func TestControllerAddExpectations_ExistingKey_ReturnUpdatedOneItem(t *testing.T) {
	var exp1 = Expectation{Key: "k", Delay: 1}
	var exp2 = Expectation{Key: "k", Delay: 2}

	storage := ControllerCreateStorage()

	storage.AddExpectation(exp1.Key, exp1)
	exps := storage.GetExpectationsStructure()
	assert.Contains(t, exps, exp1.Key)

	storage.AddExpectation(exp2.Key, exp2)
	exps = storage.GetExpectationsStructure()
	assert.Contains(t, exps, exp2.Key)
	assert.Equal(t, 1, len(exps))
	assert.Equal(t, exp2, exps[exp2.Key])
}

func TestControllerAddExpectations_NewKey_ReturnTwoItems(t *testing.T) {
	var exp1 = Expectation{Key: "k1", Delay: 1}
	var exp2 = Expectation{Key: "k2", Delay: 2}

	storage := ControllerCreateStorage()

	storage.AddExpectation(exp1.Key, exp1)
	storage.AddExpectation(exp2.Key, exp2)

	exps := storage.GetExpectationsStructure()
	assert.Contains(t, exps, exp2.Key)

	assert.Equal(t, 2, len(exps))
	assert.Equal(t, exp1, exps[exp1.Key])
	assert.Equal(t, exp2, exps[exp2.Key])
}

func TestControllerRemoveExpectations_OneExpectations_ReturnEmptyList(t *testing.T) {
	var exp = Expectation{Key: "k"}

	storage := ControllerCreateStorage()

	storage.AddExpectation(exp.Key, exp)
	storage.RemoveExpectation(exp.Key)

	exps := storage.GetExpectationsStructure()
	assert.Empty(t, exps)
}

func TestControllerRemoveWrongKeyExpectations_OneExpectations_NotReturnError(t *testing.T) {
	var exp = Expectation{Key: "k"}

	storage := ControllerCreateStorage()

	storage.AddExpectation(exp.Key, exp)
	storage.RemoveExpectation("wrong_key")

	exps := storage.GetExpectationsStructure()
	assert.Contains(t, exps, exp.Key)
}

func TestControllerTranslateRequestToExpectation_SimpleRequest_AllFieldsTranslated(t *testing.T) {
	request, err := http.NewRequest("POST", "https://www.host.com/a/b?foo=bar#fr", strings.NewReader("body text"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("h1", "hv1")

	exp := ControllerTranslateRequestToExpectation(request)

	assert.NotNil(t, exp)
	assert.Equal(t, "POST", exp.Method)
	assert.Equal(t, "/a/b?foo=bar#fr", exp.Path)
	assert.Equal(t, "body text", exp.Body)
	assert.NotNil(t, exp.Headers)
	assert.Equal(t, 1, len(*exp.Headers))
	assert.Equal(t, "hv1", (*exp.Headers)["H1"])
}

func TestControllerTranslateHTTPHeadersToExpHeaders_TwoHeaders_HeadersTranslated(t *testing.T) {
	header := http.Header{}
	header.Add("h1", "hv1")
	header.Add("h1", "hv2")

	expHeaders := ControllerTranslateHTTPHeadersToExpHeaders(header)
	assert.NotNil(t, expHeaders)
	assert.Equal(t, 1, len(*expHeaders))
	assert.Equal(t, "hv1,hv2", (*expHeaders)["H1"])
}

func TestControllerStringPassesFilter_EmptyFilter_True(t *testing.T) {
	assert.True(t, ControllerStringPassesFilter("abc", ""))
}

func TestControllerStringPassesFilter_ExistingSubstring_True(t *testing.T) {
	assert.True(t, ControllerStringPassesFilter("abc", "ab"))
}

func TestControllerStringPassesFilter_ExistingRegex_True(t *testing.T) {
	assert.True(t, ControllerStringPassesFilter("abc", ".b."))
}

func TestControllerStringPassesFilter_NotExistingSubstring_False(t *testing.T) {
	assert.False(t, ControllerStringPassesFilter("abc", "zz"))
}

func TestControllerStringPassesFilter_NotExistingRegex_False(t *testing.T) {
	assert.False(t, ControllerStringPassesFilter("abc", ".z."))
}

func TestControllerStringPassesFilter_MultilineBody_True(t *testing.T) {
	assert.True(t, ControllerStringPassesFilter("a\nb", "a.b"))
}

func TestControllerRequestPassFilter_EmptyRequestEmptyFilter_True(t *testing.T) {
	assert.True(t, ControllerRequestPassesFilter(
		&ExpectationRequest{},
		&ExpectationRequest{}))
}

func TestControllerRequestPassFilter_MethodsAreEq_True(t *testing.T) {
	assert.True(t, ControllerRequestPassesFilter(
		&ExpectationRequest{Method: "POST"},
		&ExpectationRequest{Method: "POST"}))
}

func TestControllerRequestPassFilter_PathsAreEq_True(t *testing.T) {
	assert.True(t, ControllerRequestPassesFilter(
		&ExpectationRequest{Path: "/path"},
		&ExpectationRequest{Path: "/path"}))
}

func TestControllerRequestPassFilter_MethodsNotEqAndPathsAreEq_False(t *testing.T) {
	assert.False(t, ControllerRequestPassesFilter(
		&ExpectationRequest{Method: "GET", Path: "/path"},
		&ExpectationRequest{Method: "POST", Path: "/path"}))
}

func TestControllerRequestPassFilter_HeadersAreEq_True(t *testing.T) {
	assert.True(t, ControllerRequestPassesFilter(
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}},
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}}))
}

func TestControllerRequestPassFilter_HeaderNotEq_False(t *testing.T) {
	result := ControllerRequestPassesFilter(
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}},
		&ExpectationRequest{Headers: &Headers{"h2": "hv2"}})
	assert.False(t, result)
}

func TestControllerRequestPassFilter_HeaderValueNotEq_False(t *testing.T) {
	assert.False(t, ControllerRequestPassesFilter(
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}},
		&ExpectationRequest{Headers: &Headers{"h1": "hv2"}}))
}

func TestControllerRequestPassFilter_NoHeaderinReq_False(t *testing.T) {
	assert.False(t, ControllerRequestPassesFilter(
		&ExpectationRequest{},
		&ExpectationRequest{Headers: &Headers{"h2": "hv2"}}))
}

func TestControllerRequestPassFilter_NoHeaderInFilter_True(t *testing.T) {
	assert.True(t, ControllerRequestPassesFilter(
		&ExpectationRequest{Headers: &Headers{"h1": "hv1"}},
		&ExpectationRequest{}))
}

func TestControllerRequestPassFilter_BodysEq_True(t *testing.T) {
	assert.True(t, ControllerRequestPassesFilter(
		&ExpectationRequest{Body: "body"},
		&ExpectationRequest{Body: "body"}))
}

func TestControllerSortExpectationsByPriority_EmptyExps_OK(t *testing.T) {
	storage := ControllerCreateStorage()

	sortedMap := storage.GetExpectationsOrderedByPriority()
	assert.Equal(t, 0, len(sortedMap))
}

func TestControllerSortExpectationsByPriority_ListOfExpectations_OK(t *testing.T) {
	exp1 := Expectation{Key: "k1", Priority: 1}
	exp2 := Expectation{Key: "k0", Priority: 0}
	exp3 := Expectation{Key: "k2", Priority: 2}

	storage := ControllerCreateStorage()
	storage.AddExpectation(exp1.Key, exp1)
	storage.AddExpectation(exp2.Key, exp2)
	storage.AddExpectation(exp3.Key, exp3)

	sortedMap := storage.GetExpectationsOrderedByPriority()

	assert.Equal(t, 3, len(sortedMap))
	assert.Equal(t, "k2", sortedMap[0].Key)
	assert.Equal(t, "k1", sortedMap[1].Key)
	assert.Equal(t, "k0", sortedMap[2].Key)
}

func TestControllerControllerCreateHTTPRequestWithHeaders(t *testing.T) {
	expReq := &ExpectationRequest{Method: "GET", Path: "/request", Headers: &Headers{"h_req": "hv_req"}}
	expFwd := &ExpectationForward{Scheme: "https", Host: "localhost_fwd", Headers: &Headers{"h_req": "hv_fwd", "h_fwd": "hv_fwd"}}
	httpReq := ControllerCreateHTTPRequest(expReq, expFwd)
	assert.NotNil(t, httpReq)
	assert.Equal(t, expReq.Method, httpReq.Method)
	assert.Equal(t, expFwd.Host, httpReq.Host)
	assert.Equal(t, fmt.Sprintf("%s://%s%s", expFwd.Scheme, expFwd.Host, expReq.Path), httpReq.URL.String())
	assert.Equal(t, "hv_fwd", httpReq.Header.Get("h_req"))
	assert.Equal(t, "hv_fwd", httpReq.Header.Get("h_fwd"))
}

func TestControllerControllerCreateHTTPRequestHostRewrite(t *testing.T) {
	expReq := &ExpectationRequest{Method: "GET", Path: "/request"}
	expFwd := &ExpectationForward{Scheme: "https", Host: "localhost_fwd", Headers: &Headers{"Host": "fwd_host"}}
	httpReq := ControllerCreateHTTPRequest(expReq, expFwd)
	assert.NotNil(t, httpReq)
	assert.Equal(t, "fwd_host", httpReq.Host)
	assert.Equal(t, fmt.Sprintf("%s://%s%s", expFwd.Scheme, expFwd.Host, expReq.Path), httpReq.URL.String())
}
