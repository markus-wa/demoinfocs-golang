#!/bin/bash

set -e

# default reference/baseline is master
if [[ "$base_rev" == "" ]]; then
	base_rev='origin/master'
fi

echo "Linting changes between/since $base_rev"

golangci-lint run --new-from-rev $base_rev | reviewdog -f=golangci-lint -diff="git diff $base_rev" -reporter="${REVIEWDOG_REPORTER:-local}"
