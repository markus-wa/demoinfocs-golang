#!/bin/bash

set -e

# -short to skip long running regression tests
# -timeout 15m for CI, which is quite slow
go test -v -race -short ./... -timeout 15m

# run TestDemoInfoCs which is skipped by -short
# so we at least check one demo with race tests
scripts_dir=$(dirname "$0")
$scripts_dir/download-test-data.sh default.7z
go test -v -race -run TestDemoInfoCs ./pkg/demoinfocs -timeout 15m
