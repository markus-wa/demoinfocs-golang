#!/bin/sh
cd $(dirname $0)/..
if [[ -n "$1" ]] ; then
	bt="-benchtime $1"
fi
go test -run _NONE_ -bench . ${bt}
