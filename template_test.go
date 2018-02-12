package main

import (
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
