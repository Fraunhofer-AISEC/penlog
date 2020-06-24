// SPDX-License-Identifier: Apache-2.0

// Package penlog implements the penlog(7) specification.
package penlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/v22/journal"
)

type Prio int

// RFC5424 Section 6.2.1
const (
	PrioEmergency Prio = iota
	PrioAlert
	PrioCritical
	PrioError
	PrioWarning
	PrioNotice
	PrioInfo
	PrioDebug
)

type Logger struct {
	HRFormatter *HRFormatter

	host           string
	component      string
	timespec       string
	writer         io.Writer
	buf            bytes.Buffer
	mu             sync.Mutex
	lines          bool
	stacktrace     bool
	systemdJournal bool
	jsonStderr     bool
	loglevel       Prio
}

const (
	msgTypeRead     = "read"
	msgTypeWrite    = "write"
	msgTypeMessage  = "message"
	msgTypePreamble = "preamble"
)

func getLineNumber(depth int) string {
	if _, file, line, ok := runtime.Caller(depth); ok {
		return fmt.Sprintf("%s:%d", file, line)
	}
	return ""
}

func getEnvBool(name string) bool {
	if rawVal, ok := os.LookupEnv(name); ok {
		if val, err := strconv.ParseBool(rawVal); val && err == nil {
			return val
		}
	}
	return false
}

func NewLogger(component string, w io.Writer) *Logger {
	systemdJournal := getEnvBool("PENLOG_SYSTEMD_JOURNAL")

	if systemdJournal && !journal.Enabled() {
		panic("systemd-journal is not available")
	}

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
		HRFormatter:    NewHRFormatter(),
		host:           hostname,
		loglevel:       PrioDebug,
		component:      component,
		timespec:       "2006-01-02T15:04:05.000000",
		lines:          getEnvBool("PENLOG_LINES"),
		stacktrace:     getEnvBool("PENLOG_STACKTRACE"),
		systemdJournal: systemdJournal,
		jsonStderr:     getEnvBool("PENLOG_HR"),
		writer:         w,
	}
}

func (l *Logger) EnableLines(enable bool) {
	l.mu.Lock()
	l.lines = enable
	l.mu.Unlock()
}

func (l *Logger) SetLogLevel(prio Prio) {
	l.mu.Lock()
	l.loglevel = prio
	l.mu.Unlock()
}

func (l *Logger) GetLogLevel(prio Prio) Prio {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.loglevel
}

func convertVarsForJournal(in map[string]interface{}) map[string]string {
	// penlog fields are converted to strings suitable for systemd journal
	var (
		out = make(map[string]string)
		re  = regexp.MustCompile(`(.+):([0-9]+)$`)
	)

	if rawVal, ok := in["component"]; ok {
		if val, ok := rawVal.(string); ok {
			out["COMPONENT"] = val
		}
	}
	if rawVal, ok := in["line"]; ok {
		if val, ok := rawVal.(string); ok {
			m := re.FindStringSubmatch(val)
			if m != nil {
				out["CODE_FILE"] = m[1]
				out["CODE_LINE"] = m[2]
			}
		}
	}
	if rawVal, ok := in["stacktrace"]; ok {
		if val, ok := rawVal.(string); ok {
			out["STACKTRACE"] = val
		}
	}
	if rawVal, ok := in["tags"]; ok {
		if val, ok := rawVal.([]string); ok {
			out["TAGS"] = strings.Join(val, ", ")
		}
	}
	return out
}

func (l *Logger) outputJournal(msg map[string]interface{}) {
	var (
		data string
		prio = -1
		vars = convertVarsForJournal(msg)
	)
	if rawVal, ok := msg["priority"]; ok {
		if val, ok := rawVal.(int); ok {
			prio = val
		}
	}
	if prio == -1 {
		prio = int(PrioInfo)
	}
	if rawData, ok := msg["data"]; ok {
		if val, ok := rawData.(string); ok {
			data = val
		}
	}
	if err := journal.Send(data, journal.Priority(prio), vars); err != nil {
		panic(err)
	}
}

func (l *Logger) outputHr(msg map[string]interface{}) {
	line, err := l.HRFormatter.Format(msg)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "%s\n", line)
}

func (l *Logger) output(msg map[string]interface{}, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if rawVal, ok := msg["priority"]; ok {
		if val, ok := rawVal.(Prio); ok {
			if val < l.loglevel {
				return
			}
		}
	}

	msg["timestamp"] = time.Now().Format(l.timespec)
	msg["component"] = l.component
	msg["host"] = l.host
	if l.lines {
		msg["line"] = getLineNumber(depth)
	}
	if l.stacktrace {
		msg["stacktrace"] = string(debug.Stack())
	}

	if l.systemdJournal {
		l.outputJournal(msg)
		return
	}
	if l.jsonStderr {
		b, err := json.Marshal(msg)
		if err != nil {
			// This is clearly a bug!
			panic(err)
		}

		l.buf.Write(b)
		l.buf.WriteString("\n")
		l.buf.WriteTo(l.writer)
	} else {
		panic("hr output is not yet implemented")
	}
}

func (l *Logger) Log(msg map[string]interface{}) {
	l.output(msg, 3)
}

func (l *Logger) LogMessage(msgType string, prio Prio, tags []string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgType,
		"priority": prio,
		"tags":     tags,
	}
	l.output(msg, 3)
}

func (l *Logger) LogMessagef(msgType string, prio Prio, tags []string, format string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprintf(format, v...),
		"type":     msgType,
		"priority": prio,
		"tags":     tags,
	}
	l.output(msg, 3)
}

func (l *Logger) logMessage(msgType string, prio Prio, tags []string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgType,
		"priority": prio,
		"tags":     tags,
	}
	l.output(msg, 4)
}

func (l *Logger) logMessagef(msgType string, prio Prio, tags []string, format string, v ...interface{}) {
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

func (l *Logger) LogNotice(v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioNotice, nil, v...)
}

func (l *Logger) LogNoticef(format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioNotice, nil, format, v...)
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
