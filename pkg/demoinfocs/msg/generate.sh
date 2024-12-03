#!/bin/bash

# sed -i '1i\package com.github.markus_wa.demoinfocs_golang.s2;\n' proto/s2/*.proto
# sed -i 's/ \.(?!google)/ com.github.markus_wa.demoinfocs_golang.s2./g' proto/s2/*.proto

protoc -Iproto \
       --go_out=. \
       --go_opt=Ms2/cstrike15_usermessages.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/cstrike15_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/engine_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/netmessages.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/steammessages.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/gcsdk_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/networkbasetypes.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/network_connection.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/demo.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/gameevents.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/usermessages.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/cs_gameevents.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=Ms2/te.proto=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       --go_opt=module=github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg \
       s2/cstrike15_gcmessages.proto \
       s2/cstrike15_usermessages.proto \
       s2/engine_gcmessages.proto \
       s2/netmessages.proto \
       s2/steammessages.proto \
       s2/gcsdk_gcmessages.proto \
       s2/networkbasetypes.proto \
       s2/network_connection.proto \
       s2/demo.proto \
       s2/gameevents.proto \
       s2/usermessages.proto \
       s2/cs_gameevents.proto \
       s2/te.proto
