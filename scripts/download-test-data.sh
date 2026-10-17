#!/bin/bash

set -e

echo "downloading/updating '$@', this may take a while ..."

git submodule init
git submodule update

pushd test/cs-demos >/dev/null

# p7zip (apt, older brew) provides 7z, brew's sevenzip only 7zz
sevenZip=7z
if ! command -v $sevenZip >/dev/null; then
  sevenZip=7zz
fi

minSize=1000000
for f in "$@"; do
  fileSize=$(wc -c < "$f")
  if (( fileSize < minSize )); then
	  git lfs pull -I "$f"
	  $sevenZip x "$f" -aoa
	fi
done

popd >/dev/null

echo 'download complete'
