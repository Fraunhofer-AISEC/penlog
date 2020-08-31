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

### Implementations

- go: 
- python:
