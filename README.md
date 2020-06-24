# PENLog

[![AUR package](https://img.shields.io/aur/version/penlog)](https://aur.archlinux.org/packages/penlog/)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/Fraunhofer-AISEC/penlog)

PENLog provides a specification, library, and tooling for simple machine readable logging.

## How does it work?

Log entries look like this:

``` txt
$ cat log.json
{"timestamp": "2020-04-02T12:48:08.906523", "component": "scanner", "type": "message", "data": "Starting tshark", "host": "kronos"}
{"timestamp": "2020-04-02T12:48:09.583521", "component": "moncay", "type": "message", "data": "Doing stuff", "host": "kronos"}
```

They can be converted with the included `hr` tool into this:

``` txt
$ hr log.json
Apr  2 12:48:08.906 {scanner } [message]: Starting tshark with
Apr  2 12:48:09.583 {moncay  } [message]: Doing stuff
```

## Why?

Long test runs generate tons of data.
This logging format enables powerful postprocessing **and** is nice to look at in the terminal as well.

## But JSON has so much overhead!!??

Just use the tooling like e.g. `hr -f file.log.zst`.
Much of the overhead is compressed away.
More examples are in the documentation.

## Where is the Specification?

The manpages are in `man/` in this repository.
They are written in the `asciidoc` markup language.

## How do I use it?

The converter is in `bin/hr` and can be build using:

```
$ make hr
```

For additional information, see the mapage `hr(1)` in the `man` directory.

The philosophy is: Let your program log everything at any time to stderr, pipe it into `hr` and let the tool do the filtering and archiving.
A Go and Python library for emitting log messages is included in this repository as well.
Usage is easy, e.g. in Go:

``` go
logger = penlog.NewLogger("component", os.Stderr)
logger.LogInfo("my log message")
```

## Special Features

penlog is a very simple yet powerful library.
For debugging, several helpers are available.
For instance, penlog can obtain line numbers and stacktraces.
A tool which uses penlog is [rtcp](https://git.sr.ht/~rumpelsepp/rtcp).

```
$ ./rtcp
Jun 24 16:56:46.379: started rumpelsepp's rtcp server
Jun 24 16:56:46.379: proxy terminated, did you provide a config?
```

```
$ PENLOG_CAPTURE_LINES=1 PENLOG_CAPTURE_STACKTRACES=1 ./rtcp
Jun 24 16:56:34.227: started rumpelsepp's rtcp server (/home/rumpelsepp/Projects/private/rtcp/main.go:153)
  |goroutine 1 [running]:
  |runtime/debug.Stack(0x6c9a40, 0xc00009cc90, 0x71de20)
  |     /usr/lib/go/src/runtime/debug/stack.go:24 +0x9d
  |github.com/Fraunhofer-AISEC/penlog.(*Logger).output(0xc0000f4000, 0xc00009cc90, 0x4)
  |     /home/rumpelsepp/go/pkg/mod/github.com/!fraunhofer-!a!i!s!e!c/penlog@v0.1.2-0.20200624142937-c5bc9405e8ab/log.go:253 +0x3b3
  |github.com/Fraunhofer-AISEC/penlog.(*Logger).logMessagef(0xc0000f4000, 0x71e696, 0x7, 0x6, 0x0, 0x0, 0x0, 0x7271ed, 0x20, 0x0, ...)
  |     /home/rumpelsepp/go/pkg/mod/github.com/!fraunhofer-!a!i!s!e!c/penlog@v0.1.2-0.20200624142937-c5bc9405e8ab/log.go:317 +0x26f
  |github.com/Fraunhofer-AISEC/penlog.(*Logger).LogInfof(...)
  |     /home/rumpelsepp/go/pkg/mod/github.com/!fraunhofer-!a!i!s!e!c/penlog@v0.1.2-0.20200624142937-c5bc9405e8ab/log.go:381
  |main.main()
  |     /home/rumpelsepp/Projects/private/rtcp/main.go:153 +0x63b
  |

Jun 24 16:56:34.228: proxy terminated, did you provide a config? (/home/rumpelsepp/Projects/private/rtcp/main.go:155)
  |goroutine 1 [running]:
  |runtime/debug.Stack(0x6c9a40, 0xc00009ccc0, 0x71de20)
  |     /usr/lib/go/src/runtime/debug/stack.go:24 +0x9d
  |github.com/Fraunhofer-AISEC/penlog.(*Logger).output(0xc0000f4000, 0xc00009ccc0, 0x4)
  |     /home/rumpelsepp/go/pkg/mod/github.com/!fraunhofer-!a!i!s!e!c/penlog@v0.1.2-0.20200624142937-c5bc9405e8ab/log.go:253 +0x3b3
  |github.com/Fraunhofer-AISEC/penlog.(*Logger).logMessage(0xc0000f4000, 0x71e696, 0x7, 0x3, 0x0, 0x0, 0x0, 0xc000105f38, 0x1, 0x1)
  |     /home/rumpelsepp/go/pkg/mod/github.com/!fraunhofer-!a!i!s!e!c/penlog@v0.1.2-0.20200624142937-c5bc9405e8ab/log.go:307 +0x24f
  |github.com/Fraunhofer-AISEC/penlog.(*Logger).LogError(...)
  |     /home/rumpelsepp/go/pkg/mod/github.com/!fraunhofer-!a!i!s!e!c/penlog@v0.1.2-0.20200624142937-c5bc9405e8ab/log.go:353
  |main.main()
  |     /home/rumpelsepp/Projects/private/rtcp/main.go:155 +0x6cd
  |

```
