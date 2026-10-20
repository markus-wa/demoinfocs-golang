#!/bin/bash

set -e

echo "downloading/updating '$@', this may take a while ..."

git submodule init
git submodule update

pushd test/cs-demos >/dev/null

minSize=1000000
for f in "$@"; do
  # extracted before, e.g. restored from the CI cache
  if [[ -f ".$f.extracted" ]]; then
    echo "$f: already extracted"
    continue
  fi

  fileSize=$(stat -c%s "$f")
  if (( $fileSize < $minSize)); then
	  git lfs pull -I $f
	  7z x $f -aoa
	  touch ".$f.extracted"
	fi
done

popd >/dev/null

echo 'download complete'
