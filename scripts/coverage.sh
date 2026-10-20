#!/bin/bash

set -e

# usage: coverage.sh [output file] [extra go test args, e.g. -run / -skip]
# output file defaults to 'coverage.txt' (the name Codecov looks for)
out=${1:-coverage.txt}
shift || true

scripts_dir=$(dirname "$0")
$scripts_dir/download-test-data.sh s2.7z deathmatch.7z

# don't cover test helpers and generated protobuf code
coverpkg_ignore='/(msg|internal/stfake|internal/stbridge|internal/fuzzcheck)'
coverpkg=$(go list ./... | grep -v -E ${coverpkg_ignore} | awk -vORS=, '{ print $1 }' | sed 's/,$/\n/')

# -timeout 30m because the CI is slow
go test -v -timeout 30m -coverprofile="$out" -coverpkg=$coverpkg -tags unassert_panic "$@" ./...
