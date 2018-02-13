package main

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsTemplateCreateResponseBodySimpleJson(t *testing.T) {
	expReq := &ExpectationRequest{
		Body: `{
			"a": [
				{
					"b": "bv1"
				}
				]}`}

	tmpl := `var response = {"response": JSON.parse(request.Body)["a"][0]["b"]};
	JSON.stringify(response)`

	expectedOutput := `{"response":"bv1"}`

	res := JsTemplateCreateResponseBody(tmpl, expReq)

	assert.Equal(t, expectedOutput, res)
}

func TestJsTemplateCreateResponseBodySimpleJsonEncodedBase64(t *testing.T) {
	expReq := &ExpectationRequest{
		Body: `{
			"a": [
				{
					"b": "bv1"
				}
				]}`}

	tmpl := base64.StdEncoding.EncodeToString([]byte(`
		var response = {"response": JSON.parse(request.Body)["a"][0]["b"]};
		JSON.stringify(response)`))

	expectedOutput := `{"response":"bv1"}`

	res := JsTemplateCreateResponseBody(tmpl, expReq)

	assert.Equal(t, expectedOutput, res)
}
