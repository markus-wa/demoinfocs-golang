#!/bin/bash

set -e

bin_dir=$(dirname "$0")

$bin_dir/generate-interfaces.sh
diff_output=$(git diff --ignore-submodules)
if [[ "$diff_output" != "" ]]; then
	# don't keep the changes used for the check
	git stash save --keep-index --quiet
	git stash drop --quiet

	echo "ERROR: generated code is not up-to-date"
	echo "$diff_output"
	exit 1
fi
