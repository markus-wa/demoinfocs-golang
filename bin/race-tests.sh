#!/bin/bash

set -e

# -short to skip long running regression tests
# -timeout 15m for CI, which is quite slow
go test -v -race -short ./... -timeout 15m

# run TestDemoInfoCs which is skipped by -short
# so we at least check one demo with race tests
go test -v -race -run TestDemoInfoCs . -timeout 15m
