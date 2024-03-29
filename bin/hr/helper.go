// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
	"golang.org/x/sys/unix"
)

func padOrTruncate(s string, maxLen int) string {
	res := s
	if len(s) > maxLen {
		res = s[:maxLen]
	} else if len(s) < maxLen {
		res += strings.Repeat(" ", maxLen-len(s))
	}
	return res
}

func isatty(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	return err == nil
}

func castField(data map[string]interface{}, field string) (string, error) {
	if vIface, ok := data[field]; ok {
		if vString, ok := vIface.(string); ok {
			return vString, nil
		}
		return "", fmt.Errorf("%w: field '%s' is not a string", errInvalidData, field)
	}
	return "", fmt.Errorf("%w: field '%s' does not exist in data", errInvalidData, field)
}

func createErrorRecord(msg string) map[string]interface{} {
	var record = map[string]interface{}{
		"timestamp": "NONE",
		"data":      msg,
		"component": "JSON",
		"type":      "ERROR",
	}
	return record
}

func removeEmpy(data []string) []string {
	b := data[:0]
	for _, x := range data {
		x = strings.TrimSpace(x)
		if x != "" {
			b = append(b, x)
		}
	}
	return b
}

func getReader(filename string) (io.Reader, error) {
	var reader io.Reader
	if s, err := os.Stat(filename); err != nil {
		return nil, err
	} else {
		if s.IsDir() {
			return nil, fmt.Errorf("%s: is a directory", filename)
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	switch filepath.Ext(filename) {
	case ".gz":
		reader, err = gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
	case ".zst":
		reader, err = zstd.NewReader(file)
		if err != nil {
			return nil, err
		}
	default:
		reader = file
	}
	return reader, nil
}

func copyData(data map[string]interface{}) map[string]interface{} {
	d := make(map[string]interface{})
	for k, v := range data {
		d[k] = v
	}
	return d
}

type broadcaster struct {
	inCh   chan map[string]interface{}
	outChs []chan map[string]interface{}
	wg     *sync.WaitGroup
}

func (bc *broadcaster) serve() {
	for data := range bc.inCh {
		for _, listener := range bc.outChs {
			d := copyData(data)
			listener <- d
		}
	}
	for _, ch := range bc.outChs {
		close(ch)
	}
	bc.wg.Done()
}
