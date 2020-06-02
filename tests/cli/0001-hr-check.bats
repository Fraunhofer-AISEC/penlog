#!/usr/bin/env bats

load lib-helpers

data=""
expected=""
HRFLAGS=("--complen=8" "--typelen=7")

setup() {
	data="$(cat example.log.json)"
	expected="$(cat example.log)"
	data_color="$(cat example-colors.log.json)"
	expected_colors="$(cat example-colors.log)"
	expected_colors_stripped="$(cat example-colors-stripped.log)"
}

@test "data from pipe to stdout" {
	local out
	out="$(echo "$data" | hr "${HRFLAGS[@]}")"
	compstr "$out" "$expected"
}

@test "data from file to stdout" {
	local out
	out="$(hr "${HRFLAGS[@]}" example.log.json)"
	compstr "$out" "$expected"
}

@test "data from multiple files to stdout" {
	local out
	echo "$expected" > "/tmp/expected"
	echo "$expected" >> "/tmp/expected"
	out="$(hr "${HRFLAGS[@]}" example.log.json example.log.json)"
	compstr "$out" "$(cat /tmp/expected)"
	rm "/tmp/expected"
}

@test "data from pipe to file without filters" {
	local out
	echo "$data" | hr "${HRFLAGS[@]}" -f "/tmp/foo.log" > /dev/null
	out="$(cat /tmp/foo.log)"
	compstr "$out" "$data"
	rm "/tmp/foo.log"
}

@test "data from pipe to multiple files without filters" {
	local out
	echo "$data" | hr "${HRFLAGS[@]}" -f "/tmp/foo.log" -f "/tmp/foo2.log" > /dev/null
	out="$(cat /tmp/foo.log)"
	out2="$(cat /tmp/foo2.log)"
	compstr "$out" "$data"
	compstr "$out2" "$data"
	rm "/tmp/foo2.log"
	rm "/tmp/foo.log"
}

@test "data from pipe to compressed file" {
	local out
	echo "$data" | hr "${HRFLAGS[@]}" -f "/tmp/foo.log.gz" > /dev/null
	out="$(zcat /tmp/foo.log.gz)"
	compstr "$out" "$data"
	rm "/tmp/foo.log.gz"
}

@test "data from file with priorities redirected to file" {
	local out
	hr "${HRFLAGS[@]}" example-colors.log.json > "/tmp/foo.log"
	out="$(cat /tmp/foo.log)"
	compstr "$out" "$expected_colors_stripped"
	rm "/tmp/foo.log"
}

@test "data from file with priorities to stdout" {
	local out
	out="$(env PENLOG_FORCE_COLORS=1 hr --colors=true "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_colors"
}