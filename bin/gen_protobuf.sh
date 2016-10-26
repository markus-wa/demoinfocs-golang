#!/bin/sh
cd $GOPATH
msg_path=github.com/markus-wa/demoinfocs-golang/msg
protoc_args=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:
protoc -I=%${msg_path}/proto --gogofaster_out=${protoc_args}${msg_path} ${msg_path}/proto/*.proto
