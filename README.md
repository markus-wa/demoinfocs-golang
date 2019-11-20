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
		fmt.Printf("%s <%v%s%s> %s\n", e.Killer, e.Weapon, hs, wallBang, e.Victim)
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
* Tracking of game-state (players, teams, grenades, ConVars etc.) - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang#GameState)
* Grenade projectiles / trajectories - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang#GameState.GrenadeProjectiles) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/nade-trajectories)
* Access to entities, server-classes & data-tables - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang/sendtables#ServerClasses) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/entities)
* Access to all net-messages - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang#NetMessageCreator) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/net-messages)
* Chat & console messages <sup id="achat1">1</sup> - [docs](https://godoc.org/github.com/markus-wa/demoinfocs-golang/events#ChatMessage) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/print-events)
* POV demo support <sup id="achat1">2</sup>
* [Easy debugging via build-flags](#debugging)
* Built with performance & concurrency in mind

1. <small id="f1">Only for some demos; in MM demos the chat is encrypted for example.</small>
2. <small id="f2">Only partially supported (as good as other parsers), some POV demos seem to be inherently broken</small>

## Performance / Benchmarks

Two of the top priorities of this parser are performance and concurrency.

Here are some benchmark results from a system with an Intel i7 6700k CPU and a SSD disk running Windows 10 and a demo with 85'000 frames.

### Overview

|Benchmark|Description|Average Duration|Speed|
|-|-|-|-|
|`BenchmarkConcurrent`|Read and parse 8 demos concurrently|2.06 s (per 8 demos)|~330'000 ticks / s|
|`BenchmarkDemoInfoCs`|Read demo from drive and parse|0.89 s|~95'000 ticks / s
|`BenchmarkInMemory`|Read demo from memory and parse|0.88 s|~96'000 ticks / s

*That's almost 1.5 hours of gameplay per second when parsing in parallel (recorded at 64 ticks per second) - or 25 minues per second when only parsing a single demo at a time.*

### Raw output

```
$ go test -run _NONE_ -bench . -benchtime 30s -benchmem -concurrentdemos 8
goos: windows
goarch: amd64
pkg: github.com/markus-wa/demoinfocs-golang
BenchmarkDemoInfoCs-8             50     894500010 ns/op    257610127 B/op    914355 allocs/op
BenchmarkInMemory-8               50     876279984 ns/op    257457271 B/op    914143 allocs/op
BenchmarkConcurrent-8             20    2058303680 ns/op    2059386582 B/op  7313145 allocs/op
--- BENCH: BenchmarkConcurrent-8
    demoinfocs_test.go:315: Running concurrency benchmark with 8 demos
    demoinfocs_test.go:315: Running concurrency benchmark with 8 demos
PASS
ok      github.com/markus-wa/demoinfocs-golang  134.244s
```

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/markus-wa/demoinfocs-golang/tags).
There is one caveat however: Beta features - which are marked as such via comments and in release notes - may change in minor releases.

## Projects using demoinfocs-golang

- [noesis.gg](https://www.noesis.gg/) - A suite of explorative tools to help you analyze and improve your CS:GO performance
- [cs-demo-minifier](https://github.com/markus-wa/cs-demo-minifier) - Converts demos to JSON, MessagePack and more
- [csgo_spray_pattern_plotter](https://github.com/o40/csgo_spray_pattern_plotter) - A tool to extract and plot spray patterns from CS:GO replays
- [CS:GO Player Skill Prediction](https://drive.google.com/file/d/1JXIB57BA2XBTYVLSy6Xg_5nfL6dWyDmG/view) - Machine learning master thesis by [@quancore](https://github.com/quancore) about predicting player performance
- [csgoverview](https://github.com/Linus4/csgoverview) - A 2D demo replay tool for CS:GO

If your project is using this library feel free to submit a PR or send a message in Gitter to be included in the list.

## Development

### Git hooks

To install some (optional, but quite handy) `pre-commit` and `pre-push` hooks, you can run the following script.

    bin/git-hooks/link-git-hooks.sh

#### `pre-commit`:
- check if [interfaces have been updated](#generating-interfaces)
- build the code
- run unit tests

#### `pre-push`:
- run regression tests

### Testing

#### Unit tests

For any new features, [Test Driven Development](https://medium.com/@pierreprinetti/test-driven-development-in-go-baeab5adb468) should be practiced where possible.
However, due to some design flaws in some parts of the code it's currently not always practical to do so.

Running unit tests:

    bin/unit-tests.sh
    # or (identical)
    go test -short ./...

#### Regression tests

For the full regression suite you will need to download the test demo-set.

Prerequisites:
- [Git LFS](https://git-lfs.github.com) must be installed
- [`7z`](https://www.7-zip.org/) must be in your `PATH` environment variable (`p7zip` or `p7zip-full` package on most Linux distros)

Downloading demos + running regression tests:

    bin/regression-tests.sh

#### Updating the `default.golden` file

The file [`test/default.golden`](https://github.com/markus-wa/demoinfocs-golang/blob/master/test/default.golden) file contains a serialized output of all expected game events in `test/cs-demos/default.dem`.

If there is a change to game events (new fields etc.) it is necessary to update this file so the regression tests pass.
To update it you can run the following command:

	go test -run TestDemoInfoCs -update

Please don't update the `.golden` file if you are not sure it's required. Maybe the failing CI is just pointing out a regression.

### Debugging

You can use the build tag `debugdemoinfocs` (i.e. `go test -tags debugdemoinfocs -v`) to print out debugging information - such as game events or unhandled demo-messages - during the parsing process.<br>
Side-note: The tag isn't called `debug` to avoid naming conflicts with other libs (and underscores in tags don't work, apparently).

To change the default debugging behavior, Go's `ldflags` parameter can be used. Example for additionally printing out all server-classes with their properties: `-ldflags '-X github.com/markus-wa/demoinfocs-golang.debugServerClasses=YES'`

Check out `debug_on.go` for any other settings that can be changed.

### Generating interfaces

We generate interfaces such as `IGameState` from structs to make it easier to keep docs in synch over structs and interfaces.
For this we use [@vburenin](https://github.com/vburenin)'s [`ifacemaker`](https://github.com/vburenin/ifacemaker) tool.

You can download the latest version [here](https://github.com/vburenin/ifacemaker/releases).
After adding it to your `PATH` you can use `bin/generate-interfaces.sh` to update interfaces.

### Generating protobuf code

Should you need to re-generate the protobuf generated code in the `msg` package, you will need the following tools:

- The latest protobuf generator (`protoc`) from your package manager or https://github.com/google/protobuf/releases

- And `protoc-gen-gogofaster` from [gogoprotobuf](https://github.com/gogo/protobuf) to generate code for go.

		go get -u github.com/gogo/protobuf/protoc-gen-gogofaster

[//]: # "The go get above needs two tabs so it's displayed a) as part of the last list entry and b) as a code-block"
[//]: # "Oh and don't try to move these comments above it either"

Make sure both are inside your `PATH` variable.

After installing these use `go generate ./msg` to generate the protobuf code. If you're on Windows you'll need to run go generate from CMD, not Bash.

## Acknowledgements

Thanks to [@JetBrains](https://github.com/JetBrains) for sponsoring a license of their awesome [GoLand](https://www.jetbrains.com/go/) IDE for this project - go check it out!

And a very special thanks goes out to all the ⭐️contributors⭐️, be it in the form of PRs, issues or anything else.

## License

This project is licensed under the [MIT license](LICENSE.md).

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang?ref=badge_large)
