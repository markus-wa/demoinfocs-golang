#!/bin/bash

set -e

# examples live in their own Go module (examples/go.mod) and are not covered by `go test ./...` in the root
scripts_dir=$(dirname "$0")
$scripts_dir/download-test-data.sh s2.7z valve_matchmaking.7z

cd "$scripts_dir/../examples"
go vet ./...
go test "$@" ./...
