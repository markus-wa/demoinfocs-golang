#!/bin/sh
# This should work for generating protobufs on Windows (Cygwin / MinGw) as well as Linux & Mac.
# Why such a hassle?
# With just the protoc command in go:generate Windows works but Linux won't find the protos.
# With 'sh -c <protoc cmd>' wrapped around it Linux works but Windows won't map the descriptor.

unameOut="$(uname -s)"
case "${unameOut}" in
    Linux*)   machine=Linux;;
    Darwin*)  machine=Mac;;
    CYGWIN*)  ;&
    MINGW*)   machine=Windows;;
    *)        machine="UNKNOWN:${unameOut}"
esac

if [ "${machine}" = "Windows" ]; then
	cmd <<< 'protoc -I=proto --gogofaster_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:. proto/*.proto'
else
	protoc -I=proto --gogofaster_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:. proto/*.proto
fi