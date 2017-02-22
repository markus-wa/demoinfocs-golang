@REM Generate protobuf code
cd %GOPATH%/src
set MSG_PATH=github.com/markus-wa/demoinfocs-golang/msg
set PROTOC_ARGS=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor
protoc -I=%MSG_PATH%/proto --gogofaster_out=%PROTOC_ARGS%:%MSG_PATH% %MSG_PATH%/proto/*.proto
cmd.exe