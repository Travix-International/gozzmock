package main

import (
	"encoding/base64"

	"github.com/robertkrimen/otto"

	"github.com/rs/zerolog/log"
)

// JsTemplateCreateResponseBody creates response body as string based on template and incoming request
func JsTemplateCreateResponseBody(tmpl string, req *ExpectationRequest) string {
	fLog := log.With().Str("function", "JsTemplateCreateResponseBody").Logger()

	decodedTmpl, err := templateDecodeFromBase64(tmpl)
	if err != nil {
		fLog.Warn().Err(err)
		decodedTmpl = tmpl
	}

	vm := otto.New()
	vm.Set("req", req)
	value, err := vm.Run(decodedTmpl)
	if err != nil {
		fLog.Error().Err(err).Msgf("Error running template %s", decodedTmpl)
	}
	return value.String()
}

// templateDecodeFromBase64 decodes string from base64
func templateDecodeFromBase64(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return string(decoded), nil
	}
	return "", err
}
