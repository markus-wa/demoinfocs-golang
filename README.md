# demoinfocs-golang - A CSGO Demo Parser

Is a Go library for super fast parsing and analysing of Counter Strike: Global Offensive (CSGO) demos (aka replays). It is based on [Valve's demoinfogo](https://github.com/ValveSoftware/csgo-demoinfo) and [SatsHelix's demoinfo](https://github.com/StatsHelix/demoinfo).

[![GoDoc](https://godoc.org/github.com/markus-wa/demoinfocs-golang?status.svg)](https://godoc.org/github.com/markus-wa/demoinfocs-golang)
[![Build Status](https://travis-ci.org/markus-wa/demoinfocs-golang.svg?branch=master)](https://travis-ci.org/markus-wa/demoinfocs-golang)
[![codecov](https://codecov.io/gh/markus-wa/demoinfocs-golang/branch/master/graph/badge.svg)](https://codecov.io/gh/markus-wa/demoinfocs-golang)
[![Go Report](https://goreportcard.com/badge/github.com/markus-wa/demoinfocs-golang)](https://goreportcard.com/report/github.com/markus-wa/demoinfocs-golang)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE.md)

## Discussions / Chat

You can use gitter to ask questions and discuss ideas about this project.

[![Gitter chat](https://badges.gitter.im/csgodemos/demoinfo-lib.png)](https://gitter.im/csgodemos/demoinfo-lib)

## Requirements

This library is intended to be used with `go 1.11` or higher as it is built using Go modules.

It's recommended to use modules for consumers as well if possible.
If you are unfamiliar with Go modules there's a [list of recommended resources](https://github.com/markus-wa/demoinfocs-golang/wiki/Go-Modules#recommended-links--articles) in the wiki.

## Go Get

	go get -u github.com/markus-wa/demoinfocs-golang@v1.0.0

	# For non-module projects / GOPATH (not recommended)
	go get -u github.com/markus-wa/demoinfocs-golang

## Example

This is a simple example on how to handle game events using this library.
It prints all kills in a given demo (killer, weapon, victim, was it a wallbang/headshot?) by registering a handler for [`events.Kill`](https://godoc.org/github.com/markus-wa/demoinfocs-golang/events#Kill).

Check out the [godoc of the `events` package](https://godoc.org/github.com/markus-wa/demoinfocs-golang/events) for some information about the other available events and their purpose.

```go
package main

import (
	"fmt"
	"os"

	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

// Run like this: go run print_kills.go
func main() {
	f, err := os.Open("/path/to/demo.dem")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	p := dem.NewParser(f)

	// Register handler on kill events
	p.RegisterEventHandler(func(e events.Kill) {
		var hs string
		if e.IsHeadshot {
			hs = " (HS)"
		}
		var wallBang string
		if e.PenetratedObjects > 0 {
			wallBang = " (WB)"
		}
		fmt.Printf("%s <%v%s%s> %s\n", e.Killer.Name, e.Weapon.Weapon, hs, wallBang, e.Victim.Name)
	})

	// Parse to end
	err = p.ParseToEnd()
	if err != nil {
		panic(err)
	}
}
```

### Sample output

Running the code above will print something like this:

```
xms <AK-47 (HS)> crisby
tiziaN <USP-S (HS)> Ex6TenZ
tiziaN <USP-S> mistou
tiziaN <USP-S (HS)> ALEX
xms <Glock-18 (HS)> tiziaN
...
keev <AWP (HS) (WB)> to1nou
...
```

### More examples

Check out the [examples](examples) folder for more examples, like [how to generate heatmaps](examples/heatmap) like this one:

![Example heatmap](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/master/examples/heatmap/heatmap.jpg)

## Features

* Game events (kills, shots, round starts/ends, footsteps etc.) - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang/events) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/print-events)
* Tracking of game-state (players, teams, grenades etc.) - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang#GameState)
* Grenade projectiles / trajectories - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang#GameState.GrenadeProjectiles) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/nade-trajectories)
* Access to entities, server-classes & data-tables - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang/sendtables#ServerClasses) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/entities)
* Access to all net-messages - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang#NetMessageCreator) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/net-messages)
* Chat & console messages <sup id="achat1">1</sup> - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang/events#ChatMessage) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/print-events)
* [Easy debugging via build-flags](#debugging)
* Built with performance & concurrency in mind

1. <small id="f1">Only for some demos; in MM demos the chat is encrypted for example.</small>

## Performance / Benchmarks

Two of the top priorities of this parser are performance and concurrency.

Here are some benchmark results from a system with a Intel i7 2600k CPU and SSD disk running Windows 10 and a demo with 85'000 frames.

### Overview

|Benchmark|Description|Average Duration|Speed|
|-|-|-|-|
|`BenchmarkConcurrent`|Read and parse 8 demos concurrently|2.90 s (per 8 demos)|~234'000 ticks / s|
|`BenchmarkDemoInfoCs`|Read demo from drive and parse|1.39 s|~61'000 ticks / s
|`BenchmarkInMemory`|Read demo from memory and parse|1.38 s|~61'000 ticks / s

### Raw output

```
$ go test -run _NONE_ -bench . -benchtime 30s -benchmem -concurrentdemos 8
goos: windows
goarch: amd64
pkg: github.com/markus-wa/demoinfocs-golang
BenchmarkDemoInfoCs-8                 30        1397398190 ns/op        162254528 B/op    839779 allocs/op
BenchmarkInMemory-8                   30        1384877250 ns/op        162109924 B/op    839628 allocs/op
BenchmarkConcurrent-8                 20        2902574295 ns/op        1297042534 B/op  6717163 allocs/op
--- BENCH: BenchmarkConcurrent-8
        demoinfocs_test.go:425: Running concurrency benchmark with 8 demos
        demoinfocs_test.go:425: Running concurrency benchmark with 8 demos
PASS
ok      github.com/markus-wa/demoinfocs-golang  147.800s
```

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/markus-wa/demoinfocs-golang/tags).
There is one caveat however: Beta features - which are marked as such via comments and in release notes - may change in minor releases.

It's recommended to use some kind of dependency management system such as [dep](https://github.com/golang/dep) to ensure reproducible builds.

## Development

### Running tests

To run tests [Git LFS](https://git-lfs.github.com) is required.

```sh
git submodule init
git submodule update
pushd cs-demos && git lfs pull -I '*' && popd
go test
```

Here's a cool [gist of a pre-commit hook](https://gist.github.com/micvbang/4c8cb1f24cfe04d1a0dfab010eb851d8) to run tests before each commit. You can put this inside the `.git/hooks` directory to avoid committing/pushing code with build errors or failing tests.

### Debugging

You can use the build tag `debugdemoinfocs` (i.e. `go test -tags debugdemoinfocs -v`) to print out debugging information - such as game events or unhandled demo-messages - during the parsing process.<br>
Side-note: The tag isn't called `debug` to avoid naming conflicts with other libs (and underscores in tags don't work, apparently).

To change the default debugging behavior Go's `ldflags` parameter can be used. Example for additionally printing out all server-classes with their properties: `-ldflags '-X github.com/markus-wa/demoinfocs-golang.debugServerClasses=YES'`

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
