package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestV3LogFormatMessage(t *testing.T) {
	// Arrange
	var outbuf bytes.Buffer
	v3logger := zerolog.New(V3FormatWriter{Out: &outbuf})

	// Act
	v3logger.Info().Msg("")

	// Assert
	actualMessage := outbuf.String()
	expectedMessage := `{"log_format":"v3","metadata":{"log_level":"INFO","message_type":"","timestamp":""},"payload":{"message":""},"source":{"app_name":""}}`
	expectedMessage += "\n"
	assert.Equal(t, expectedMessage, actualMessage)
}

func returnFixedTimestamp() time.Time {
	return time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
}

func TestV3LogFormatAllFields(t *testing.T) {
	// Arrange
	var outbuf bytes.Buffer
	v3logger := zerolog.New(V3FormatWriter{Out: &outbuf}).
		With().
		Str("message_type", "MesssageTypeTest").
		Str("app_name", "AppNameTest").
		Timestamp().
		Logger()
	zerolog.TimestampFunc = returnFixedTimestamp

	// Act
	v3logger.Debug().Msg("Test")

	// Assert
	actualMessage := outbuf.String()
	expectedMessage := `{"log_format":"v3","metadata":{"log_level":"DEBUG","message_type":"MesssageTypeTest","timestamp":"2009-11-17T20:34:58Z"},"payload":{"message":"Test"},"source":{"app_name":"AppNameTest"}}`
	expectedMessage += "\n"
	assert.Equal(t, expectedMessage, actualMessage)
}
