package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
func TestTemplateCreateResponseBodyFromTemplateSimpleTest(t *testing.T) {
	tmpl := `{
		{{$idegex := "\"id\"\\s*:\\s*(\\d+)" -}}
		"item_ids": "
		{{- range $index, $item := regexLocator $idegex }}
			{{- if $index}},{{end}}
			{{- $item }}
			{{- end}}",

		{{- $item_names_regex := "\"name\"\\s*:\\s*\"(\\w+)" }}
		{{- $item_names := regexLocator $item_names_regex }}
		"item_names": [
		{{- range $index, $item := $item_names }}
				{{- if $index}},{{end}}
			"{{$item}}"{{end}}
		]
	}`

	expReq := &ExpectationRequest{
		Body: `{
			"items": [
				{
					"id":0,
					"name": "item0",
				},
				{
					"id" : 1,
					"name": "item1",
				}
			]
		}`}

	expectedOutput := `{
		"item_ids": "0,1",
		"item_names": [
			"item0",
			"item1"
		]
	}`

	res := TemplateCreateResponseBody(tmpl, expReq)
	assert.Equal(t, expectedOutput, res)
}

func TestTemplateCreateResponseBodyFromBase64EncodedTemplate(t *testing.T) {
	tmpl := `{
		{{$idegex := "\"id\"\\s*:\\s*(\\d+)" -}}
		"item_ids": "
		{{- range $index, $item := regexLocator $idegex }}
			{{- if $index}},{{end}}
			{{- $item }}
			{{- end}}",

		{{- $item_names_regex := "\"name\"\\s*:\\s*\"(\\w+)" }}
		{{- $item_names := regexLocator $item_names_regex }}
		"item_names": [
		{{- range $index, $item := $item_names }}
				{{- if $index}},{{end}}
			"{{$item}}"{{end}}
		]
	}`
	tmplEncoded := base64.StdEncoding.EncodeToString([]byte(tmpl))

	expReq := &ExpectationRequest{
		Body: `{
			"items": [
				{
					"id":0,
					"name": "item0",
				},
				{
					"id" : 1,
					"name": "item1",
				}
			]
		}`}

	expectedOutput := `{
		"item_ids": "0,1",
		"item_names": [
			"item0",
			"item1"
		]
	}`

	res := TemplateCreateResponseBody(tmplEncoded, expReq)
	assert.Equal(t, expectedOutput, res)
}

func TestTemplateCreateResponseStringSplit(t *testing.T) {
	expReq := &ExpectationRequest{}

	tmpl := `{ {{index (stringsSplit "a.b.c" ".") 0}} }`

	expectedOutput := `{ a }`

	res := TemplateCreateResponseBody(tmpl, expReq)
	assert.Equal(t, expectedOutput, res)
}

func TestTemplateCreateResponseWithHeaders(t *testing.T) {
	expReq := &ExpectationRequest{
		Headers: &Headers{"Host": "api.staging.travix.com"}}

	tmpl := `{ {{index requestHeaders "Host"}} }`

	expectedOutput := `{ api.staging.travix.com }`

	res := TemplateCreateResponseBody(tmpl, expReq)
	assert.Equal(t, expectedOutput, res)
}

func TestTemplateCreateResponseWithAdd(t *testing.T) {
	expReq := &ExpectationRequest{}

	tmpl := `{ {{add 1 1}}, {{add 1 -1}}, {{add 0 0}} }`

	expectedOutput := `{ 2, 0, 0 }`

	res := TemplateCreateResponseBody(tmpl, expReq)
	assert.Equal(t, expectedOutput, res)
}

func TestTemplateCreateResponseJsonPath(t *testing.T) {
	expReq := &ExpectationRequest{
		Body: `{
			"a": [
				{
					"b": "bv1"
				}
				]}`}

	tmpl := `{{- $value := jsonLocator .Body "$.a[0].b" -}}
	{ "jsonpath": "{{ $value }}" }`

	expectedOutput := `{ "jsonpath": "bv1" }`

	res := TemplateCreateResponseBody(tmpl, expReq)

	assert.Equal(t, expectedOutput, res)
}

func TestTemplateCreateResponseJsonPathToObject(t *testing.T) {
	expReq := &ExpectationRequest{
		Body: `{
			"a": [
				{
					"b": "bv1"
				}
				]}`}

	tmpl := `{{- $value := jsonLocator .Body "$.a" -}}
	{ "jsonpath": "{{ $value }}" }`

	expectedOutput := `{ "jsonpath": "bv1" }`

	res := TemplateCreateResponseBody(tmpl, expReq)

	assert.Equal(t, expectedOutput, res)
}
*/
func TestJS(t *testing.T) {
	expReq := &ExpectationRequest{
		Body: `{
			"a": [
				{
					"b": "bv1"
				}
				]}`}

	//tmpl := `JSON.stringify(JSON.parse(req.Body)["a"])`
	tmpl := `JSON.parse(req.Body)["a"][0]["b"]`

	expectedOutput := `{ bv1 }`

	res := TemplateCreateResponseBody(tmpl, expReq)

	assert.Equal(t, expectedOutput, res)
}
