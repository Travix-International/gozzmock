package zerologv3

/*
Package zerologv3 provides a writer for log format v3 which is used in Travix
Implementation is based on zerolog.ConsoleWriter

Example of log message in v3 format:

{
    "log_format": "v3",
    "metadata": {
        "log_level": "DEBUG",
        "message_type": "function_name",
        "timestamp": "2006-01-02T15:04:05Z07:00"
    },
    "payload": {
        "message": "message_text"
    },
    "source": {
        "app_name": "gozzmock"
    }
}
*/

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog"
)

type metadata struct {
	LogLevel    string `json:"log_level"`
	MessageType string `json:"message_type"`
	Timestamp   string `json:"timestamp"`
}

type payload struct {
	Message string `json:"message"`
}

type source struct {
	AppName string `json:"app_name"`
}

type v3LogMessage struct {
	LogFormat string   `json:"log_format"`
	Metadata  metadata `json:"metadata"`
	Payload   payload  `json:"payload"`
	Source    source   `json:"source"`
}

// V3FormatWriter is a writer for log messages in v3 format
type V3FormatWriter struct {
	Out io.Writer
}

func (w V3FormatWriter) Write(p []byte) (n int, err error) {
	var event map[string]interface{}
	err = json.Unmarshal(p, &event)
	if err != nil {
		fmt.Println("Error unmarshalling events!")
		return
	}

	loglevel, ok := event[zerolog.LevelFieldName].(string)
	if !ok {
		loglevel = ""
	}

	message, ok := event[zerolog.MessageFieldName].(string)
	if !ok {
		message = ""
	}

	timestamp, ok := event[zerolog.TimestampFieldName].(string)
	if !ok {
		timestamp = ""
	}

	messagetype, ok := event["message_type"].(string)
	if !ok {
		messagetype = ""
	}

	appname, ok := event["app_name"].(string)
	if !ok {
		appname = ""
	}

	logMessage := v3LogMessage{
		LogFormat: "v3",
		Metadata: metadata{
			LogLevel:    strings.ToUpper(loglevel),
			MessageType: messagetype,
			Timestamp:   timestamp},
		Payload: payload{
			Message: message},
		Source: source{
			AppName: appname}}

	msg, err := json.Marshal(logMessage)
	if err != nil {
		fmt.Println("Error marshalling log message!")
		return
	}
	fmt.Fprintln(w.Out, string(msg))

	n = len(msg)
	return
}
