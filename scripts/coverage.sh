#!/bin/bash

set -e

scripts_dir=$(dirname "$0")
$scripts_dir/download-test-data.sh s2.7z

# don't cover mocks and generated protobuf code
coverpkg_ignore='/(fake|msg)'
coverpkg=$(go list ./... | grep -v -E ${coverpkg_ignore} | awk -vORS=, '{ print $1 }' | sed 's/,$/\n/')

# -timeout 30m because the CI is slow
# output file must be called 'coverage.txt' for Codecov
go test -v -timeout 30m -coverprofile=coverage.txt -coverpkg=$coverpkg -tags unassert_panic ./...
