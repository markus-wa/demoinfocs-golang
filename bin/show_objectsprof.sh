#!/bin/sh
cd $(dirname $0)/..
go tool pprof -alloc_objects -web test/results/mem.prof
