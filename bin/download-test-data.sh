#!/bin/bash

set -e

echo 'downloading/updating test data (~ 1 GB), this may take a while ...'

git submodule init
git submodule update

pushd test/cs-demos >/dev/null
git lfs pull -I '*'
popd >/dev/null

echo 'download complete'
