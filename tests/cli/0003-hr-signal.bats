#!/usr/bin/env bats

data=""

setup() {
	data="$(< hr/example.log.json)"
}

@test "receive SIGINT" {
    skip "this test is still flaky and fails sometimes"
    echo "$data" | hr &
    kill -INT "$!"
}

@test "receive SIGTERM" {
    skip "this test is still flaky and fails sometimes"
    echo "$data" | hr &
    kill -TERM "$!"
}

@test "receive SIGQUIT" {
    skip "this test is still flaky and fails sometimes"
    echo "$data" | hr &
    kill -QUIT "$!"
}
