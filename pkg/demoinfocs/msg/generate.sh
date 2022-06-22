#!/bin/bash

protoc --go_out=. \
       --go_opt=module=github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg \
       -I=./proto ./proto/*.proto
