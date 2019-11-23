#!/bin/bash

set -e

go test -tags unassert_panic -short ./...
