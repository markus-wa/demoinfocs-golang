#!/bin/bash

bin_dir=$(dirname "$0")/../../bin

if [[ "$(git diff --ignore-submodules)" != "" ]]; then
	echo 'stashing unstaged changes for checks'
	git stash -q --keep-index
	trap 'echo "unstashing unstaged changes" && git stash pop -q' EXIT
fi

echo 'running regression tests'
TEST_OUTPUT=$($bin_dir/regression-tests.sh 2>&1)
retVal=$?

if [ $retVal -eq 0 ]; then
	echo ' OK'
else
	echo -e "\n\n$TEST_OUTPUT\n"
	echo 'ERROR: regression tests failed. please fix before pushing.'
	exit 1
fi
