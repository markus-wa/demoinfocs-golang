#!/bin/bash

protoc --go_out=. \
       --go_opt=module=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg \
       -I=./proto ./proto/*.proto
