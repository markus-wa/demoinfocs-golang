#!/bin/sh
cd $(dirname $0)/..
go tool pprof -alloc_space -web test/results/mem.prof
