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
	"github.com/google/uuid"
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

type OutType int

const (
	OutTypeHR OutType = iota
	OutTypeHRTiny
	OutTypeJSON
	OutTypeJSONPretty
	OutTypeSystemdJournal
)

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

type Logger struct {
	hrFormatter *HRFormatter
	host        string
	component   string
	timespec    string
	writer      io.Writer
	buf         bytes.Buffer
	mu          sync.Mutex
	lines       bool
	stacktrace  bool
	loglevel    Prio
	outputType  OutType
}

func NewLogger(component string, w io.Writer) *Logger {
	var (
		outputType  OutType
		hrFormatter = NewHRFormatter()
	)
	switch strings.ToLower(os.Getenv("PENLOG_OUTPUT")) {
	case "", "hr-nano":
		outputType = OutTypeHRTiny
		hrFormatter.Dialect = HRNano
	case "hr-tiny":
		outputType = OutTypeHR
		hrFormatter.Dialect = HRTiny
	case "hr", "hr-full":
		outputType = OutTypeHR
		hrFormatter.Dialect = HRFull
	case "json":
		outputType = OutTypeJSON
	case "json-pretty":
		outputType = OutTypeJSONPretty
	case "systemd":
		outputType = OutTypeSystemdJournal
	default:
		panic("invalid penlog output")
	}
	if outputType == OutTypeSystemdJournal && !journal.Enabled() {
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

	loglevel := PrioDebug
	if rawVal, ok := os.LookupEnv("PENLOG_LOGLEVEL"); ok {
		switch strings.ToLower(rawVal) {
		case "critical":
			loglevel = PrioCritical
		case "error":
			loglevel = PrioError
		case "warning":
			loglevel = PrioWarning
		case "notice":
			loglevel = PrioNotice
		case "info":
			loglevel = PrioInfo
		case "debug":
			loglevel = PrioDebug
		}
	}

	return &Logger{
		hrFormatter: hrFormatter,
		host:        hostname,
		loglevel:    loglevel,
		component:   component,
		timespec:    time.RFC3339Nano,
		lines:       getEnvBool("PENLOG_CAPTURE_LINES"),
		stacktrace:  getEnvBool("PENLOG_CAPTURE_STACKTRACES"),
		outputType:  outputType,
		writer:      w,
	}
}

func (l *Logger) SetColors(enable bool) {
	l.mu.Lock()
	l.hrFormatter.ShowColors = enable
	l.mu.Unlock()
}

func (l *Logger) SetLevelPrefix(enable bool) {
	l.mu.Lock()
	l.hrFormatter.ShowLevelPrefix = enable
	l.mu.Unlock()
}

func (l *Logger) SetLines(enable bool) {
	l.mu.Lock()
	l.lines = enable
	l.mu.Unlock()
}

func (l *Logger) SetStacktrace(enable bool) {
	l.mu.Lock()
	l.stacktrace = enable
	l.mu.Unlock()
}

func (l *Logger) SetLogLevel(prio Prio) {
	l.mu.Lock()
	l.loglevel = prio
	l.mu.Unlock()
}

func convertVarsForJournal(in map[string]interface{}) map[string]string {
	// penlog fields are converted to strings suitable for systemd journal
	var (
		out = make(map[string]string)
		re  = regexp.MustCompile(`(.+):([0-9]+)$`)
	)

	if rawVal, ok := in["id"]; ok {
		if val, ok := rawVal.(string); ok {
			out["MESSAGE_ID"] = val
		}
	}

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
		if val, ok := rawVal.(Prio); ok {
			prio = int(val)
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

func (l *Logger) outputHR(msg map[string]interface{}) {
	line, err := l.hrFormatter.Format(msg)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(l.writer, "%s\n", line)
}

func (l *Logger) output(msg map[string]interface{}, depth int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if rawVal, ok := msg["priority"]; ok {
		if val, ok := rawVal.(Prio); ok {
			if val > l.loglevel {
				return
			}
		}
	}
	var (
		now = time.Now()
		id  = uuid.New()
	)
	msg["id"] = id.String()
	msg["timestamp"] = now.Format(l.timespec)
	msg["component"] = l.component
	msg["host"] = l.host
	if l.lines {
		msg["line"] = getLineNumber(depth)
	}
	if l.stacktrace {
		msg["stacktrace"] = string(debug.Stack())
	}

	switch l.outputType {
	// hr and hr-tiny are set in the formatter
	case OutTypeHR, OutTypeHRTiny:
		l.outputHR(msg)
	case OutTypeJSON:
		b, err := json.Marshal(msg)
		if err != nil {
			// This is clearly a bug!
			panic(err)
		}
		l.buf.Write(b)
		l.buf.WriteString("\n")
		l.buf.WriteTo(l.writer)
	case OutTypeJSONPretty:
		b, err := json.MarshalIndent(msg, "", "  ")
		if err != nil {
			// This is clearly a bug!
			panic(err)
		}
		l.buf.Write(b)
		l.buf.WriteString("\n")
		l.buf.WriteTo(l.writer)
	case OutTypeSystemdJournal:
		l.outputJournal(msg)
	default:
		panic("BUG: impossible output type")
	}
}

// Log is the lowest level logging primitive. Any fields can be added
// to the emitted logging message. The penlog fields id, timestamp,
// timezone, component, host, line, and stacktrace are added by the
// underlying logic. Besides this any field can be added. Be aware of
// the mandatory fields in the penlog(7) specification. There might be
// dragons.
func (l *Logger) Log(msg map[string]interface{}) {
	l.output(msg, 3)
}

// LogMessage is a higher level interface for emitting log messages.
// In contrast to Log it is not possible to emit invalid messages by
// accident. The following penlog fields are filled: data, type, priority
// tags. Tags might be nil.
func (l *Logger) LogMessage(msgType string, prio Prio, tags []string, v ...interface{}) {
	var msg = map[string]interface{}{
		"data":     fmt.Sprint(v...),
		"type":     msgType,
		"priority": prio,
		"tags":     tags,
	}
	l.output(msg, 3)
}

// LogMessagef is the same as LogMessage except that it takes a Printf
// like format string and the relevant arguments.
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

// Write is the implemantation of the Go Writer interface. This
// method is very limited; it emits messages of type "message"
// at the priority info. The writer interface might be useful to
// pass a penlog logger to a library which supports a Writer for
// logging purposes.
func (l *Logger) Write(p []byte) (int, error) {
	l.logMessage(msgTypeMessage, PrioInfo, nil, string(p))
	return len(p), nil
}

func (l *Logger) Print(v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioInfo, nil, v...)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioInfo, nil, format, v...)
}

func (l *Logger) LogPreamble(v ...interface{}) {
	l.logMessage(msgTypePreamble, PrioNotice, nil, v...)
}

func (l *Logger) LogPreamblef(format string, v ...interface{}) {
	l.logMessagef(msgTypePreamble, PrioNotice, nil, format, v...)
}

func (l *Logger) LogRead(v ...interface{}) {
	l.logMessage(msgTypeRead, PrioDebug, nil, v...)
}

func (l *Logger) LogReadf(format string, v ...interface{}) {
	l.logMessagef(msgTypeRead, PrioDebug, nil, format, v...)
}

func (l *Logger) LogWrite(v ...interface{}) {
	l.logMessage(msgTypeWrite, PrioDebug, nil, v...)
}

func (l *Logger) LogWritef(format string, v ...interface{}) {
	l.logMessagef(msgTypeWrite, PrioDebug, nil, format, v...)
}

func (l *Logger) LogCritical(v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioCritical, nil, v...)
}

func (l *Logger) LogCriticalf(format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioCritical, nil, format, v...)
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

func (l *Logger) LogReadTagged(tags []string, v ...interface{}) {
	l.logMessage(msgTypeRead, PrioDebug, tags, v...)
}

func (l *Logger) LogReadfTagged(tags []string, format string, v ...interface{}) {
	l.logMessagef(msgTypeRead, PrioDebug, tags, format, v...)
}

func (l *Logger) LogWriteTagged(tags []string, v ...interface{}) {
	l.logMessage(msgTypeWrite, PrioDebug, tags, v...)
}

func (l *Logger) LogWritefTagged(tags []string, format string, v ...interface{}) {
	l.logMessagef(msgTypeWrite, PrioDebug, tags, format, v...)
}

func (l *Logger) LogCriticalTagged(tags []string, v ...interface{}) {
	l.logMessage(msgTypeMessage, PrioCritical, tags, v...)
}

func (l *Logger) LogCriticalTaggedf(tags []string, format string, v ...interface{}) {
	l.logMessagef(msgTypeMessage, PrioCritical, tags, format, v...)
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
