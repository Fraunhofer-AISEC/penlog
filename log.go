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
	lines     bool
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

func getLineNumber(depth int) string {
	if _, file, line, ok := runtime.Caller(depth); ok {
		return fmt.Sprintf("%s:%d", file, line)
	}
	return ""
}

func NewLogger(component string, w io.Writer) *Logger {
	lines := false
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
			lines = true
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

func (l *Logger) EnableLines(enable bool) {
	l.mu.Lock()
	l.lines = enable
	l.mu.Unlock()
}

func (l *Logger) output(msg map[string]interface{}, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg["timestamp"] = time.Now().Format(l.timespec)
	msg["component"] = l.component
	msg["host"] = l.host
	if l.lines {
		msg["line"] = getLineNumber(depth)
	}

	b, err := json.Marshal(msg)
	if err != nil {
		// This is clearly a bug!
		panic(err)
	}

	l.buf.Write(b)
	l.buf.WriteString("\n")
	l.buf.WriteTo(l.writer)
}

func (l *Logger) Log(msg map[string]interface{}) {
	l.output(msg, 3)
}

func (l *Logger) LogMessage(msgType string, prio int, tags []string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgType,
		"priority": prio,
		"tags":     tags,
	}
	l.output(msg, 3)
}

func (l *Logger) LogMessagef(msgType string, prio int, tags []string, format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgType,
		"priority": prio,
		"tags":     tags,
	}
	l.output(msg, 3)
}

func (l *Logger) logMessage(msgType string, prio int, tags []string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgType,
		"priority": prio,
		"tags":     tags,
	}
	l.output(msg, 4)
}

func (l *Logger) logMessagef(msgType string, prio int, tags []string, format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgType,
		"priority": prio,
		"tags":     tags,
	}
	l.output(msg, 4)
}

func (l *Logger) LogPreamble(v ...interface{}) {
	l.logMessage(msgTypePreamble, PrioNotice, nil, v...)
}

func (l *Logger) LogPreamblef(format string, v ...interface{}) {
	l.logMessagef(msgTypePreamble, PrioNotice, nil, format, v...)
}

func (l *Logger) LogRead(handle string, v ...interface{}) {
	l.logMessage(msgTypeRead, PrioDebug, nil, v...)
}

func (l *Logger) LogReadf(handle, format string, v ...interface{}) {
	l.logMessagef(msgTypeRead, PrioDebug, nil, format, v...)
}

func (l *Logger) LogWrite(handle string, v ...interface{}) {
	l.logMessage(msgTypeWrite, PrioDebug, nil, v...)
}

func (l *Logger) LogWritef(handle, format string, v ...interface{}) {
	l.logMessagef(msgTypeWrite, PrioDebug, nil, format, v...)
}
func (l *Logger) LogError(v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioError, nil, v...)
}

func (l *Logger) LogErrorf(format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioError, nil, format, v...)
}

func (l *Logger) LogWarning(v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioWarning, nil, v...)
}

func (l *Logger) LogWarningf(format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioWarning, nil, format, v...)
}

func (l *Logger) LogInfo(v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioInfo, nil, v...)
}

func (l *Logger) LogInfof(format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioInfo, nil, format, v...)
}

func (l *Logger) LogDebug(v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioDebug, nil, v...)
}

func (l *Logger) LogDebugf(format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioDebug, nil, format, v...)
}

func (l *Logger) LogErrorTagged(tags []string, v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioError, tags, v...)
}

func (l *Logger) LogErrorTaggedf(tags []string, format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioError, tags, format, v...)
}

func (l *Logger) LogWarningTagged(tags []string, v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioWarning, tags, v...)
}

func (l *Logger) LogWarningTaggedf(tags []string, format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioWarning, tags, format, v...)
}

func (l *Logger) LogInfoTagged(tags []string, v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioInfo, tags, v...)
}

func (l *Logger) LogInfoTaggedf(tags []string, format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioInfo, tags, format, v...)
}

func (l *Logger) LogDebugTagged(tags []string, v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioDebug, tags, v...)
}

func (l *Logger) LogDebugTaggedf(tags []string, format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioDebug, tags, format, v...)
}
