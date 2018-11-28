package main

import (
	"fmt"
	"os"

	proto "github.com/gogo/protobuf/proto"

	dem "github.com/markus-wa/demoinfocs-golang"
	ex "github.com/markus-wa/demoinfocs-golang/examples"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

// Run like this: go run netmessages.go -demo /path/to/demo.dem > out.png
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	// Configure parsing of ConVar net-message (id=6)
	cfg := dem.DefaultParserConfig
	cfg.AdditionalNetMessageCreators = map[int]dem.NetMessageCreator{
		6: func() proto.Message {
			return new(msg.CNETMsg_SetConVar)
		},
	}

	p := dem.NewParserWithConfig(f, cfg)

	// Register handler for ConVar updates
	p.RegisterNetMessageHandler(func(m *msg.CNETMsg_SetConVar) {
		for _, cvar := range m.Convars.Cvars {
			fmt.Println(fmt.Sprintf("cvar %s=%s", cvar.Name, cvar.Value))
		}
	})

	// Parse to end
	err = p.ParseToEnd()
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
