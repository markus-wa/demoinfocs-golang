package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"

	ex "github.com/markus-wa/demoinfocs-golang/v6/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
)

// Run like this: go run netmessages.go -demo /path/to/demo.dem > out.png
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)

	defer f.Close()

	// Configure parsing of BSPDecal net-message
	p := demoinfocs.NewParser(f, option.AdditionalNetMessageCreators(map[int]option.NetMessageCreator{
		int(msg.SVC_Messages_svc_BSPDecal): func() proto.Message {
			return new(msg.CSVCMsg_BSPDecal)
		},
	}))
	defer p.Close()

	// Register handler for BSPDecal messages
	p.RegisterNetMessageHandler(func(m *msg.CSVCMsg_BSPDecal) {
		fmt.Printf("bullet decal at x=%f y=%f z=%f\n", m.Pos.GetX(), m.Pos.GetY(), m.Pos.GetZ())
	})

	// Parse to end
	err = p.ParseToEnd(context.Background())
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
