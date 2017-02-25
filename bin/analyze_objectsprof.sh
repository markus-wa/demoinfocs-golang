#!/bin/sh
cd $(dirname $0)/..
go tool pprof -alloc_objects test/results/mem.prof
