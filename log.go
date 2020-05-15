// SPDX-License-Identifier: Apache-2.0

// Package penlog implements the penlog(7) specification.
package penlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Logger struct {
	host      string
	component string
	timespec  string
	writer    io.Writer
	buf       bytes.Buffer
	mu        sync.Mutex
}

const (
	msgTypeError    = "error"
	msgTypeWarning  = "warning"
	msgTypeInfo     = "info"
	msgTypeDebug    = "debug"
	msgTypeRead     = "read"
	msgTypeWrite    = "write"
	msgTypeMessage  = "msg"
	msgTypePreamble = "preamble"
)

func NewLogger(component string, w io.Writer) *Logger {
	hostname, err := os.Hostname()
	// This should not happen!
	if err != nil {
		panic(err)
	}

	if component == "" {
		if val, ok := os.LookupEnv("PENLOG_COMPONENT"); ok {
			component = val
		} else {
			component = "root"
		}
	}

	return &Logger{
		host:      hostname,
		component: component,
		timespec:  "2006-01-02T15:04:05.000000",
		writer:    w,
	}
}

func (l *Logger) output(msg map[string]interface{}) error {
	msg["timestamp"] = time.Now().Format(l.timespec)
	msg["component"] = l.component
	msg["host"] = l.host

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}

	l.buf.Write(b)
	l.buf.WriteString("\n")
	l.buf.WriteTo(l.writer)

	return nil
}

func (l *Logger) Log(msg map[string]interface{}) {
	l.mu.Lock()
	if err := l.output(msg); err != nil {
		// This is clearly a bug!
		panic(err)
	}
	l.mu.Unlock()
}

func (l *Logger) LogPreamble(v ...interface{}) {
	var msg = map[string]interface{}{
		"data": fmt.Sprint(v...),
		"type": msgTypePreamble,
	}

	l.Log(msg)
}

func (l *Logger) LogPreamblef(format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data": fmt.Sprintf(format, v...),
		"type": msgTypePreamble,
	}

	l.Log(msg)
}

func (l *Logger) LogRead(handle string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":   fmt.Sprint(v...),
		"type":   msgTypeRead,
		"handle": handle,
	}

	l.Log(msg)
}

func (l *Logger) LogReadf(handle, format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":   fmt.Sprintf(format, v...),
		"type":   msgTypeRead,
		"handle": handle,
	}

	l.Log(msg)
}

func (l *Logger) LogWrite(handle string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":   fmt.Sprint(v...),
		"type":   msgTypeWarning,
		"handle": handle,
	}

	l.Log(msg)
}

func (l *Logger) LogWritef(handle, format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":   fmt.Sprintf(format, v...),
		"type":   msgTypeWrite,
		"handle": handle,
	}

	l.Log(msg)
}

func (l *Logger) LogMessage(v ...interface{}) {
	var msg = map[string]interface{}{
		"data": fmt.Sprint(v...),
		"type": msgTypeMessage,
	}

	l.Log(msg)
}

func (l *Logger) LogMessagef(format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data": fmt.Sprintf(format, v...),
		"type": msgTypeMessage,
	}

	l.Log(msg)
}
