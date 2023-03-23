#!/bin/bash

protoc -Iproto \
       --go_out=. \
       --go_opt=Mcstrike15_usermessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       --go_opt=Mcstrike15_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       --go_opt=Mengine_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       --go_opt=Mnetmessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       --go_opt=Msteammessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       --go_opt=Mgcsdk_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       --go_opt=Mnetworkbasetypes.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       --go_opt=Mnetwork_connection.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       --go_opt=Mdemo.proto=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       --go_opt=module=github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2 \
       cstrike15_gcmessages.proto cstrike15_usermessages.proto engine_gcmessages.proto netmessages.proto steammessages.proto gcsdk_gcmessages.proto networkbasetypes.proto network_connection.proto demo.proto
