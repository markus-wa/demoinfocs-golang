# demoinfocs-golang

Is a CS:GO demo parser written in Go based on [Valve's demoinfogo](https://github.com/ValveSoftware/csgo-demoinfo) and [SatsHelix's demoinfo](https://github.com/StatsHelix/demoinfo).

[![GoDoc](https://godoc.org/github.com/markus-wa/demoinfocs-golang?status.svg)](https://godoc.org/github.com/markus-wa/demoinfocs-golang)
[![Build Status](https://travis-ci.org/markus-wa/demoinfocs-golang.svg?branch=master)](https://travis-ci.org/markus-wa/demoinfocs-golang)
[![codecov](https://codecov.io/gh/markus-wa/demoinfocs-golang/branch/master/graph/badge.svg)](https://codecov.io/gh/markus-wa/demoinfocs-golang)
[![Go Report](https://goreportcard.com/badge/github.com/markus-wa/demoinfocs-golang)](https://goreportcard.com/report/github.com/markus-wa/demoinfocs-golang)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE.md)

## Discussions / Chat

Use gitter to ask questions and discuss ideas about this project.<br>
There are also [some other rooms](https://gitter.im/csgodemos) available around the topic of CS:GO demos.

[![Gitter chat](https://badges.gitter.im/csgodemos/demoinfo-lib.png)](https://gitter.im/csgodemos/demoinfo-lib)

## Go Get

	go get github.com/markus-wa/demoinfocs-golang

## Example

This is a simple example on how to use the library. After each round (on every `RoundEndedEvent`) it prints out which team won.

Check out the godoc of the `events` package for some more information about the available events and their purpose.

```go
import (
	"fmt"
	"log"
	"os"

	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

func main() {
	f, err := os.Open("path/to/demo.dem")
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}

	p := dem.NewParser(f, dem.WarnToStdErr)

	// Parse header
	h, err := p.ParseHeader()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Map:", h.MapName)

	// Register handler on round end to figure out who won
	p.RegisterEventHandler(func(e events.RoundEndedEvent) {
		switch e.Winner {
		case common.TeamTerrorists:
			fmt.Println("T-side won the round - score:", p.TState().Score()+1) // Score + 1 because it hasn't actually been updated yet
		case common.TeamCounterTerrorists:
			fmt.Println("CT-side won the round - score:", p.CTState().Score()+1)
		default:
			fmt.Println("Apparently neither the Ts nor CTs won the round, interesting")
		}
	})

	// Parse to end
	err = p.ParseToEnd()
	if err != nil {
		log.Fatal(err)
	}
}
```

## Development

### Running tests

To run tests [Git LFS](https://git-lfs.github.com) is required.

```sh
git submodule init
git submodule update
pushd test/cs-demos && git lfs pull && popd
go test
```

### Generating protobuf code

Should you need to re-generate the protobuf generated code in the `msg` package, you will need the following tools:

- The latest protobuf generator (`protoc`) from your package manager or https://github.com/google/protobuf/releases

- And `protoc-gen-gogofaster` from [gogoprotobuf](https://github.com/gogo/protobuf) to generate code for go.

		go get github.com/gogo/protobuf/protoc-gen-gogofaster

Make sure both are inside your `PATH` variable.

After installing these use `go generate ./msg` to generate the protobuf code.
