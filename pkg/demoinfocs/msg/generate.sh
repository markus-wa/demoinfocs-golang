#!/bin/bash

# The protos live in proto/v6 with package com.github.markus_wa.demoinfocs_golang.v6 and without Valve's custom
# options, so they don't clash with demoinfocs v4's protos when a program imports both versions
# (e.g. CS Demo Analyzer). After updating them from upstream, run ./prepare-protos.py first.

protoc -Iproto \
       --go_out=. \
       --go_opt=Mv6/cs_usercmd.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/usercmd.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/cstrike15_usermessages.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/cstrike15_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/engine_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/netmessages.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/steammessages.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/gcsdk_gcmessages.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/networkbasetypes.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/network_connection.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/demo.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/gameevents.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/usermessages.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/cs_gameevents.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/te.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/source2_steam_stats.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=Mv6/valveextensions.proto=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       --go_opt=module=github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg \
       v6/cs_usercmd.proto \
       v6/cstrike15_gcmessages.proto \
       v6/cstrike15_usermessages.proto \
       v6/engine_gcmessages.proto \
       v6/netmessages.proto \
       v6/steammessages.proto \
       v6/gcsdk_gcmessages.proto \
       v6/networkbasetypes.proto \
       v6/network_connection.proto \
       v6/demo.proto \
       v6/gameevents.proto \
       v6/usermessages.proto \
       v6/cs_gameevents.proto \
       v6/te.proto \
       v6/source2_steam_stats.proto \
       v6/usercmd.proto \
       v6/valveextensions.proto
