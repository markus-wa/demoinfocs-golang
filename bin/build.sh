#!/bin/bash

set -e

# compile all packages + tests
go build ./...
go test -run ^$ ./...
