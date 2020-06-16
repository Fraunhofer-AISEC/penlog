#!/usr/bin/env bats

load lib-helpers

data=""
HRFLAGS=("--complen=8" "--typelen=7")

setup() {
	data="$(cat example.log.json)"
}

@test "filter component" {
	local expected
	out="$(echo "$data" | hr "${HRFLAGS[@]}" -f abcd::-)"
	expected="$(cat example-abcd.log)"
	compstr "$out" "$expected"
}

@test "filter message types" {
	local expected
	out="$(echo "$data" | hr "${HRFLAGS[@]}" -f read,write:-)"
	expected="$(cat example-read-write.log)"
	compstr "$out" "$expected"
}

@test "filter component and types" {
	local expected
	out="$(echo "$data" | hr "${HRFLAGS[@]}" -f abcd:read,write:-)"
	expected="$(cat example-abcd-read-write.log)"
	compstr "$out" "$expected"
}

@test "error logs in archived file" {
	local out
	echo "hans" | hr -f "$BATS_TMPDIR/foo.log"
	out="$(jq -r ".data" < "$BATS_TMPDIR/foo.log")"
	compstr "$out" "hans"
	rm "$BATS_TMPDIR/foo.log"
}

@test "error logs in archived files" {
	local out
	echo "hans" | hr -f "$BATS_TMPDIR/foo.log" -f "$BATS_TMPDIR/foo1.log"

	out="$(jq -r ".data" < "$BATS_TMPDIR/foo.log")"
	compstr "$out" "hans"
	out="$(jq -r ".data" < "$BATS_TMPDIR/foo1.log")"
	compstr "$out" "hans"

	rm "$BATS_TMPDIR/foo.log"
	rm "$BATS_TMPDIR/foo1.log"
}
