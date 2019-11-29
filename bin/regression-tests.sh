#!/bin/bash

set -e

bin/download-test-data.sh default.7z unexpected_end_of_demo.7z regression-set.7z

go test -tags unassert_panic ./...
