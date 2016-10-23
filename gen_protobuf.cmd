cd %GOPATH%/src
set MSG_PATH=github.com/markus-wa/demoinfocs-golang/msg
protoc -I=%MSG_PATH%/proto --gogofaster_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:%MSG_PATH% %MSG_PATH%/proto/*.proto
cmd.exe