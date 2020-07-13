#!/usr/bin/env bats

data=""

setup() {
	data="$(< example.log.json)"
}

@test "receive SIGINT" {
    echo "$data" | hr &
    kill -INT "$!"
}

@test "receive SIGTERM" {
    echo "$data" | hr &
    kill -TERM "$!"
}

@test "receive SIGQUIT" {
    echo "$data" | hr &
    kill -QUIT "$!"
}