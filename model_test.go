package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpectationsFromString(t *testing.T) {
	str := "[{\"key\": \"k1\"},{\"key\": \"k2\"}]"
	exps := ExpectationsFromString(str)
	assert.Equal(t, 2, len(exps))
	assert.Equal(t, "k1", exps[0].Key)
	assert.Equal(t, "k2", exps[1].Key)
}

func TestExpectationsDefaultValues(t *testing.T) {
	str := "[{\"key\": \"k1\", \"forward\":{\"host\":\"localhost\"}}]"
	exps := ExpectationsFromString(str)
	assert.Equal(t, 1, len(exps))
	assert.Equal(t, "k1", exps[0].Key)
	assert.NotNil(t, exps[0].Forward)
	assert.Equal(t, "localhost", exps[0].Forward.Host)
	assert.Equal(t, "http", exps[0].Forward.Scheme)
}

func TestConvertationExpectationFromReadCloser(t *testing.T) {
	str := "{\"key\": \"k\"}"
	exp := Expectation{}
	err := ObjectFromJSON(ioutil.NopCloser(strings.NewReader(str)), &exp)
	assert.Nil(t, err)
	assert.Equal(t, "k", exp.Key)
}

func TestConvertationExpectationFromFile(t *testing.T) {
	str := "[{\"key\": \"k\"}]"
	file := "test.json"
	err := ioutil.WriteFile(file, []byte(str), 0644)
	assert.Nil(t, err)

	exps := ExpectationsFromJSONFile(file)
	assert.Nil(t, err)

	err = os.Remove(file)
	assert.Nil(t, err)

	assert.Len(t, exps, 1)
	assert.Equal(t, "k", exps[0].Key)
}
