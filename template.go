package main

import (
	"bytes"
	"encoding/base64"
	"regexp"
	"text/template"

	"github.com/rs/zerolog/log"
)

func (req *ExpectationRequest) regexLocator(filter string) interface{} {
	r := regexp.MustCompile("(?m)" + filter)

	res := r.FindAllStringSubmatch(req.Body, -1)
	resSlice := make([]string, 0, len(res))

	for _, match := range res {
		if len(match) > 1 {
			resSlice = append(resSlice, match[1])
		} else {
			resSlice = append(resSlice, match[0])
		}
	}
	return resSlice
}

// TemplateCreateResponseBody creates response body as string based on template and incoming request
func TemplateCreateResponseBody(tmpl string, req *ExpectationRequest) string {
	fLog := log.With().Str("function", "TemplateCreateResponseBody").Logger()

	decodedTmpl, err := TemplateDecodeFromBase64(tmpl)
	if err != nil {
		fLog.Warn().Err(err)
		decodedTmpl = tmpl
	}

	buf := new(bytes.Buffer)
	fmap := template.FuncMap{
		"regexLocator": req.regexLocator}
	t := template.Must(template.New("main").Funcs(fmap).Parse(decodedTmpl))
	err = t.Execute(buf, req.Body)
	if err != nil {
		fLog.Error().Err(err)
	}
	res := buf.String()
	return res
}

// TemplateDecodeFromBase64 decodes string from base64
func TemplateDecodeFromBase64(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return string(decoded), nil
	}
	return "", err
}
