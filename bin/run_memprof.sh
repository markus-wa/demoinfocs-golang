#!/bin/sh
cd $(dirname $0)/..
go test -run TestDemoInfoCs -memprofile test/results/mem.prof

if [[ "$1" == "show" || "$2" == "show" ]] ; then
	prefix=show
else
	prefix=analyze
fi

if [[ "$1" == "obj" ]] ; then
	./bin/${prefix}_objectsprof.sh
else
	./bin/${prefix}_spaceprof.sh
fi
