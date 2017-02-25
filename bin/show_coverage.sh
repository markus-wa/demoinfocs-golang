#!/bin/sh
cd $(dirname $0)/..
go tool cover -html test/results/cover.prof
