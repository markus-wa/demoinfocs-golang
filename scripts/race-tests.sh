#!/bin/bash

set -e

scripts_dir=$(dirname "$0")

# a few non-short tests which are skipped by -short,
# so we at least check the main demo and the polymorphic rules path with race tests
$scripts_dir/download-test-data.sh s2.7z deathmatch.7z

# Both runs in parallel: the -short run (skips long running regression tests) in the background,
# the demo tests in the foreground. Their output is kept apart and printed once both are done.
# -timeout 15m for CI, which is quite slow
short_log=$(mktemp)
go test -v -race -short ./... -timeout 15m > "$short_log" 2>&1 &
short_pid=$!

demo_status=0
go test -v -race -run 'TestDemoInfoCs|TestPolymorphicGameModeRules' ./pkg/demoinfocs -timeout 15m || demo_status=$?

short_status=0
wait $short_pid || short_status=$?

echo "--- go test -race -short ./... (exit $short_status)"
cat "$short_log"
rm -f "$short_log"

if (( short_status != 0 || demo_status != 0 )); then
	echo "race tests failed: -short run exit $short_status, demo run exit $demo_status"
	exit 1
fi
