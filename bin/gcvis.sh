#!/bin/sh
# Run benchmark with gcvis
cd $(dirname $0)/..
binary=/tmp/demoinfocs-golang.test
go test -c -o ${binary}
gcvis ${binary} -test.run _NONE_ -test.bench . -test.benchtime 120s
