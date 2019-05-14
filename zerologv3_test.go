package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestV3LogEmptyMessage(t *testing.T) {
	var outbuf bytes.Buffer
	v3logger := zerolog.New(V3FormatWriter{Out: &outbuf})

	// Act
	v3logger.Info().Msg("")

	// Assert
	actualMessage := outbuf.String()
	expectedMessage := `{"log_format":"v3","metadata":{"log_level":"INFO","messagetype":"","timestamp":""},"payload":{},"source":{"appname":""}}`
	expectedMessage += "\n"
	assert.Equal(t, expectedMessage, actualMessage)
}

func returnFixedTimestamp() time.Time {
	return time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
}

func TestV3LogAllV3Fields(t *testing.T) {
	var outbuf bytes.Buffer
	v3logger := zerolog.New(V3FormatWriter{Out: &outbuf}).
		With().
		Str("messagetype", "MesssageTypeTest").
		Str("appname", "AppNameTest").
		Timestamp().
		Logger()
	zerolog.TimestampFunc = returnFixedTimestamp

	// Act
	v3logger.Debug().Msg("Test")

	// Assert
	actualMessage := outbuf.String()
	expectedMessage := `{"log_format":"v3","metadata":{"log_level":"DEBUG","messagetype":"MesssageTypeTest","timestamp":"2009-11-17T20:34:58Z"},"payload":{"message":"Test"},"source":{"appname":"AppNameTest"}}`
	expectedMessage += "\n"
	assert.Equal(t, expectedMessage, actualMessage)
}

func TestV3LogCustomField(t *testing.T) {
	var outbuf bytes.Buffer
	v3logger := zerolog.New(V3FormatWriter{Out: &outbuf}).
		With().
		Str("custom_field", "CustomField1").
		Logger()
	zerolog.TimestampFunc = returnFixedTimestamp

	// Act
	v3logger.Debug().Msg("")

	// Assert
	actualMessage := outbuf.String()
	expectedMessage := `{"log_format":"v3","metadata":{"log_level":"DEBUG","messagetype":"","timestamp":""},"payload":{"custom_field":"CustomField1"},"source":{"appname":""}}`
	expectedMessage += "\n"
	assert.Equal(t, expectedMessage, actualMessage)
}
