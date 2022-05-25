= hr(1)
:doctype:    manpage
:man source: penlog

== Name

hr - Process the penlog logging format

== Synopsis

----
hr [ARGS] [FILE…]
----

== Description

`hr(1)` is used to process data in the `penlog(7)` format.
It converts data form the JSON format into the human readable format.
Data is tried to be always aligned in order to make the logs as readable as possible.
`hr(1)` provides filtering and archiving functionality.

With no `FILE` from stdin is read.
Multiple files are concatenated, similar to `cat(1)`.
However, `-` as a `FILE` is not supported.
If `FILE` has the file extension `.gz` (gzip) or `zst` (zstd) it is automatically decompressed.

== Arguments

`-c` int::
`--complen` int::
    The lenghth of the component field (default 8).

`-f` string::
`--filter` string::
    A filter expression using one of the following syntaxes:
    `file`, `type,…:file`, `component,…:type,…:file`.
    For `file` the same compression algorithms as for reading are supported.
    The first one saves the JSON data into `file`.
    The second one only writes messages of `type` into `file`.
    The third one only writes messages from `comonent` and `type` into `file`.
    Filters to stdout can be applied using the filename `-`.

`-i` string::
`--id` string::
    Only show messages with this unique id.

`-j` string::
`--jq` string::
    Pass this string as an argument to the jq(1) utility.
    `jq` is used as a preprocessor for the entire data; no file based filters are possible.
    This is equivalent to running `hr` in a pipeline: `zcat FILE | jq -c "FILTER" | hr`.
    The advantage is automatic decompression of archived files and easier typing.
    Be aware of dragons if your `jq` filter becomes too complex and alters the json data too much.

`-p` string::
`--priority` string::
    Only display messages with the priority < `string`.
    An integer or a string can be specified.
    The following strings are recognized: `debug`, `info`, `notice`, `warning`, `error`, `critical`, `alert`, `emergency`.
    This option only applies to the human readable output.

`--show-colors`::
    Enable or disable the colorization of output.

`--show-ids`::
    Enable or disable the output of optional unique message ids.

`--show-lines`::
    Enable or disable the output of optional linenumbers.

`--show-stacktraces`::
    Enable or disable the output of optional stacktraces.

`-s` string::
`--timespec` string::
    The golang timspec for the timestamp, default: `"Jan _2 15:04:05.000"`.

`--tiny`::
    Enable `hr-tiny` format (`component` and `type` are omitted).

`-t` int::
`--typelen` int::
    The lenghth of the type field (default 8).

== Examples

Read from stdin and only display debug messages:

    $ fancy-command | hr -f debug:-

Read from compressed file:

    $ hr log.json.zst

Archive testrun into multiple files; only show info on stdout:

    $ fancy-command | hr -f info:- -f error:errors.json.zst -f all.json.zst

== Environment Variables

hr(1) follows the recommendations described in penlog(7) for environment variables.
hr(1) understands these additional environment variables:

`PENLOG_FORCE_COLORS` (bool)::
    It is best practice to disable color escape codes when the relevant output streams are redirected to a file or a pipe.
    Setting thes evironmental variable enforces color escape codes.

`PENLOG_SHOW_LINES` (bool)::
    The display of line numbers can be enabled or disabled with this variable.

`PENLOG_SHOW_STACKTRACES` (bool)::
    The display of stacktraces can be enabled or disabled with this variable.

== See Also

penlog(7), penlog-pest-practice(7)

include::footer.adoc[]
