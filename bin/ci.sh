#!/bin/bash
# This script simulates a CI run.
# Might not work on windows since race tests are broken.
# Prerequisites must be installed manually:
# - ifacemaker -> GO111MODULE=off go get github.com/vburenin/ifacemaker
# - golangci-lint -> https://github.com/golangci/golangci-lint/releases
# - test-data -> pushd test/cs-demos && git lfs pull -I '*' && popd

set -e

# get the commands straight form the travis config
ci_commands=$(sed -n '/^script:/,/^\S/p' .travis.yml | grep -E '^  - ' | cut -c 5-)

while read -r cmd; do
	echo "running $cmd:"
	$cmd
	echo -e '\n'
done <<<"$ci_commands"
