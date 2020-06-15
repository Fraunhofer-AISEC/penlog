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
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Fraunhofer-AISEC/penlog"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/pflag"
)

var (
	errInvalidData = errors.New("Invalid data")
)

type compressor interface {
	io.WriteCloser
	Flush() error
}

type converter struct {
	timespec     string
	compLen      int
	typeLen      int
	logFmt       string
	jq           string
	colors       bool
	showLines    bool
	prioLevel    int
	filters      []*filter
	stdoutFilter *filter

	cleanedUp   bool
	workers     int
	broadcastCh chan map[string]interface{}
	writers     []chan map[string]interface{}
	mutex       sync.Mutex
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
		switch determineFilterType(spec) {
		case filterTypeSimple:
			filter, err := parseSimpleFilter(spec)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			// stdout requires special treatment.
			if filter.simpleSpec.filename == "-" {
				c.stdoutFilter = filter
				continue
			}

			file, err := os.Create(filter.simpleSpec.filename)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			dataCh := make(chan map[string]interface{})
			c.workers++
			c.writers = append(c.writers, dataCh)
			go c.fileWorker(&c.wg, dataCh, file, filter)
		default:
			panic("BUG: bogos filter spec")
		}
	}
	c.initializeOutstreams()
}

func (c *converter) addPrioFilter(spec string) error {
	if val, err := strconv.ParseInt(spec, 10, 64); err == nil {
		c.prioLevel = int(val)
		return nil
	}
	switch strings.ToLower(spec) {
	case "debug":
		c.prioLevel = 7
	case "info":
		c.prioLevel = 6
	case "notice":
		c.prioLevel = 5
	case "warning":
		c.prioLevel = 4
	case "error":
		c.prioLevel = 3
	case "critical":
		c.prioLevel = 2
	case "alert":
		c.prioLevel = 1
	case "emergency":
		c.prioLevel = 0
	default:
		return fmt.Errorf("invalid priolevel '%s'", spec)
	}
	return nil
}

func (c *converter) initializeOutstreams() {
	if c.workers > 0 {
		c.workers++
		bc := broadcaster{
			inCh:   c.broadcastCh,
			outChs: c.writers,
			wg:     &c.wg,
		}
		go bc.serve()
	}
	c.wg.Add(c.workers)
}

