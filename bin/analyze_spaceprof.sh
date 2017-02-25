#!/bin/sh
cd $(dirname $0)/..
go tool pprof -alloc_space test/results/mem.prof
