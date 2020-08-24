= penlog-best-practice(7)
:doctype:    manpage
:man source: penlog

== Name

penlog-best-practice - practical advice for using penlog

== Description

The penlog(7) logging format provides a versatile toolset for generating JSON based logs.
This document serves as a guide in how to use the available metadata fields most efficiently.

== Components

The `component` field is often also referred to as a "software module".
Programs consisting of multiple modules, e.g. a protocol stack and higher level application code, can use the `component` field to distinguish between those.
If only one software module is used, this field can be omitted.
Do not use the `component` for describing anything but software modules; it will get confusing otherwise.

== Message Types

== Priorities

== Tags

== See Also

hr(1), penlog(7)

include::footer.adoc[]
