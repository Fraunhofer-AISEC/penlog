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
	expected_prio_debug="$(cat example-level-debug.log)"
	expected_prio_info="$(cat example-level-info.log)"
	expected_prio_notice="$(cat example-level-notice.log)"
	expected_prio_warning="$(cat example-level-warning.log)"
	expected_prio_error="$(cat example-level-error.log)"
	expected_prio_critical="$(cat example-level-critical.log)"
	expected_prio_alert="$(cat example-level-alert.log)"
	expected_prio_emergency="$(cat example-level-emergency.log)"
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
	echo "$expected" > "$BATS_TMPDIR/expected"
	echo "$expected" >> "$BATS_TMPDIR/expected"
	out="$(hr "${HRFLAGS[@]}" example.log.json example.log.json)"
	compstr "$out" "$(cat $BATS_TMPDIR/expected)"
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
	hr "${HRFLAGS[@]}" example-colors.log.json > "$BATS_TMPDIR/foo.log"
	out="$(cat $BATS_TMPDIR/foo.log)"
	compstr "$out" "$expected_colors_stripped"
	rm "$BATS_TMPDIR/foo.log"
}

@test "data from file with priorities to stdout" {
	local out
	out="$(env PENLOG_FORCE_COLORS=1 hr --colors=true "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_colors"
}

@test "data from file with prioritiy filter to stdout" {
	local out
	local HRFLAGS
	HRFLAGS=("--complen=8" "--typelen=8")

	out="$(hr -p debug "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_prio_debug"

	out="$(hr -p info "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_prio_info"

	out="$(hr -p notice "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_prio_notice"

	out="$(hr -p warning "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_prio_warning"

	out="$(hr -p error "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_prio_error"

	out="$(hr -p critical "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_prio_critical"

	out="$(hr -p alert "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_prio_alert"

	out="$(hr -p emergency "${HRFLAGS[@]}" example-colors.log.json)"
	compstr "$out" "$expected_prio_emergency"
}

@test "pipe arbitrary data through hr" {
	local out

	out="$(echo hans | hr)"
	# Strip prefix, as the timestamp is not reproducible.
	compstr "$(echo $out | sed -e 's/.*: //')" "hans"
}
