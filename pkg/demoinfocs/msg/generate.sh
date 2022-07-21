#!/bin/bash

protoc --proto_path=proto \
       --go_out=. \
       --go_opt=Mcstrike15_usermessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg \
       --go_opt=Mcstrike15_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg \
       --go_opt=Mengine_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg \
       --go_opt=Mnetmessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg \
       --go_opt=Msteammessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg \
       --go_opt=module=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg \
       cstrike15_gcmessages.proto cstrike15_usermessages.proto engine_gcmessages.proto netmessages.proto steammessages.proto
