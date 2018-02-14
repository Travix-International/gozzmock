package main

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsTemplateCreateResponseBodySimpleJsonEncodedBase64(t *testing.T) {
	// Arrange
	expReq := &ExpectationRequest{
		Body: `{
			"a": [
				{
					"b": "bv1"
				}
				]}`}

	tmpl := base64.StdEncoding.EncodeToString([]byte(`
		var response = {"response": JSON.parse(request.Body)["a"][0]["b"]};
		JSON.stringify(response);`))

	expectedOutput := `{"response":"bv1"}`

	// Act
	res, err := JsTemplateCreateResponseBody(tmpl, expReq)

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, expectedOutput, res)
}

func TestJsTemplateCreateResponseBodyWrongEncoding(t *testing.T) {
	// Arrange
	expReq := &ExpectationRequest{
		Body: `{
			"a": [
				{
					"b": "bv1"
				}
				]}`}

	tmpl := `"abc"`

	// Act
	res, err := JsTemplateCreateResponseBody(tmpl, expReq)

	// Assert
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Error decoding from base64 template")
	assert.Equal(t, "", res)
}
