#!/bin/sh
cd $(dirname $0)/..
go test -run _NONE_ -bench . -benchtime 15s -cpuprofile test/results/cpu.prof
if [[ "$1" == "show" ]] ; then
	./bin/show_cpuprof.sh
else
	./bin/analyze_cpuprof.sh
fi
