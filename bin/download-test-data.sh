#!/bin/bash

set -e

echo "downloading/updating '$@', this may take a while ..."

git submodule init
git submodule update

pushd test/cs-demos >/dev/null

for f in "$@"; do
	git lfs pull -I $f
	7z x $f -aoa
done

popd >/dev/null

echo 'download complete'
