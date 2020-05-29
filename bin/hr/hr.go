// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.sr.ht/~rumpelsepp/helpers"
	"github.com/Fraunhofer-AISEC/penlog"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/pflag"
	"golang.org/x/sys/unix"
)

const (
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
)

var (
	errIsFiltered  = errors.New("Line is filtered")
	errInvalidData = errors.New("Invalid data")
)

func colorize(color, s string) string {
	return color + s + colorReset
}

type converter struct {
	timespec    string
	typeFilters []string
	compFilters []string
	compLen     int
	typeLen     int
	logFmt      string
	color       bool
	showLines   bool

	cleanedUp   bool
	workers     int
	broadcastCh chan []byte
	writers     []chan []byte
	mutex       sync.Mutex
	pool        sync.Pool
	wg          sync.WaitGroup
}

func (c *converter) cleanup() {
	c.mutex.Lock()
	if c.cleanedUp {
		c.mutex.Unlock()
		return
	}
	if c.workers > 0 {
		close(c.broadcastCh)
		c.wg.Wait()
	}
	c.cleanedUp = true
	c.mutex.Unlock()
}

func (c *converter) addFilterSpecs(specs []string) {
	for _, spec := range specs {
		filter, err := parseFilter(spec)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// stdout requires special treatment.
		if filter["file"][0] == "-" {
			c.compFilters = filter["components"]
			c.typeFilters = filter["filters"]
			continue
		}

		file, err := os.Create(filter["file"][0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		dataCh := make(chan []byte)
		c.workers++
		c.writers = append(c.writers, dataCh)
		go bufferedWriter(&c.wg, &c.pool, dataCh, file, filter)
	}
	c.initializeOutstreams()
}

func (c *converter) initializeOutstreams() {
	if c.workers > 0 {
		c.workers++
		bc := helpers.Broadcaster{
			InCh:    c.broadcastCh,
			OutChs:  c.writers,
			MemPool: &c.pool,
			WG:      &c.wg,
		}
		go bc.Serve()
	}
	c.wg.Add(c.workers)
}

func (c *converter) genHRLine(data map[string]interface{}) (string, error) {
	var payload string

	ts, err := castField(data, "timestamp")
	if err != nil {
		return "", err
	}
	comp, err := castField(data, "component")
	if err != nil {
		return "", err
	}
	msgType, err := castField(data, "type")
	if err != nil {
		return "", err
	}

	if !isFilterMatch(comp, c.compFilters) {
		return "", errIsFiltered
	}
	if !isFilterMatch(msgType, c.typeFilters) {
		return "", errIsFiltered
	}

	// Type switch for the data field. We support string and a list
	// of strings. The reflect stuff is a bit ugly, but it works... :)
	switch v := data["data"].(type) {
	case []interface{}:
		d := make([]string, 0, len(v))
		for _, val := range v {
			s := val.(string)
			d = append(d, s)
		}
		payload = strings.Join(d, " ")
	case string:
		payload = v
	default:
		return "", fmt.Errorf("unsupported data: %v", v)
	}

	fmtStr := "%s"
	if c.color {
		if prio, ok := data["priority"]; ok {
			if p, ok := prio.(float64); ok {
				switch p {
				case penlog.PrioEmergency,
					penlog.PrioAlert,
					penlog.PrioCritical,
					penlog.PrioError:
					fmtStr = colorize(colorBold, colorize(colorRed, "%s"))
				case penlog.PrioWarning:
					fmtStr = colorize(colorBold, colorize(colorYellow, "%s"))
				case penlog.PrioNotice:
					fmtStr = colorize(colorBold, "%s")
				case penlog.PrioInfo:
				case penlog.PrioDebug:
					fmtStr = colorize(colorGray, "%s")
				}
			}
		}
	}
	payload = fmt.Sprintf(fmtStr, payload)
	if c.showLines {
		if line, ok := data["line"]; ok {
			if c.color {
				fmtStr += " " + colorize(colorBlue, "(%s)")
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

	ts = tsParsed.Format(c.timespec)
	comp = padOrTruncate(comp, c.compLen)
	msgType = padOrTruncate(msgType, c.typeLen)
	return fmt.Sprintf(c.logFmt, ts, comp, msgType, payload), nil
}

func (c *converter) transformLine(line []byte) (string, error) {
	var (
		err    error
		parsed map[string]interface{}
	)

	err = json.Unmarshal(line, &parsed)
	if err != nil {
		return "", err
	}

	hrLine, err := c.genHRLine(parsed)
	if err != nil {
		return "", err
	}

	return hrLine, nil
}

func (c *converter) transform(scanner *bufio.Scanner) {
	for scanner.Scan() {
		if jsonLine := scanner.Bytes(); len(bytes.TrimSpace(jsonLine)) > 0 {
			if c.workers > 0 {
				buf := helpers.GetSlice(&c.pool, len(jsonLine))
				n := copy(buf, jsonLine)
				c.mutex.Lock()
				// Avoid sends on closed channel by signal handler.
				if c.cleanedUp {
					c.mutex.Unlock()
					break
				}
				c.broadcastCh <- buf[:n]
				c.mutex.Unlock()
			}

			if hrLine, err := c.transformLine(jsonLine); err == nil {
				fmt.Println(hrLine)
			} else {
				if err == errIsFiltered {
					continue
				}
				if errors.Is(err, errInvalidData) {
					if c.color {
						fmt.Fprintf(os.Stderr, colorize(colorRed, "error: %s\n"), err)
					} else {
						fmt.Fprintf(os.Stderr, "error: %s\n", err)
					}
					continue
				}

				if c.color {
					fmt.Fprintf(os.Stderr, colorize(colorRed, "error: %s\n"), scanner.Text())
				} else {
					fmt.Fprintf(os.Stderr, "error: %s\n", scanner.Text())
				}
			}
		}
	}
	if scanner.Err() != nil {
		if c.color {
			fmt.Fprintf(os.Stderr, colorize(colorRed, "error: read: %s\n"), scanner.Err())
		} else {
			fmt.Fprintf(os.Stderr, "error: read: %s\n", scanner.Err())
		}
	}
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

func logError(w *bufio.Writer, msg string) {
	var line = map[string]string{
		"timestamp": time.Now().Format("2006-01-02T15:04:05.000000"),
		"data":      msg,
		"component": "JSON",
		"type":      "ERROR",
	}
	str, _ := json.Marshal(line)
	w.Write(str)
	w.WriteRune('\n')
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

func parseFilter(filterexpr string) (map[string][]string, error) {
	var (
		parts = strings.SplitN(filterexpr, ":", 3)
		res   = make(map[string][]string)
	)
	switch len(parts) {
	// Only a filename ist specified, no filters.
	case 1:
		res["file"] = []string{parts[0]}
	// Filters and filename is availabe.
	case 2:
		res["filters"] = removeEmpy(strings.Split(parts[0], ","))
		res["file"] = []string{parts[1]}
	// Components, filters, and a filename specified.
	case 3:
		res["components"] = removeEmpy(strings.Split(parts[0], ","))
		res["filters"] = removeEmpy(strings.Split(parts[1], ","))
		res["file"] = []string{parts[2]}
	// Filter expression is invalid.
	default:
		return res, fmt.Errorf("invalid filter expression")
	}
	return res, nil
}

// FIXME: exclusive is broken, thus missing
func isFilterMatch(candidate string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	c := strings.ToLower(candidate)
	for _, filter := range filters {
		f := strings.ToLower(filter)
		if c == f {
			return true
		}
	}
	return false
}

type compressor interface {
	io.WriteCloser
	Flush() error
}

func bufferedWriter(wg *sync.WaitGroup, pool *sync.Pool, data chan []byte, file *os.File, fspec map[string][]string) {
	var (
		parsed     map[string]interface{}
		fileWriter *bufio.Writer
		comp       compressor
	)

	switch filepath.Ext(file.Name()) {
	case ".gz":
		comp = gzip.NewWriter(file)
		fileWriter = bufio.NewWriter(comp)
	case ".zst":
		// error is always nil without options.
		comp, _ = zstd.NewWriter(file)
		fileWriter = bufio.NewWriter(comp)
	default:
		fileWriter = bufio.NewWriter(file)
	}

	for line := range data {
		if err := json.Unmarshal(line, &parsed); err != nil {
			logError(fileWriter, string(line))
			continue
		}
		if !isFilterMatch(parsed["component"].(string), fspec["components"]) {
			continue
		}
		if !isFilterMatch(parsed["type"].(string), fspec["filters"]) {
			continue
		}
		fileWriter.Write(line)
		fileWriter.WriteRune('\n')
		pool.Put(line)
	}

	fileWriter.Flush()
	if comp != nil {
		comp.Flush()
		comp.Close()
	}
	file.Close()
	wg.Done()
}

func getReader(filename string) io.Reader {
	var reader io.Reader
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	switch filepath.Ext(filename) {
	case ".gz":
		reader, err = gzip.NewReader(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case ".zst":
		reader, err = zstd.NewReader(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		reader = file
	}
	return reader
}

func main() {
	var (
		filterSpecs []string
		conv        = converter{
			pool:        helpers.CreateMemPool(),
			workers:     0,
			broadcastCh: make(chan []byte),
			cleanedUp:   false,
		}
	)

	pflag.BoolVar(&conv.color, "color", true, "timespec in output")
	pflag.BoolVar(&conv.showLines, "lines", true, "show line numbers if available")
	pflag.StringVarP(&conv.timespec, "timespec", "s", time.StampMilli, "timespec in output")
	pflag.IntVarP(&conv.compLen, "complen", "c", 8, "len of component field")
	pflag.IntVarP(&conv.typeLen, "typelen", "t", 8, "len of type field")
	pflag.StringVarP(&conv.logFmt, "logformat", "l", "%s {%s} [%s]: %s", "formatstring for a logline")
	pflag.StringArrayVarP(&filterSpecs, "filter", "f", []string{}, "write logs to a file, you can add filters: COMPONENT1:FILTER1:FILENAME")
	cpuprofile := pflag.String("cpuprofile", "", "write cpu profile to `file`")
	pflag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Printf("could not create CPU profile: %s\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Printf("could not start CPU profile: %s\n", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	conv.addFilterSpecs(filterSpecs)

	var (
		reader  io.Reader = os.Stdin
		scanner           = bufio.NewScanner(reader)
		c                 = make(chan os.Signal)
	)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-c
		conv.cleanup()
		os.Exit(1)
	}()

	if !isatty(uintptr(syscall.Stdout)) {
		conv.color = false
	}

	if isatty(uintptr(syscall.Stdin)) {
		for _, file := range pflag.Args() {
			reader = getReader(file)
			scanner = bufio.NewScanner(reader)
			conv.transform(scanner)
		}
	} else {
		conv.transform(scanner)
	}
	conv.cleanup()
}
