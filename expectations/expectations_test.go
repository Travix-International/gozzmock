package expectations

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzStorage_EmptyGetOrdered(t *testing.T) {
	storage := NewGzStorage()

	// Act
	res := storage.GetOrdered()

	// Assert
	assert.Empty(t, res)
}

func TestGzStorage_Add(t *testing.T) {
	storage := NewGzStorage()

	// Act
	storage.Add(Expectation{Key: "key"})

	// Assert
	res := storage.GetOrdered()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "key", res[0].Key)
}

func TestGzStorage_AddSameKey(t *testing.T) {
	storage := NewGzStorage()

	// Act
	storage.Add(Expectation{Key: "v1", Priority: 1})
	storage.Add(Expectation{Key: "v1", Priority: 2})

	// Assert
	res := storage.GetOrdered()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "v1", res[0].Key)
	assert.Equal(t, 2, res[0].Priority)
}

func TestGzStorage_Remove(t *testing.T) {
	storage := NewGzStorage()
	storage.Add(Expectation{Key: "k"})

	// Act
	storage.Remove("k")

	// Assert
	res := storage.GetOrdered()
	assert.NotNil(t, res)
	assert.Empty(t, res)
}

func TestGzStorage_RemoveWrongKey(t *testing.T) {
	storage := NewGzStorage()
	storage.Add(Expectation{Key: "key"})

	// Act
	storage.Remove("kW")

	// Assert
	res := storage.GetOrdered()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "key", res[0].Key)
}

func TestGzStorage_Order(t *testing.T) {
	storage := NewGzStorage()
	storage.Add(Expectation{Key: "p10", Priority: 10})
	storage.Add(Expectation{Key: "p5", Priority: 5})
	storage.Add(Expectation{Key: "p15", Priority: 15})

	// Act
	res := storage.GetOrdered()

	// Assert
	assert.Equal(t, 3, len(res))
	assert.Equal(t, "p15", res[0].Key)
	assert.Equal(t, "p10", res[1].Key)
	assert.Equal(t, "p5", res[2].Key)
}

func TestGzStorage_AddFromJson_Ok(t *testing.T) {
	str := "[{\"key\": \"k\"}]"
	file := "test.json"
	err := ioutil.WriteFile(file, []byte(str), 0644)
	assert.Nil(t, err)

	storage := NewGzStorage()
	// Act
	storage.AddFromJSON(file)
	err = os.Remove(file)

	// Assert
	assert.Nil(t, err)

	res := storage.GetOrdered()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "k", res[0].Key)
}

func TestGzStorage_AddFromString_Ok(t *testing.T) {
	str := "[{\"key\": \"k1\", \"priority\":1},{\"key\": \"k2\", \"priority\":0}]"
	storage := NewGzStorage()

	// Act
	storage.AddFromString(str)

	// Assert
	exps := storage.GetOrdered()
	assert.Equal(t, 2, len(exps))
	assert.Equal(t, "k1", exps[0].Key)
	assert.Equal(t, "k2", exps[1].Key)
}

func TestGzStorage_AddFromString_DefaultValues(t *testing.T) {
	str := "[{\"key\": \"k1\", \"forward\":{\"host\":\"localhost\"}}]"
	storage := NewGzStorage()

	// Act
	storage.AddFromString(str)

	// Assert
	exps := storage.GetOrdered()
	assert.Equal(t, 1, len(exps))
	assert.Equal(t, "k1", exps[0].Key)
	assert.NotNil(t, exps[0].Forward)
	assert.Equal(t, "localhost", exps[0].Forward.Host)
	assert.Equal(t, "http", exps[0].Forward.Scheme)
}

func TestHttpRequestToExpectationRequest_SimpleRequest_AllFieldsTranslated(t *testing.T) {
	request, err := http.NewRequest("POST", "https://www.host.com/a/b?foo=bar#fr", strings.NewReader("body text"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("h1", "hv1")
	request.Header.Add("h1", "hv2")

	// Act
	exp, err := HttpRequestToExpectationRequest(request)

	// Assert
	assert.Nil(t, err)
	assert.NotNil(t, exp)
	assert.Equal(t, "POST", exp.Method)
	assert.Equal(t, "/a/b?foo=bar#fr", exp.Path)
	assert.Equal(t, "body text", string(exp.Body))
	assert.Equal(t, 1, len(exp.Headers))
	assert.Equal(t, "hv1,hv2", exp.Headers["H1"])
}
