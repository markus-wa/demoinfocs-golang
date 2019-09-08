#!/bin/bash

# Windows needs special treatment
windows() { [[ -n "$WINDIR" ]]; }

# Cross-platform symlink function
link() {
	if windows; then
		# We need elevated privileges and some windows stuff
		powershell "Start-Process cmd -ArgumentList '/c','cd',(Get-Location).path,'&& mklink','${1//\//\\}','${2//\//\\}','|| cd && pause' -v RunAs"
	else
		# You know what? I think ln's parameters are backwards.
		ln -s "$2" "$1"
	fi
}

cd "$(dirname "$0")/../.."

if [ -f .git/hooks/pre-commit ]; then
	echo 'removing existing pre-commit hook'
	rm .git/hooks/pre-commit
fi
if [ -f .git/hooks/pre-push ]; then
	echo 'removing existing pre-push hook'
	rm .git/hooks/pre-push
fi

link .git/hooks/pre-commit ../../bin/git-hooks/pre-commit.sh
echo 'added pre-commit hook'

echo 'do you want to set up the pre-push hook for the regression suite?'
echo -n 'this will download ~ 1 GB of test-data (if not already done) [y/N] '

read prePushYesNo
if [[ "$prePushYesNo" == "y" || "$prePushYesNo" == "Y" ]]; then
	bin/download-test-data.sh default.7z unexpected_end_of_demo.7z regression-set.7z
	link .git/hooks/pre-push ../../bin/git-hooks/pre-push.sh
	echo 'added pre-push hook'
fi
