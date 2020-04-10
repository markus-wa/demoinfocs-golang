#!/bin/bash

go test -benchtime 10s -benchmem -cpuprofile cpu.out -memprofile mem.out -run NONE -bench . ./pkg/demoinfocs -concurrentdemos 8

go tool pprof -web cpu.out
