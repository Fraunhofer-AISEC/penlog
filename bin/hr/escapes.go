package main

import (
	"fmt"
	"os"
)

const (
	colorNop    = ""
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorGray   = "\033[0;38;5;245m"
	clearLine   = "\033[2K"
)

func colorize(color, s string) string {
	if color == colorNop {
		return s
	}
	return color + s + colorReset
}

func colorEprintf(color string, colorized bool, format string, args ...interface{}) {
	if colorized {
		fmt.Fprintf(os.Stderr, colorize(color, format), args...)
	} else {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
