#!/bin/bash

set -e

bin/download-test-data.sh

go test ./...
