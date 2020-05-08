#!/bin/bash

set -e

echo "downloading/updating '$@', this may take a while ..."

git submodule init
git submodule update

pushd test/cs-demos >/dev/null

minSize=1000000
for f in "$@"; do
  fileSize=$(stat -c%s "$f")
  if (( $fileSize < $minSize)); then
	  git lfs pull -I $f
	  7z x $f -aoa
	fi
done

popd >/dev/null

echo 'download complete'
