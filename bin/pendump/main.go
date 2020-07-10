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

var logger = penlog.NewLogger("penlog", os.Stderr)

type runtimeOptions struct {
	outfile     string
	iface       string
	filter      string
	promiscuous bool
	timeout     time.Duration
	snaplen     int
}

type dumper struct {
	handle  *pcap.Handle
	writer  *pcapgo.NgWriter
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

	d.writer.Flush()
	if d.gzipper != nil {
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
	pflag.IntVarP(&opts.snaplen, "snaplen", "s", 1600, "set pcap saplen value (expert setting)")
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
	outfile, err = os.Create(opts.outfile)
	if err != nil {
		logger.LogCritical(err)
		os.Exit(1)
	}

	handle, err := pcap.OpenLive(opts.iface, int32(opts.snaplen), opts.promiscuous, opts.timeout)
	if err != nil {
		logger.LogCritical(err)
		logger.LogInfo("if it is a permission problem, try:")
		logger.LogInfo("sudo setcap cap_dac_override,cap_net_admin,cap_net_raw+eip ./pendump")
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

	pcapw, err := pcapgo.NewNgWriter(outWriter, handle.LinkType())
	if err != nil {
		logger.LogCritical(err)
		os.Exit(1)
	}

	logger.LogInfof("capturing interface: '%s'", opts.iface)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
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
