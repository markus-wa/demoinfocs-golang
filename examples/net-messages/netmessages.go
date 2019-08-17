package main

import (
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"

	dem "github.com/markus-wa/demoinfocs-golang"
	ex "github.com/markus-wa/demoinfocs-golang/examples"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

// Run like this: go run netmessages.go -demo /path/to/demo.dem > out.png
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	// Configure parsing of BSPDecal net-message
	cfg := dem.DefaultParserConfig
	cfg.AdditionalNetMessageCreators = map[int]dem.NetMessageCreator{
		int(msg.SVC_Messages_svc_BSPDecal): func() proto.Message {
			return new(msg.CSVCMsg_BSPDecal)
		},
	}

	p := dem.NewParserWithConfig(f, cfg)

	// Register handler for BSPDecal messages
	p.RegisterNetMessageHandler(func(m *msg.CSVCMsg_BSPDecal) {
		fmt.Printf("bullet decal at x=%f y=%f z=%f\n", m.Pos.X, m.Pos.Y, m.Pos.Z)
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
