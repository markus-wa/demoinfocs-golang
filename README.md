# demoinfocs-golang - A CS:GO Demo Parser

Is a Go library for super fast parsing and analysing of Counter Strike: Global Offensive (CS:GO) demos (aka replays). It is based on <a href="https://github.com/ValveSoftware/csgo-demoinfo" rel="external">Valve's demoinfogo</a> and <a href="https://github.com/StatsHelix/demoinfo" rel="external">SatsHelix's demoinfo</a>.

[![GoDoc](https://godoc.org/github.com/markus-wa/demoinfocs-golang?status.svg)](https://godoc.org/github.com/markus-wa/demoinfocs-golang)
[![Build Status](https://travis-ci.org/markus-wa/demoinfocs-golang.svg?branch=master)](https://travis-ci.org/markus-wa/demoinfocs-golang)
[![codecov](https://codecov.io/gh/markus-wa/demoinfocs-golang/branch/master/graph/badge.svg)](https://codecov.io/gh/markus-wa/demoinfocs-golang)
[![Go Report](https://goreportcard.com/badge/github.com/markus-wa/demoinfocs-golang)](https://goreportcard.com/report/github.com/markus-wa/demoinfocs-golang)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE.md)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang?ref=badge_shield)

## Discussions / Chat

You can use gitter to ask questions and discuss ideas about this project.

[![Gitter chat](https://badges.gitter.im/csgodemos/demoinfo-lib.png)](https://gitter.im/csgodemos/demoinfo-lib)

## Requirements

This library is intended to be used with `go 1.11` or higher as it is built using Go modules.

It's recommended to use modules for consumers as well if possible.
If you are unfamiliar with Go modules there's a [list of recommended resources](https://github.com/markus-wa/demoinfocs-golang/wiki/Go-Modules#recommended-links--articles) in the wiki.

## Go Get

	go get -u github.com/markus-wa/demoinfocs-golang

If you are using a older (`v0.x`) version of the library, check out the [`v0.5` branch](https://github.com/markus-wa/demoinfocs-golang/tree/v0.5). However, it's highly recommended to switch to master (v1.0.0), which contains many new features and improvements. It also promises compatibility with new relases for the foreseeable future.

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

<img alt="sample heatmap" src="https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/master/examples/heatmap/heatmap.jpg" width="50%">

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

Here are some benchmark results from a system with an Intel i7 2600k CPU and a SSD disk running Windows 10 and a demo with 85'000 frames.

### Overview

|Benchmark|Description|Average Duration|Speed|
|-|-|-|-|
|`BenchmarkConcurrent`|Read and parse 8 demos concurrently|2.84 s (per 8 demos)|~240'000 ticks / s|
|`BenchmarkDemoInfoCs`|Read demo from drive and parse|1.24 s|~68'000 ticks / s
|`BenchmarkInMemory`|Read demo from memory and parse|1.21 s|~70'000 ticks / s

*That's about 1h of gameplay per second when parsing in parallel (recorded at 64 ticks per second) - or 18 minues per second when only parsing a single demo at a time.*

### Raw output

```
$ go test -run _NONE_ -bench . -benchtime 30s -benchmem -concurrentdemos 8
goos: windows
goarch: amd64
pkg: github.com/markus-wa/demoinfocs-golang
BenchmarkDemoInfoCs-8             30    1237133300 ns/op    256055133 B/op    879104 allocs/op
BenchmarkInMemory-8               30    1216333013 ns/op    255900492 B/op    878900 allocs/op
BenchmarkConcurrent-8             20    2840799900 ns/op    2046866843 B/op  7031208 allocs/op
--- BENCH: BenchmarkConcurrent-8
    demoinfocs_test.go:369: Running concurrency benchmark with 8 demos
    demoinfocs_test.go:369: Running concurrency benchmark with 8 demos
    demoinfocs_test.go:369: Running concurrency benchmark with 8 demos
PASS
ok      github.com/markus-wa/demoinfocs-golang  165.244s
```

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/markus-wa/demoinfocs-golang/tags).
There is one caveat however: Beta features - which are marked as such via comments and in release notes - may change in minor releases.

It's recommended to use some kind of dependency management system such as [dep](https://github.com/golang/dep) to ensure reproducible builds.

## Projects using demoinfocs-golang

- [noesis.gg](https://www.noesis.gg/) - A suite of explorative tools to help you analyze and improve your CS:GO performance.
- [cs-demo-minifier](https://github.com/markus-wa/cs-demo-minifier) - Converts demos to JSON, MessagePack and more

If your project is using this library feel free to submit a PR or send a message in Gitter to be included in the list.

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

To change the default debugging behavior, Go's `ldflags` parameter can be used. Example for additionally printing out all server-classes with their properties: `-ldflags '-X github.com/markus-wa/demoinfocs-golang.debugServerClasses=YES'`

Check out `debug_on.go` for any other settings that can be changed.

### Generating interfaces

We generate interfaces such as `IGameState` from structs to make it easier to keep docs in synch over structs and interfaces.
For this we use [@vburenin](https://github.com/vburenin)'s [`ifacemaker`](https://github.com/vburenin/ifacemaker) tool.
You can install it via `GO111MODULE=off go get github.com/vburenin/ifacemaker`.

After adding it to your `PATH` you can use `go generate` to update interfaces.

### Generating protobuf code

Should you need to re-generate the protobuf generated code in the `msg` package, you will need the following tools:

- The latest protobuf generator (`protoc`) from your package manager or https://github.com/google/protobuf/releases

- And `protoc-gen-gogofaster` from [gogoprotobuf](https://github.com/gogo/protobuf) to generate code for go.

		go get -u github.com/gogo/protobuf/protoc-gen-gogofaster

[//]: # "The go get above needs two tabs so it's displayed a) as part of the last list entry and b) as a code-block"
[//]: # "Oh and don't try to move these comments above it either"

Make sure both are inside your `PATH` variable.

After installing these use `go generate ./msg` to generate the protobuf code. If you're on Windows you'll need to run go generate from CMD, not Bash.


## License

This project is licensed under the [MIT license](LICENSE.md).

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang?ref=badge_large)
