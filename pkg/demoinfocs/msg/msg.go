// +build !windows

package msg

//go:generate protoc -I=proto --gogofaster_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:. proto/*.proto
