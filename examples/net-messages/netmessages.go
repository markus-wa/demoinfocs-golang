package main

import (
	"fmt"
	"os"

	ex "github.com/markus-wa/demoinfocs-golang/v2/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	msg "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg"
)

// Run like this: go run netmessages.go -demo /path/to/demo.dem > out.png
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	// Configure parsing of BSPDecal net-message
	cfg := demoinfocs.DefaultParserConfig
	cfg.AdditionalNetMessageCreators = map[int]demoinfocs.NetMessageCreator{
		int(msg.SVC_Messages_svc_BSPDecal): func() demoinfocs.VTProtobufMessage {
			return new(msg.CSVCMsg_BSPDecal)
		},
	}

	p := demoinfocs.NewParserWithConfig(f, cfg)
	defer p.Close()

	// Register handler for BSPDecal messages
	p.RegisterNetMessageHandler(func(m *msg.CSVCMsg_BSPDecal) {
		fmt.Printf("bullet decal at x=%f y=%f z=%f\n", m.Pos.GetX(), m.Pos.GetY(), m.Pos.GetZ())
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
