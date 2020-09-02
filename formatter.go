// SPDX-License-Identifier: Apache-2.0

package penlog

import (
	"fmt"
	"strings"
	"time"
)

const (
	ColorNop    = ""
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[0;38;5;245m"
)

func Colorize(color, s string) string {
	if color == ColorNop {
		return s
	}
	return color + s + ColorReset
}

func castField(data map[string]interface{}, field string) (string, error) {
	if vIface, ok := data[field]; ok {
		if vString, ok := vIface.(string); ok {
			return vString, nil
		}
		return "", fmt.Errorf("field '%s' is not a string", field)
	}
	return "", fmt.Errorf("field '%s' does not exist in data", field)
}

func padOrTruncate(s string, maxLen int) string {
	res := s
	if len(s) > maxLen {
		res = s[:maxLen]
	} else if len(s) < maxLen {
		res += strings.Repeat(" ", maxLen-len(s))
	}
	return res
}

type HRFormatter struct {
	Timespec        string
	CompLen         int
	TypeLen         int
	LogFmt          string
	LogLevel        Prio
	ShowColors      bool
	ShowLines       bool
	ShowStacktraces bool
	ShowLevelPrefix bool
	ShowID          bool
	ShowTags        bool
	TinyFormat      bool
}

func NewHRFormatter() *HRFormatter {
	return &HRFormatter{
		Timespec:        time.StampMilli,
		CompLen:         8,
		TypeLen:         8,
		LogLevel:        PrioDebug,
		ShowColors:      false,
		ShowLines:       false,
		ShowStacktraces: false,
		ShowLevelPrefix: false,
		ShowTags:        false,
		TinyFormat:      true,
	}
}

func (f *HRFormatter) Format(msg map[string]interface{}) (string, error) {
	var priority Prio = PrioInfo // This prio is not colorized.

	payload, err := castField(msg, "data")
	if err != nil {
		return "", err
	}
	ts, err := castField(msg, "timestamp")
	if err != nil {
		return "", err
	}
	comp, err := castField(msg, "component")
	if err != nil {
		return "", err
	}
	msgType, err := castField(msg, "type")
	if err != nil {
		return "", err
	}
	if prio, ok := msg["priority"]; ok {
		switch p := prio.(type) {
		case Prio:
			priority = p
		case int:
			priority = Prio(p)
		case float64:
			priority = Prio(p)
		}
	}

	fmtStr := "%s"
	if f.ShowLevelPrefix {
		switch priority {
		case PrioEmergency:
			fmtStr = "[E] %s"
		case PrioAlert:
			fmtStr = "[A] %s"
		case PrioCritical:
			fmtStr = "[C] %s"
		case PrioError:
			fmtStr = "[E] %s"
		case PrioWarning:
			fmtStr = "[w] %s"
		case PrioNotice:
			fmtStr = "[n] %s"
		case PrioInfo:
			fmtStr = "[i] %s"
		case PrioDebug:
			fmtStr = "[d] %s"
		}
		if comp == "JSON" && msgType == "ERROR" {
			fmtStr = "[!] %s"
		}
	}
	if f.ShowColors {
		switch priority {
		case PrioEmergency,
			PrioAlert,
			PrioCritical,
			PrioError:
			fmtStr = Colorize(ColorBold, Colorize(ColorRed, "%s"))
		case PrioWarning:
			fmtStr = Colorize(ColorBold, Colorize(ColorYellow, "%s"))
		case PrioNotice:
			fmtStr = Colorize(ColorBold, "%s")
		case PrioInfo:
		case PrioDebug:
			fmtStr = Colorize(ColorGray, "%s")
		}
		if comp == "JSON" && msgType == "ERROR" {
			fmtStr = Colorize(ColorRed, "%s")
		}
	}
	payload = fmt.Sprintf(fmtStr, payload)

	if ts == "NONE" {
        ts = "0000000000000000000"
	} else {
		var tsParsed time.Time
		tsParsed, err = time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			tsParsed, err = time.Parse("2006-01-02T15:04:05.000000", ts)
			if err != nil {
				return "", err
			}
		}
		ts = tsParsed.Format(f.Timespec)
	}

	var out string
	if f.TinyFormat {
		out = fmt.Sprintf("%s: %s", ts, payload)
	} else {
		comp = padOrTruncate(comp, f.CompLen)
		msgType = padOrTruncate(msgType, f.TypeLen)
		out = fmt.Sprintf("%s {%s} [%s]: %s", ts, comp, msgType, payload)
	}
	if f.ShowID {
		if rawVal, ok := msg["id"]; ok {
			if val, ok := rawVal.(string); ok {
				out += "\n"
				out += "  => id  : "
				if f.ShowColors {
					out += Colorize(ColorYellow, val)
				} else {
					out += val
				}
			}
		}
	}
	if f.ShowLines {
		if line, ok := msg["line"]; ok {
			out += "\n"
			out += "  => line: "
			if f.ShowColors {
				out += fmt.Sprintf(Colorize(ColorBlue, "%s"), line)
			} else {
				out += fmt.Sprintf("%s", line)
			}
		}
	}
	if f.ShowTags {
		if rawVal, ok := msg["tags"]; ok {
			if val, ok := rawVal.([]interface{}); ok && len(val) > 0 {
				out += "\n"
				out += "  => tags: "
				for _, tag := range val {
					out += fmt.Sprintf("%v ", tag)
				}
			}
		}
	}
	if f.ShowStacktraces {
		if rawVal, ok := msg["stacktrace"]; ok {
			if val, ok := rawVal.(string); ok {
				out += "\n"
				out += "  => stacktrace: \n"
				for _, line := range strings.Split(val, "\n") {
					if f.ShowColors {
						out += Colorize(ColorGray, "  |")
						out += Colorize(ColorGray, line)
					} else {
						out += "  |"
						out += line
					}
					out += "\n"
				}
			}
		}
	}
	return out, nil
}
