#!/usr/bin/env bats

load lib-helpers

data=""

setup() {
	data="$(cat example.log.json)"
}

@test "filter component" {
	local expected
	out="$(echo "$data" | hr -f abcd::-)"
	expected="$(cat example-abcd.log)"
	compstr "$out" "$expected"
}

@test "filter message types" {
	local expected
	out="$(echo "$data" | hr -f read,write:-)"
	expected="$(cat example-read-write.log)"
	compstr "$out" "$expected"
}

@test "filter component and types" {
	local expected
	out="$(echo "$data" | hr -f abcd:read,write:-)"
	expected="$(cat example-abcd-read-write.log)"
	compstr "$out" "$expected"
}
