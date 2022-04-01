#!/bin/bash

protoc --go_out=. --go_opt=module=github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg \
--go-vtproto_out=. \
--go-vtproto_opt=features=unmarshal+size+pool \
--go-vtproto_opt=module=github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg \
--go-vtproto_opt=pool=github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg.CSVCMsg_PacketEntities \
--go-vtproto_opt=pool=github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg.CSVCMsg_GameEvent \
-I=./proto ./proto/*.proto
