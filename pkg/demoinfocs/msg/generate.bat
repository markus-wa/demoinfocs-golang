FOR /f %%f IN ('dir /b proto\*.proto') DO protoc -I=proto --gogofaster_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:. %%f
