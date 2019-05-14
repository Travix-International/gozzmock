package main

/*
Provides a writer for log format v3 which is used in Travix
Implementation is based on zerolog.ConsoleWriter

Example of log message in v3 format:

{
    "log_format": "v3",
    "metadata": {
        "log_level": "DEBUG",
        "messagetype": "function_name",
        "timestamp": "2006-01-02T15:04:05Z07:00"
    },
    "payload": {
        "message": "message_text"
    },
    "source": {
        "appname": "gozzmock"
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
	MessageType string `json:"messagetype"`
	Timestamp   string `json:"timestamp"`
}

type payload map[string]string

type source struct {
	AppName string `json:"appname"`
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

	logMessage := v3LogMessage{
		LogFormat: "v3",
		Metadata: metadata{
			LogLevel:    "",
			MessageType: "",
			Timestamp:   ""},
		Payload: payload{},
		Source: source{
			AppName: ""}}

	for field, value := range event {
		switch field {
		case zerolog.LevelFieldName:
			logMessage.Metadata.LogLevel = strings.ToUpper(value.(string))
		case zerolog.TimestampFieldName:
			logMessage.Metadata.Timestamp = value.(string)
		case "messagetype":
			logMessage.Metadata.MessageType = value.(string)
		case "appname":
			logMessage.Source.AppName = value.(string)
		case zerolog.MessageFieldName:
			logMessage.Payload["message"] = value.(string)
		default:
			strValue, ok := value.(string)
			if !ok {
				logMessage.Payload[field] = ""
			}
			logMessage.Payload[field] = strValue
		}
	}

	msg, err := json.Marshal(logMessage)
	if err != nil {
		fmt.Println("Error marshalling log message!")
		return 0, err
	}
	fmt.Fprintln(w.Out, string(msg))
	return len(msg), nil
}
