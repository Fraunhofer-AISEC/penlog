// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"codeberg.org/rumpelsepp/helpers"
	penlog "github.com/Fraunhofer-AISEC/penlogger"
	jsoniter "github.com/json-iterator/go"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/pflag"
)

var (
	version string
	json    = jsoniter.ConfigCompatibleWithStandardLibrary
)

var (
	errInvalidData = errors.New("Invalid data")
)

type compressor interface {
	io.WriteCloser
	Flush() error
}

type converter struct {
	formatter    *penlog.HRFormatter
	logFmt       string
	logLevel     penlog.Prio
	filters      []*filter
	stdoutFilter *filter
	id           string
	volatileInfo bool

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

func (c *converter) addFilterSpecs(specs []string) error {
	for _, spec := range specs {
		switch determineFilterType(spec) {
		case filterTypeSimple:
			filter, err := parseSimpleFilter(spec)
			if err != nil {
				return err
			}
			// stdout requires special treatment.
			if filter.simpleSpec.filename == "-" {
				c.stdoutFilter = filter
				continue
			}

			file, err := os.Create(filter.simpleSpec.filename)
			if err != nil {
				return err
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
	return nil
}

func (c *converter) addPrioFilter(spec string) error {
	if val, err := strconv.ParseInt(spec, 10, 64); err == nil {
		c.logLevel = penlog.Prio(val)
		return nil
	}
	switch strings.ToLower(spec) {
	case "debug":
		c.logLevel = penlog.PrioDebug
	case "info":
		c.logLevel = penlog.PrioInfo
	case "notice":
		c.logLevel = penlog.PrioNotice
	case "warning":
		c.logLevel = penlog.PrioWarning
	case "error":
		c.logLevel = penlog.PrioError
	case "critical":
		c.logLevel = penlog.PrioCritical
	case "alert":
		c.logLevel = penlog.PrioAlert
	case "emergency":
		c.logLevel = penlog.PrioEmergency
	default:
		return fmt.Errorf("invalid loglevel '%s'", spec)
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

func fPrintError(w io.Writer, msg string) {
	line := createErrorRecord(msg)
	str, _ := json.Marshal(line)
	fmt.Fprintln(w, string(str))
}

func (c *converter) printError(msg string) {
	line := createErrorRecord(msg)
	str, _ := c.formatter.Format(line)
	fmt.Print(str)
}

func (c *converter) transform(r io.Reader) {
	var (
		err         error
		jsonLine    []byte
		reader      = bufio.NewReader(r)
		cursorReset = false
	)
	// ErrUnexpectedEOF occurs when reading a compressed file which is not yet
	// finalized. Let's just error out in this case.
	for !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		jsonLine, err = reader.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				c.printError(err.Error())
			}
			continue
		}
		var (
			data         map[string]interface{}
			deferredCont = false
		)
		if err := json.Unmarshal(jsonLine, &data); err != nil {
			c.printError(string(jsonLine))
			deferredCont = true
			// If there are workers avail, send
			// the error to them as well. The error
			// needs to be included in the logfiles
			// as well.
			data = createErrorRecord(string(jsonLine))
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
		if deferredCont {
			deferredCont = false
			continue
		}

		var (
			err error
			d   = copyData(data)
		)
		if c.stdoutFilter != nil {
			d, err = c.stdoutFilter.filter(d)
			if err != nil {
				c.printError(string(jsonLine))
				continue
			}
			if d == nil {
				continue
			}
		}

		var priority penlog.Prio

		if prio, ok := d["priority"]; ok {
			if p, ok := prio.(float64); ok {
				priority = penlog.Prio(p)
				if priority > c.logLevel {
					continue
				}
			}
		}
		if idRaw, ok := d["id"]; ok && c.id != "" {
			if id, ok := idRaw.(string); ok {
				if id != c.id {
					continue
				}
			}
		}
		if hrLine, err := c.formatter.Format(d); err == nil {
			if c.volatileInfo && isatty(uintptr(syscall.Stdout)) {
				// If the cursor has been reset, the line has to be cleared
				// before new content can be written
				if cursorReset {
					fmt.Print(clearLine)
				}
				fmt.Print(hrLine)
				// If in volatile info mode override infos in the same line
				if priority == penlog.PrioInfo {
					fmt.Print("\r")
					cursorReset = true
				} else {
					fmt.Println()
					cursorReset = false
				}
			} else {
				fmt.Println(hrLine)
			}
		} else {
			if errors.Is(err, errInvalidData) {
				c.printError(err.Error())
				continue
			}
			c.printError(string(jsonLine))
		}
	}
	if cursorReset {
		fmt.Println()
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

	encoder := json.NewEncoder(fileWriter)
	for line := range data {
		l, err := fil.filter(line)
		if l == nil || err != nil {
			continue
		}
		encoder.Encode(l)
	}

	fileWriter.Flush()
	if comp != nil {
		comp.Flush()
		comp.Close()
	}
	file.Close()
	wg.Done()
}

func main() {
	var (
		err           error
		filterSpecs   []string
		prioLevelRaw  string
		colorsCli     bool
		linesCli      bool
		stacktraceCli bool
		conv          = converter{
			formatter:   penlog.NewHRFormatter(),
			workers:     0,
			broadcastCh: make(chan map[string]interface{}),
			cleanedUp:   false,
		}
	)

	pflag.BoolVar(&colorsCli, "show-colors", true, "enable colorized output based on priorities")
	pflag.BoolVar(&linesCli, "show-lines", false, "show line numbers if available")
	pflag.BoolVar(&stacktraceCli, "show-stacktraces", false, "show stacktrace if available")
	pflag.BoolVar(&conv.formatter.ShowID, "show-ids", false, "show unique message id")
	pflag.BoolVar(&conv.formatter.ShowTags, "show-tags", false, "show penlog message tags")
	pflag.StringVarP(&conv.id, "id", "i", "", "only show this particular message")
	pflag.IntVarP(&conv.formatter.CompLen, "complen", "c", 8, "len of component field")
	pflag.IntVarP(&conv.formatter.TypeLen, "typelen", "t", 8, "len of type field")
	pflag.StringVarP(&prioLevelRaw, "priority", "p", "debug", "show messages with a lower priority level")
	pflag.StringArrayVarP(&filterSpecs, "filter", "f", []string{}, "write logs to a file with filters")
	pflag.BoolVar(&conv.volatileInfo, "volatile-info", false, "Overwrite info messages in the same line")
	version := pflag.BoolP("version", "V", false, "Show version and exit")
	cpuprofile := pflag.String("cpuprofile", "", "write cpu profile to `file`")
	pflag.Parse()

	if *version {
		fmt.Println(version)
		os.Exit(0)
	}

	conv.logFmt = "%s {%s} [%s]: %s"

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			colorEprintf(colorRed, conv.formatter.ShowColors, "could not create CPU profile: %s\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			colorEprintf(colorRed, conv.formatter.ShowColors, "could not start CPU profile: %s\n", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	if err := conv.addFilterSpecs(filterSpecs); err != nil {
		colorEprintf(colorRed, conv.formatter.ShowColors, "error: %s\n", err)
		os.Exit(1)
	}
	if err := conv.addPrioFilter(prioLevelRaw); err != nil {
		colorEprintf(colorRed, conv.formatter.ShowColors, "error: %s\n", err)
		os.Exit(1)
	}

	var (
		reader io.Reader = os.Stdin
		c                = make(chan os.Signal)
	)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-c
		exitCode := 1
		if s, ok := sig.(syscall.Signal); ok {
			exitCode = 128 + int(s)
		}
		conv.cleanup()
		os.Exit(exitCode)
	}()

	conv.formatter.ShowColors = colorsCli
	if colorsCli {
		if !isatty(uintptr(syscall.Stdout)) {
			conv.formatter.ShowColors = false
		}
		if helpers.GetEnvBool("PENLOG_FORCE_COLORS") {
			conv.formatter.ShowColors = colorsCli
		}
	}
	conv.formatter.ShowLines = linesCli
	if valRaw, ok := os.LookupEnv("PENLOG_SHOW_LINES"); ok {
		if val, err := strconv.ParseBool(valRaw); val && err == nil {
			conv.formatter.ShowLines = val
		}
	}
	conv.formatter.ShowStacktraces = stacktraceCli
	if valRaw, ok := os.LookupEnv("PENLOG_SHOW_STACKTRACES"); ok {
		if val, err := strconv.ParseBool(valRaw); val && err == nil {
			conv.formatter.ShowStacktraces = val
		}
	}

	if pflag.NArg() > 0 {
		for _, file := range pflag.Args() {
			reader, err = getReader(file)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			conv.transform(reader)
		}
	} else {
		conv.transform(reader)
	}
	conv.cleanup()
}
