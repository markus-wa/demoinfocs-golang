#!/bin/sh
cd $(dirname $0)/..
go tool pprof -web test/results/cpu.prof