func (c *converter) transformLine(line map[string]interface{}) (string, error) {
	var (
		payload  string
		priority int = penlog.PrioInfo // This prio is not colorized.
	)

	ts, err := castField(line, "timestamp")
	if err != nil {
		return "", err
	}
	comp, err := castField(line, "component")
	if err != nil {
		return "", err
	}
	msgType, err := castField(line, "type")
	if err != nil {
		return "", err
	}
	if prio, ok := line["priority"]; ok {
		if p, ok := prio.(float64); ok {
			priority = int(p)
		}
	}

	// Type switch for the data field. We support string and a list
	// of strings. The reflect stuff is a bit ugly, but it works... :)
	switch v := line["data"].(type) {
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
	if c.colors {
		switch priority {
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
	payload = fmt.Sprintf(fmtStr, payload)
	if c.showLines {
		if line, ok := line["line"]; ok {
			if c.colors {
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

func (c *converter) transform(scanner *bufio.Scanner) {
	for scanner.Scan() {
		if jsonLine := scanner.Bytes(); len(bytes.TrimSpace(jsonLine)) > 0 {
			var data map[string]interface{}
			if err := json.Unmarshal(jsonLine, &data); err != nil {
				// TODO: log error here
				continue
			}
			if c.workers > 0 {
				c.mutex.Lock()
				// Avoid sends on closed channel by signal handler.
				if c.cleanedUp {
					c.mutex.Unlock()
					break
				}
				d := copyData(data)
				c.broadcastCh <- d
				c.mutex.Unlock()
			}

			var (
				err error
				d   = copyData(data)
			)
			if c.stdoutFilter != nil {
				d, err = c.stdoutFilter.filter(d)
				if err != nil {
					// TODO: log error
					continue
				}
				if d == nil {
					continue
				}
			}
			if prio, ok := d["priority"]; ok {
				if p, ok := prio.(float64); ok {
					if int(p) > c.prioLevel {
						continue
					}
				}
			}
			if hrLine, err := c.transformLine(d); err == nil {
				fmt.Println(hrLine)
			} else {
				if errors.Is(err, errInvalidData) {
					if c.colors {
						fmt.Fprintf(os.Stderr, colorize(colorRed, "error: %s\n"), err)
					} else {
						fmt.Fprintf(os.Stderr, "error: %s\n", err)
					}
					continue
				}
				if c.colors {
					fmt.Fprintf(os.Stderr, colorize(colorRed, "error: %s\n"), scanner.Text())
				} else {
					fmt.Fprintf(os.Stderr, "error: %s\n", scanner.Text())
				}
			}
		}
	}
	if scanner.Err() != nil {
		if c.colors {
			fmt.Fprintf(os.Stderr, colorize(colorRed, "error: read: %s\n"), scanner.Err())
		} else {
			fmt.Fprintf(os.Stderr, "error: read: %s\n", scanner.Err())
		}
	}
}

func (c *converter) fileWorker(wg *sync.WaitGroup, data chan map[string]interface{}, file *os.File, fil *filter) {
	var (
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
		l, err := fil.filter(line)
		if l == nil || err != nil {
			continue
		}
		// TODO: maybe an encoder works here?
		b, _ := json.Marshal(l)
		fileWriter.Write(b)
		fileWriter.WriteRune('\n')
	}

	fileWriter.Flush()
	if comp != nil {
		comp.Flush()
		comp.Close()
	}
	file.Close()
	wg.Done()
}

func createJQ(r io.Reader, filter string) (*bufio.Scanner, *exec.Cmd, error) {
	cmd := exec.Command("jq", "-c", "--unbuffered", filter)
	cmd.Stderr = os.Stderr
	jqOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	jqIn, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	go func() {
		if _, err := io.Copy(jqIn, r); err != nil {
			panic(err)
		}
		jqIn.Close()
	}()
	return bufio.NewScanner(jqOut), cmd, nil
}

func main() {
	var (
		err          error
		filterSpecs  []string
		prioLevelRaw string
		colorsCli    bool
		conv         = converter{
			workers:     0,
			broadcastCh: make(chan map[string]interface{}),
			cleanedUp:   false,
		}
	)

	pflag.BoolVar(&colorsCli, "colors", true, "enable colorized output based on priorities")
	pflag.BoolVar(&conv.showLines, "lines", true, "show line numbers if available")
	pflag.StringVarP(&conv.timespec, "timespec", "s", time.StampMilli, "timespec in output")
	pflag.StringVarP(&conv.jq, "jq", "j", "", "run the jq tool as a preprocessor")
	pflag.IntVarP(&conv.compLen, "complen", "c", 8, "len of component field")
	pflag.IntVarP(&conv.typeLen, "typelen", "t", 8, "len of type field")
	pflag.StringVarP(&prioLevelRaw, "priority", "p", "debug", "show messages with a lower priority level")
	pflag.StringArrayVarP(&filterSpecs, "filter", "f", []string{}, "write logs to a file with filters")
	cpuprofile := pflag.String("cpuprofile", "", "write cpu profile to `file`")
	pflag.Parse()

	conv.logFmt = "%s {%s} [%s]: %s"

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not create CPU profile: %s\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "could not start CPU profile: %s\n", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	conv.addFilterSpecs(filterSpecs)
	if err := conv.addPrioFilter(prioLevelRaw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

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

	conv.colors = colorsCli
	if colorsCli {
		if !isatty(uintptr(syscall.Stdout)) {
			conv.colors = false
		}
		if valRaw, ok := os.LookupEnv("PENLOG_FORCE_COLORS"); ok {
			if val, err := strconv.ParseBool(valRaw); val && err == nil {
				conv.colors = colorsCli
			}
		}
	}
	if valRaw, ok := os.LookupEnv("PENLOG_SHOW_LINES"); ok {
		if val, err := strconv.ParseBool(valRaw); val && err == nil {
			conv.showLines = val
		}
	}

	var jq *exec.Cmd
	if isatty(uintptr(syscall.Stdin)) {
		for _, file := range pflag.Args() {
			reader = getReader(file)
			if conv.jq != "" {
				scanner, jq, err = createJQ(reader, conv.jq)
				if err != nil {
					panic(err)
				}
			} else {
				scanner = bufio.NewScanner(reader)
			}
			conv.transform(scanner)
			if jq != nil {
				jq.Process.Kill()
				jq.Wait()
			}
		}
	} else {
		if conv.jq != "" {
			scanner, jq, err = createJQ(reader, conv.jq)
			if err != nil {
				panic(err)
			}
		}
		conv.transform(scanner)
		if jq != nil {
			jq.Process.Kill()
			jq.Wait()
		}
	}
	conv.cleanup()
}
