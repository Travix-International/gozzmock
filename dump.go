package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/rs/zerolog/log"
)

// dumpRequest dumps http request and writes content to log
func dumpRequest(req *http.Request) {
	fLog := log.With().Str("message_type", "dumpRequest").Logger()
	reqDumped, err := httputil.DumpRequest(req, true)
	if err != nil {
		fLog.Panic().Err(err)
		return
	}
	fLog.Debug().Str("messagetype", "Request").Msg(string(reqDumped))
}

// drainBody is copied from dump.go
// drainBody reads all of b to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
//
// It returns an error if the initial slurp of all bytes fails. It does not attempt
// to make the returned ReadClosers have identical error-matching behavior.
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

// dumpCompressedResponse is the same as DumpRequest from httputil
// dump.go but for gzipped body
//
// DumpRequest returns the given request in its HTTP/1.x wire
// representation. It should only be used by servers to debug client
// requests. The returned representation is an approximation only;
// some details of the initial request are lost while parsing it into
// an http.Request. In particular, the order and case of header field
// names are lost. The order of values in multi-valued headers is kept
// intact. HTTP/2 requests are dumped in HTTP/1.x form, not in their
// original binary representations.
//
// If body is true, DumpRequest also returns the body. To do so, it
// consumes req.Body and then replaces it with a new io.ReadCloser
// that yields the same bytes. If DumpRequest returns an error,
// the state of req is undefined.
//
// The documentation for http.Request.Write details which fields
// of req are included in the dump.
func dumpCompressedResponse(resp *http.Response, body bool) ([]byte, error) {
	var b bytes.Buffer
	var err error
	// emptyBody is an instance of empty reader.
	var emptyBody = ioutil.NopCloser(strings.NewReader(""))
	// errNoBody is a sentinel error value used by failureToReadBody so we
	// can detect that the lack of body was intentional.
	var errNoBody = errors.New("sentinel error value")
	save := resp.Body
	savecl := resp.ContentLength

	if !body {
		resp.Body = emptyBody
	} else if resp.Body == nil {
		resp.Body = emptyBody
	} else {
		save, resp.Body, err = drainBody(resp.Body)
		if err != nil {
			return nil, err
		}

		var reader *gzip.Reader
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body = ioutil.NopCloser(reader)
	}
	err = resp.Write(&b)
	if err == errNoBody {
		err = nil
	}
	resp.Body = save
	resp.ContentLength = savecl
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// dumpResponse dumps http response and writes content to log
func dumpResponse(resp *http.Response) {
	fLog := log.With().Str("message_type", "dumpResponse").Logger()
	var respDumped []byte
	var err error
	if resp.Header.Get("Content-Encoding") == "gzip" {
		respDumped, err = dumpCompressedResponse(resp, true)
	} else {
		respDumped, err = httputil.DumpResponse(resp, true)
	}
	if err != nil {
		fLog.Panic().Err(err)
		return
	}

	fLog.Debug().Str("messagetype", "DumpedResponse").Msg(string(respDumped))
}
