#!/usr/bin/env bats

load lib-helpers

data=""
expected=""
HRFLAGS=("--complen=8" "--typelen=7")

setup() {
	data="$(< hr/example.log.json)"
	expected="$(< hr/example.log)"
	data_color="$(< hr/example-colors.log.json)"
	expected_colors="$(< hr/example-colors.log)"
	expected_colors_stripped="$(< hr/example-colors-stripped.log)"
	expected_prio_debug="$(< hr/example-level-debug.log)"
	expected_prio_info="$(< hr/example-level-info.log)"
	expected_prio_notice="$(< hr/example-level-notice.log)"
	expected_prio_warning="$(< hr/example-level-warning.log)"
	expected_prio_error="$(< hr/example-level-error.log)"
	expected_prio_critical="$(< hr/example-level-critical.log)"
	expected_prio_alert="$(< hr/example-level-alert.log)"
	expected_prio_emergency="$(< hr/example-level-emergency.log)"
	data_with_error="$(< hr/example-with-error.log.json)"
}

@test "data from pipe to stdout" {
	local out
	out="$(echo "$data" | hr "${HRFLAGS[@]}")"
	compstr "$out" "$expected"
}

@test "data from file to stdout" {
	local out
	out="$(hr "${HRFLAGS[@]}" hr/example.log.json)"
	compstr "$out" "$expected"
}

@test "data from multiple files to stdout" {
	local out
	echo "$expected" > "$BATS_TMPDIR/expected"
	echo "$expected" >> "$BATS_TMPDIR/expected"
	out="$(hr "${HRFLAGS[@]}" hr/example.log.json hr/example.log.json)"
	compstr "$out" "$(< $BATS_TMPDIR/expected)"
	rm "$BATS_TMPDIR/expected"
}

@test "data from pipe to file without filters" {
	local out
	echo "$data" | hr "${HRFLAGS[@]}" -f "$BATS_TMPDIR/foo.log" > /dev/null
	out="$(cat $BATS_TMPDIR/foo.log)"
	compjson "$out" "$data"
	rm "$BATS_TMPDIR/foo.log"
}

@test "data from pipe to multiple files without filters" {
	local out
	echo "$data" | hr "${HRFLAGS[@]}" -f "$BATS_TMPDIR/foo.log" -f "$BATS_TMPDIR/foo2.log" > /dev/null
	out="$(cat $BATS_TMPDIR/foo.log)"
	out2="$(cat $BATS_TMPDIR/foo2.log)"
	compjson "$out" "$data"
	compjson "$out2" "$data"
	rm "$BATS_TMPDIR/foo2.log"
	rm "$BATS_TMPDIR/foo.log"
}

@test "data from pipe to compressed file" {
	local out
	echo "$data" | hr "${HRFLAGS[@]}" -f "$BATS_TMPDIR/foo.log.gz" > /dev/null
	out="$(zcat $BATS_TMPDIR/foo.log.gz)"
	compjson "$out" "$data"
	rm "$BATS_TMPDIR/foo.log.gz"
}

@test "data from file with priorities redirected to file" {
	local out
	hr "${HRFLAGS[@]}" "hr/example-colors.log.json" > "$BATS_TMPDIR/foo.log"
	out="$(cat $BATS_TMPDIR/foo.log)"
	compstr "$out" "$expected_colors_stripped"
	rm "$BATS_TMPDIR/foo.log"
}

@test "force colors using pipes" {
	local out
	out="$(PENLOG_FORCE_COLORS=1 hr --show-colors=true "${HRFLAGS[@]}" hr/example-colors.log.json)"
	compstr "$out" "$expected_colors"
}

@test "data from file with priority filter to stdout" {
	local out
	local HRFLAGS
	HRFLAGS=("--complen=8" "--typelen=8")

	out="$(hr -p debug "${HRFLAGS[@]}" hr/example-colors.log.json)"
	compstr "$out" "$expected_prio_debug"

	out="$(hr -p info "${HRFLAGS[@]}" hr/example-colors.log.json)"
	compstr "$out" "$expected_prio_info"

	out="$(hr -p notice "${HRFLAGS[@]}" hr/example-colors.log.json)"
	compstr "$out" "$expected_prio_notice"

	out="$(hr -p warning "${HRFLAGS[@]}" hr/example-colors.log.json)"
	compstr "$out" "$expected_prio_warning"

	out="$(hr -p error "${HRFLAGS[@]}" hr/example-colors.log.json)"
	compstr "$out" "$expected_prio_error"

	out="$(hr -p critical "${HRFLAGS[@]}" hr/example-colors.log.json)"
	compstr "$out" "$expected_prio_critical"

	out="$(hr -p alert "${HRFLAGS[@]}" hr/example-colors.log.json)"
	compstr "$out" "$expected_prio_alert"

	out="$(hr -p emergency "${HRFLAGS[@]}" hr/example-colors.log.json)"
	compstr "$out" "$expected_prio_emergency"
}

@test "pipe arbitrary data through hr" {
	local out

	out="$(echo hans | hr)"
	# Strip prefix, as the timestamp is not reproducible.
	compstr "$(echo $out | striptimestamp)" "hans"
}

@test "data with an error" {
    local out

	# Strip prefix, as the timestamp is not reproducible.
    out="$(hr hr/example-with-error.log.json | striptimestamp)"
    compstr "$out" "$(cat hr/expected-with-error.log | striptimestamp)"
}
