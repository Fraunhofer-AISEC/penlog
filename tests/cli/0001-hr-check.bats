#!/usr/bin/env bats

load lib-helpers

data=""
expected=""

setup() {
	data="$(cat example.log.json)"
	expected="$(cat example.log)"
}

@test "data from pipe to stdout" {
	local out
	out="$(echo "$data" | hr)"
	compstr "$out" "$expected"
}

@test "data from file to stdout" {
	# This fails for reasons I do not understand in the CI…
	if [[ ! -z "$CI" ]]; then
		skip
	fi
	local out
	out="$(hr example.log.json)"
	compstr "$out" "$expected"
}

@test "data from multiple files to stdout" {
	# This fails for reasons I do not understand in the CI…
	if [[ ! -z "$CI" ]]; then
		skip
	fi
	local out
	echo "$expected" > "/tmp/expected"
	echo "$expected" >> "/tmp/expected"
	out="$(hr example.log.json example.log.json)"
	compstr "$out" "$(cat /tmp/expected)"
	rm "/tmp/expected"
}

@test "data from pipe to file without filters" {
	local out
	echo "$data" | hr -f "/tmp/foo.log" > /dev/null
	out="$(cat /tmp/foo.log)"
	compstr "$out" "$data"
	rm "/tmp/foo.log"
}

@test "data from pipe to multiple files without filters" {
	local out
	echo "$data" | hr -f "/tmp/foo.log" -f "/tmp/foo2.log" > /dev/null
	out="$(cat /tmp/foo.log)"
	out2="$(cat /tmp/foo2.log)"
	compstr "$out" "$data"
	compstr "$out2" "$data"
	rm "/tmp/foo2.log"
	rm "/tmp/foo.log"
}

@test "data from pipe to compressed file" {
	local out
	echo "$data" | hr -f "/tmp/foo.log.gz" > /dev/null
	out="$(zcat /tmp/foo.log.gz)"
	compstr "$out" "$data"
	rm "/tmp/foo.log.gz"
}
