package main

import (
	"bytes"
	"encoding/base64"
	"regexp"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
	"github.com/satori/go.uuid"
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

func uuidv4() string {
	return uuid.NewV4().String()
}

func (req *ExpectationRequest) requestHeaders() interface{} {
	return (*req.Headers)
}

// TemplateCreateResponseBody creates response body as string based on template and incoming request
func TemplateCreateResponseBody(tmpl string, req *ExpectationRequest) string {
	fLog := log.With().Str("function", "TemplateCreateResponseBody").Logger()

	decodedTmpl, err := templateDecodeFromBase64(tmpl)
	if err != nil {
		fLog.Warn().Err(err)
		decodedTmpl = tmpl
	}

	buf := new(bytes.Buffer)
	fmap := template.FuncMap{
		"regexLocator":   req.regexLocator,
		"requestHeaders": req.requestHeaders,
		"uuidv4":         uuidv4,
		"stringsSplit":   strings.Split}
	t := template.Must(template.New("main").Funcs(fmap).Parse(decodedTmpl))
	err = t.Execute(buf, req)
	if err != nil {
		fLog.Error().Err(err).Msgf("Error parsing template %s", decodedTmpl)
	}
	res := buf.String()
	return res
}

// templateDecodeFromBase64 decodes string from base64
func templateDecodeFromBase64(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return string(decoded), nil
	}
	return "", err
}
