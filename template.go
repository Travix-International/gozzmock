package main

import (
	"bytes"
	"regexp"
	"text/template"
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
	buf := new(bytes.Buffer)
	fmap := template.FuncMap{
		"regexLocator": req.regexLocator}
	t := template.Must(template.New("main").Funcs(fmap).Parse(tmpl))
	err := t.Execute(buf, req.Body)
	if err != nil {
		panic(err)
	}
	res := buf.String()
	return res
}
