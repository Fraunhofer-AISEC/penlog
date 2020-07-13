#!/usr/bin/env bats

load lib-helpers

data=""

setup() {
	data="$(< example.log.json)"
    expected="$(< example-no-ipc.log)"
}

@test "jq with reading from stdin" {
	local expected
	out="$(hr -j '.|select(.component != "ipc")' < example.log.json)"
	expected="$(< example-no-ipc.log)"
	compstr "$out" "$expected"
}

@test "jq with reading from file" {
	local expected
	out="$(hr -j '.|select(.component != "ipc")' example.log.json)"
	expected="$(< example-no-ipc.log)"
	compstr "$out" "$expected"
}

@test "jq with invalid json input" {
	out="$(echo hans | hr -j '.')"
	compstr "$(echo $out | sed -e 's/.*: //')" "hans"
}

