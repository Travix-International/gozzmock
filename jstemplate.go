package main

import (
	"encoding/base64"
	"fmt"

	"github.com/robertkrimen/otto"
)

// JsTemplateCreateResponseBody creates response body as string based on template and incoming request
func JsTemplateCreateResponseBody(tmpl string, req *ExpectationRequest) (string, error) {
	decodedTmpl, err := templateDecodeFromBase64(tmpl)
	if err != nil {
		return "", fmt.Errorf("Error decoding from base64 template %s \n %s", decodedTmpl, err.Error())
	}

	vm := otto.New()
	vm.Set("request", req)
	value, err := vm.Run(decodedTmpl)
	if err != nil {
		return "", fmt.Errorf("Error running template %s \n %s", decodedTmpl, err.Error())
	}

	return value.String(), nil
}

// templateDecodeFromBase64 decodes string from base64
func templateDecodeFromBase64(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return string(decoded), nil
	}
	return "", err
}
