package main

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	tmpl := `{ {{$s := (stringsSplit "a.b.c" ".")}}{{index $s 0}} }`

	expectedOutput := `{ a }`

	res := TemplateCreateResponseBody(tmpl, expReq)
	assert.Equal(t, expectedOutput, res)
}

func TestTemplateCreateResponseWithHeaders(t *testing.T) {
	expReq := &ExpectationRequest{
		Headers: &Headers{"Host": "api.staging.cheaptickets.be"}}

	tmpl := `{ {{$s := index requestHeaders "Host"}}{{$s}} }`

	expectedOutput := `{ api.staging.cheaptickets.be }`

	res := TemplateCreateResponseBody(tmpl, expReq)
	assert.Equal(t, expectedOutput, res)
}
