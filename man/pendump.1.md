= pendump(1)
:doctype:    manpage
:man source: penlog

== Name

pendump - A simple tool for capturing network traffic

== Synopsis

----
pendump [ARGS]
----

== Description

pendump is a capture tool for network packets.
pendump uses libpcap internally, thus features known from tshark(1) or tcpdump(1) are supported.
In contrast to the prior art, this tool is much simpler as its only task is capturing traffic in pcap files.
Optionally a readiness notification mechanism (see the Environment Variables section) is supported.
Using this notification feature a parent process can synchronize its execution to pendump.
The indended use case for pendump is being used in scripts for capturing traffic.
For interactive usage tshark(1) or tcpdump(1) are superior.

== Arguments

`-f` capture_filter::
`--filter` capture_filter::
    Set the capture filter expression.
    This option can occur multiple times.
    For more information about the syntax and the rational, see pcap-filter(7).

`-i` interface::
`--iface` interface::
    Set the name of the network interface to use for live packet capture.
    If you're using Linux, `ip link` might be used to list interface names.

`-o` path::
`--out` path::
    A filepath where the pcapng data should be written to.
    If `path` ends with `.gz` the output is automatically gzipped.
    `-` can be used to indicate stdout.
    With `-` binary data is written to stdout, which is indended to be used with e.g. pipes.
    `pendump` overwrites existing files without question except if `path` is a named pipe (aka FIFO).
    FIFOs are opended and used as is.

== Permissions

As known from tshark(1), special permission are required to capture network packets.
On Linux root permissions can be avoided by adding the following capabilities to the pendump binary:

    # setcap cap_dac_override,cap_net_admin,cap_net_raw+eip ./pendump

Ideally, your Linux distribution has already done this job during building a package including pendump.

== Examples

The following example captures all traffic on tcp port 443 on interface wlan0 into the file `capture.pcap.gz.`
The file is compressed with the gzip algorithm.

    $ pendump -i wlan0 -f "tcp port 443" -o capture.pcap.gz

Remote capture is easy with invoking `pendump` via ssh and piping the data into wireshark.

    $ ssh HOST pendump -o - | wireshark -k -S -i -

== Environment Variables

READY_FD::
    This variable is can be used for a simple yet powerful syncronization feature.
    When set to an integer describing a pipe filedescriptor, a parent process is notified by pendump when it is ready with initialization work.
    The filedescriptor is closed (i.e. the parent process is syncronized) by the child process at the point where recording packets actually starts.
    The mechanism is described in detail on this https://michael.stapelberg.ch/posts/2020-02-02-readiness-notifications-in-golang[blog post].
    The only difference to the blog post is the name of the environment variable.

== See Also

penlog(7), penlog-pest-practice(7), pcap-filter(7), tshark(1), tcpdump(1)

include::footer.adoc[]
