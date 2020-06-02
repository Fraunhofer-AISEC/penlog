// SPDX-License-Identifier: Apache-2.0

// Package penlog implements the penlog(7) specification.
package penlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
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
	lines     int
}

const (
	msgTypeRead     = "read"
	msgTypeWrite    = "write"
	msgTypeMessage  = "message"
	msgTypePreamble = "preamble"
)

// RFC5424 Section 6.2.1
const (
	PrioEmergency = iota
	PrioAlert
	PrioCritical
	PrioError
	PrioWarning
	PrioNotice
	PrioInfo
	PrioDebug
)

func NewLogger(component string, w io.Writer) *Logger {
	lines := 0
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
	if rawVal, ok := os.LookupEnv("PENLOG_LINES"); ok {
		if val, err := strconv.ParseBool(rawVal); val && err == nil {
			lines = 3
		}
	}

	return &Logger{
		host:      hostname,
		component: component,
		timespec:  "2006-01-02T15:04:05.000000",
		lines:     lines,
		writer:    w,
	}
}

func (l *Logger) output(msg map[string]interface{}) error {
	msg["timestamp"] = time.Now().Format(l.timespec)
	msg["component"] = l.component
	msg["host"] = l.host
	if l.lines > 0 {
		_, file, line, ok := runtime.Caller(l.lines)
		if !ok {
			file = "???"
			line = 0
		}
		msg["line"] = fmt.Sprintf("%s:%d", file, line)
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}

	l.buf.Write(b)
	l.buf.WriteString("\n")
	l.buf.WriteTo(l.writer)

	return nil
}

func (l *Logger) EnableLines(calldepth int) {
	l.mu.Lock()
	l.lines = calldepth
	l.mu.Unlock()
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
		"data":     fmt.Sprint(v...),
		"type":     msgTypePreamble,
		"priority": PrioNotice,
	}

	l.Log(msg)
}

func (l *Logger) LogPreamblef(format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgTypePreamble,
		"priority": PrioNotice,
	}

	l.Log(msg)
}

func (l *Logger) LogRead(handle string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgTypeRead,
		"priority": PrioDebug,
		"handle":   handle,
	}

	l.Log(msg)
}

func (l *Logger) LogReadf(handle, format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgTypeRead,
		"priority": PrioDebug,
		"handle":   handle,
	}

	l.Log(msg)
}

func (l *Logger) LogWrite(handle string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgTypeWrite,
		"priority": PrioDebug,
		"handle":   handle,
	}

	l.Log(msg)
}

func (l *Logger) LogWritef(handle, format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgTypeWrite,
		"priority": PrioDebug,
		"handle":   handle,
	}

	l.Log(msg)
}

func (l *Logger) LogMessage(v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgTypeMessage,
		"priority": PrioInfo,
	}

	l.Log(msg)
}

func (l *Logger) LogMessagef(format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgTypeMessage,
		"priority": PrioInfo,
	}

	l.Log(msg)
}

func (l *Logger) LogError(v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgTypeMessage,
		"priority": PrioError,
	}

	l.Log(msg)
}

func (l *Logger) LogErrorf(format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgTypeMessage,
		"priority": PrioError,
	}

	l.Log(msg)
}

func (l *Logger) LogWarning(v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgTypeMessage,
		"priority": PrioError,
	}

	l.Log(msg)
}

func (l *Logger) LogWarningf(format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgTypeMessage,
		"priority": PrioWarning,
	}

	l.Log(msg)
}

func (l *Logger) LogDebug(v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgTypeMessage,
		"priority": PrioDebug,
	}

	l.Log(msg)
}

func (l *Logger) LogDebugf(format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgTypeMessage,
		"priority": PrioDebug,
	}

	l.Log(msg)
}
