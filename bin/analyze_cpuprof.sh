#!/bin/sh
cd $(dirname $0)/..
go tool pprof test/results/cpu.prof
