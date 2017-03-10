// Package msg contains the generated protobuf demo message code.
// Use 'go generate' to generate the code from the .proto files inside the proto sub directory.
package msg

//go:generate protoc -I=proto --gogofaster_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:. proto/*.proto
