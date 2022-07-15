package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	dem "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/events"
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	fl := new(flag.FlagSet)

	demPathPtr := fl.String("demo", "", "Demo file `path`")
	infoPathPtr := fl.String("info", "", "Info file `path`")

	err := fl.Parse(os.Args[1:])
	checkErr(err)

	demPath := *demPathPtr
	infoPath := *infoPathPtr

	infoF, err := os.Open(infoPath)
	checkErr(err)

	b, err := ioutil.ReadAll(infoF)
	checkErr(err)

	k, err := dem.MatchInfoDecryptionKey(b)
	checkErr(err)

	f, err := os.Open(demPath)
	checkErr(err)

	defer f.Close()

	cfg := dem.DefaultParserConfig
	cfg.NetMessageDecryptionKey = k

	p := dem.NewParserWithConfig(f, cfg)

	p.RegisterEventHandler(func(warn events.ParserWarn) {
		log.Println("WARNING:", warn.Message)
	})

	p.RegisterEventHandler(func(message events.ChatMessage) {
		log.Println(message)
	})

	err = p.ParseToEnd()
	checkErr(err)
}
