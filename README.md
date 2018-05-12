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

	go get -u github.com/markus-wa/demoinfocs-golang

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
		gs := p.GameState()
		switch e.Winner {
		case common.TeamTerrorists:
			// Winner's score + 1 because it hasn't actually been updated yet
			fmt.Printf("Round finished: winnerSide=T  ; score=%d:%d\n", gs.TState().Score()+1, gs.CTState().Score())
		case common.TeamCounterTerrorists:
			fmt.Printf("Round finished: winnerSide=CT ; score=%d:%d\n", gs.CTState().Score()+1, gs.TState().Score())
		default:
			// Probably match medic or something similar
			fmt.Println("Round finished: No winner (tie)")
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

### Debugging

You can use the build tag `debugdemoinfocs` (i.e. `go test -tags debugdemoinfocs -v`) to print out debugging information - such as game events or unhandled demo-messages - during the parsing process.<br>
Side-note: The tag isn't called `debug` to avoid naming conflicts with other libs (and underscores in tags don't work, apparently).

To change the default debugging behavior Go's `ldflags` paramter can be used. Example for additionally printing out the ingame-tick-numbers: `-ldflags '-X github.com/markus-wa/demoinfocs-golang.debugIngameTicks=YES'`

Check out `debug_on.go` for any other settings that can be changed.

### Generating protobuf code

Should you need to re-generate the protobuf generated code in the `msg` package, you will need the following tools:

- The latest protobuf generator (`protoc`) from your package manager or https://github.com/google/protobuf/releases

- And `protoc-gen-gogofaster` from [gogoprotobuf](https://github.com/gogo/protobuf) to generate code for go.

		go get -u github.com/gogo/protobuf/protoc-gen-gogofaster

[//]: # "The go get above needs two tabs so it's displayed as a) as part of the last list entry and b) as a code-block"
[//]: # "Oh and don't try to move these comments above it either"

Make sure both are inside your `PATH` variable.

After installing these use `go generate ./msg` to generate the protobuf code.
