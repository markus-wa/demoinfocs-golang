#!/bin/sh
# Run test coverage
cd $(dirname $0)/..
packages=$(go list ./... | grep -v msg | awk -vORS=, '{ print $1 }' | sed 's/,$/\n/')
go test -run TestDemoInfoCs -coverprofile test/results/cover.prof -covermode count -coverpkg ${packages}
./bin/show_coverage.sh
