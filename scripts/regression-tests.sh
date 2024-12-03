#!/bin/bash

set -e

scripts_dir=$(dirname "$0")
$scripts_dir/download-test-data.sh s2.7z

go test -tags unassert_panic ./...
