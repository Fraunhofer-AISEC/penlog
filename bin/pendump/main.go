package main

import (
	"compress/gzip"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/Fraunhofer-AISEC/penlog"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	"github.com/spf13/pflag"
)

var logger = penlog.NewLogger("pendump", os.Stderr)

type runtimeOptions struct {
	outfile      string
	iface        string
	filter       string
	promiscuous  bool
	timeout      time.Duration
	cleanupDelay time.Duration
	snaplen      uint32
}

type dumper struct {
	handle  *pcap.Handle
	writer  *pcapgo.Writer
	gzipper *gzip.Writer
	file    *os.File
}

func (d *dumper) run() {
	pkgsrc := gopacket.NewPacketSource(d.handle, d.handle.LinkType())

	for packet := range pkgsrc.Packets() {
		if err := d.writer.WritePacket(packet.Metadata().CaptureInfo, packet.Data()); err != nil {
			logger.LogCritical(err)
			break
		}
	}

	if d.gzipper != nil {
		d.gzipper.Flush()
		d.gzipper.Close()
	}
	d.file.Close()
}

func main() {
	opts := runtimeOptions{}
	pflag.StringVarP(&opts.iface, "iface", "i", "lo", "interface to capture")
	pflag.StringVarP(&opts.outfile, "out", "o", "dump.pcap.gz", "specifies output file")
	pflag.StringVarP(&opts.filter, "filter", "f", "", "set bpf capture filter")
	pflag.BoolVarP(&opts.promiscuous, "promiscuous", "p", true, "enable promiscuous on the interface")
	pflag.DurationVarP(&opts.timeout, "timeout", "t", 1*time.Second, "set pcap timeout value (expert setting)")
	pflag.DurationVarP(&opts.cleanupDelay, "delay", "d", 2*time.Second, "wait this amount of seconds after termination signal")
	pflag.Uint32VarP(&opts.snaplen, "snaplen", "s", 65535, "set pcap saplen value (expert setting)")
	pflag.Parse()

	var (
		readyFile  *os.File
		outfile    *os.File
		gzipWriter *gzip.Writer
		outWriter  io.Writer
		err        error
	)

	if fd, err := strconv.Atoi(os.Getenv("READY_FD")); err == nil {
		readyFile = os.NewFile(uintptr(fd), "readyfd")
		os.Unsetenv("READY_FD")
	}
	if opts.outfile == "-" {
		outfile = os.Stdout
	} else {
		if s, err := os.Stat(opts.outfile); err == nil && s.Mode()&os.ModeNamedPipe != 0 {
			outfile, err = os.Open(opts.outfile)
			if err != nil {
				logger.LogCritical(err)
				os.Exit(1)
			}
		} else {
			outfile, err = os.Create(opts.outfile)
			if err != nil {
				logger.LogCritical(err)
				os.Exit(1)
			}
		}
	}

	handle, err := pcap.OpenLive(opts.iface, int32(opts.snaplen), opts.promiscuous, opts.timeout)
	if err != nil {
		logger.LogCritical(err)
		logger.LogInfo("if it is a permission problem, see the documentation")
		os.Exit(1)
	}

	if opts.filter != "" {
		if err := handle.SetBPFFilter(opts.filter); err != nil {
			logger.LogCritical(err)
			os.Exit(1)
		}
	}

	if filepath.Ext(outfile.Name()) == ".gz" {
		gzipWriter = gzip.NewWriter(outfile)
		outWriter = gzipWriter
	} else {
		outWriter = outfile
	}

	pcapw := pcapgo.NewWriter(outWriter)
	pcapw.WriteFileHeader(opts.snaplen, handle.LinkType())
	if err != nil {
		logger.LogCritical(err)
		os.Exit(1)
	}

	logger.LogDebugf("capturing interface: '%s'", opts.iface)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		time.Sleep(opts.cleanupDelay)
		handle.Close()
	}()

	if readyFile != nil {
		logger.LogDebug("initializing done; signaling readiness")
		readyFile.Close()
	}

	d := dumper{
		handle:  handle,
		gzipper: gzipWriter,
		writer:  pcapw,
		file:    outfile,
	}

	d.run()
}
