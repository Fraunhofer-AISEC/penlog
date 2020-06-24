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
	Timespec       string
	CompLen        int
	TypeLen        int
	LogFmt         string
	LogLevel       Prio
	ShowColors     bool
	ShowLines      bool
	ShowStacktrace bool
	TinyFormat     bool
}

func NewHRFormatter() *HRFormatter {
	return &HRFormatter{
		Timespec:       time.StampMilli,
		CompLen:        8,
		TypeLen:        8,
		LogLevel:       PrioDebug,
		ShowColors:     true,
		ShowLines:      true,
		ShowStacktrace: true,
		TinyFormat:     getEnvBool("PENLOG_HR_TINY"),
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
		if p, ok := prio.(float64); ok {
			priority = Prio(p)
		}
	}

	fmtStr := "%s"
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
	if f.ShowLines {
		if line, ok := msg["line"]; ok {
			if f.ShowColors {
				fmtStr += " " + Colorize(ColorBlue, "(%s)")
			} else {
				fmtStr += " " + "(%s)"
			}
			payload = fmt.Sprintf(fmtStr, payload, line)
		}
	}
	tsParsed, err := time.Parse("2006-01-02T15:04:05.000000", ts)
	if err != nil {
		return "", err
	}

	ts = tsParsed.Format(f.Timespec)
	var out string
	if f.TinyFormat {
		out = fmt.Sprintf("%s: %s", ts, payload)
	} else {
		comp = padOrTruncate(comp, f.CompLen)
		msgType = padOrTruncate(msgType, f.TypeLen)
		out = fmt.Sprintf("%s {%s} [%s]: %s", ts, comp, msgType, payload)
	}
	if f.ShowStacktrace {
		if rawVal, ok := msg["stacktrace"]; ok {
			if val, ok := rawVal.(string); ok {
				out += "\n"
				for _, line := range strings.Split(val, "\n") {
					out += "  |"
					out += line
					out += "\n"
				}
			}
		}
	}
	return out, nil
}
