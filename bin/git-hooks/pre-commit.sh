#!/bin/bash

bin_dir=$(dirname "$0")/../../bin

if [[ "$(git diff --ignore-submodules)" != "" ]]; then
	echo 'stashing unstaged changes for checks'
	git stash -q --keep-index
	trap 'echo "unstashing unstaged changes" && git stash pop -q' EXIT
fi

echo -n 'checking interfaces ...'
if [[ "$(command -v ifacemaker)" != "" ]]; then
	INTERFACE_CHECK_OUTPUT=$($bin_dir/check-interfaces-generated.sh 2>&1)
	retVal=$?

	if [ $retVal -eq 0 ]; then
		echo ' OK'
	else
		echo -e "\n\n$INTERFACE_CHECK_OUTPUT\n"
		echo 'ERROR: interfaces not up-to-date or staged for commit. please run bin/generate-interfaces.sh'
		exit 1
	fi
else
	echo ' SKIPPED: ifacemaker not found on PATH'
fi

echo -n 'running build ...'
BUILD_OUTPUT=$($bin_dir/build.sh 2>&1)
retVal=$?

if [ $retVal -eq 0 ]; then
	echo ' OK'
else
	echo -e "\n\n$BUILD_OUTPUT\n"
	echo 'ERROR: build failed. please fix before committing.'
	exit 1
fi

echo -n 'running unit tests ...'
TEST_OUTPUT=$($bin_dir/unit-tests.sh 2>&1)
retVal=$?

if [ $retVal -eq 0 ]; then
	echo ' OK'
else
	echo -e "\n\n$TEST_OUTPUT\n"
	echo 'ERROR: unit tests failed. please fix before committing.'
	exit 1
fi
