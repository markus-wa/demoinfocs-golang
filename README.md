# demoinfocs-golang

Is a CS:GO demo parser written in Go based on [Valve's demoinfogo](https://github.com/ValveSoftware/csgo-demoinfo) and [SatsHelix's demoinfo](https://github.com/StatsHelix/demoinfo).

[![GoDoc](https://godoc.org/github.com/markus-wa/demoinfocs-golang?status.svg)](https://godoc.org/github.com/markus-wa/demoinfocs-golang)
[![Build Status](https://travis-ci.org/markus-wa/demoinfocs-golang.svg?branch=master)](https://travis-ci.org/markus-wa/demoinfocs-golang)
[![codecov](https://codecov.io/gh/markus-wa/demoinfocs-golang/branch/master/graph/badge.svg)](https://codecov.io/gh/markus-wa/demoinfocs-golang)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE.md)

## Go Get

	go get github.com/markus-wa/demoinfocs-golang

## Example

This is a simple example on how to use the library. It prints out the winner of each round after the warm up is over.

Check out the godoc on the events package for some more information about the available events and their purpose.

```go
import (
	"fmt"
	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	"log"
	"os"
)

func main() {
	f, err := os.Open("path/to/demo.dem")
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}

	p := dem.NewParser(f)

	// Parse header
	p.ParseHeader()

	// Get T / CT team state references
	tState := p.TState()
	ctState := p.CTState()

	// We need this to skip restarts and team switches before the match start
	// Might not be necessary for all demos (especially MM)
	// But for pro matches / scrims it might be depending on how the server was set up
	// TODO: This might not always be correct, needs testing
	matchStarted := false
	p.RegisterEventHandler(func(events.MatchStartedEvent) {
		matchStarted = true
	})
	matchReallyStarted := false
	p.RegisterEventHandler(func(events.RoundStartedEvent) {
		if matchStarted {
			matchReallyStarted = true
		}
	})

	// Register handler on round end to figure out who won
	p.RegisterEventHandler(func(e events.RoundEndedEvent) {
		if matchReallyStarted {
			if e.Winner == common.Team_Terrorists {
				fmt.Println("T-side won the round - score:", tState.Score()+1) // Score + 1 because it hasn't actually been updated yet
			} else if e.Winner == common.Team_CounterTerrorists {
				fmt.Println("CT-side won the round - score:", ctState.Score()+1)
			} else {
				fmt.Println("Apparently neither the Ts nor CTs won the round, interesting")
			}
		} else {
			fmt.Println("Skipping warmup event")
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
