# demoinfocs-golang - CS2 Demo Parser

A blazing fast, feature complete and production ready Go library for parsing and analysing of Counter-Strike 2 and Counter-Strike: Global Offensive (CS:GO) demos (aka replays).

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs?tab=doc)
[![Build Status](https://img.shields.io/github/actions/workflow/status/markus-wa/demoinfocs-golang/ci.yml?branch=master&style=flat-square)](https://github.com/markus-wa/demoinfocs-golang/actions)
[![codecov](https://img.shields.io/codecov/c/github/markus-wa/demoinfocs-golang?style=flat-square)](https://codecov.io/gh/markus-wa/demoinfocs-golang)
[![Go Report](https://goreportcard.com/badge/github.com/markus-wa/demoinfocs-golang?style=flat-square)](https://goreportcard.com/report/github.com/markus-wa/demoinfocs-golang)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE.md)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang?ref=badge_shield)
[![Snyk Scan](https://img.shields.io/badge/security%20scan-snyk-blueviolet?style=flat-square)](https://app.snyk.io/org/markus-wa/project/cbd87cca-a903-4553-b677-42f3bb086655)

## Discussions / Chat

You can use [Discord](https://discord.gg/eTVBgKeHnh) or [GitHub Discussions](https://github.com/markus-wa/demoinfocs-golang/discussions) to ask questions and discuss ideas about this project.<br>
For business inquiries please use the contact information found on the [GitHub profile](https://github.com/markus-wa).

[![Discord Chat](https://img.shields.io/discord/901824796302643281?color=%235865F2&label=discord&style=for-the-badge)](https://discord.gg/eTVBgKeHnh)

## Go Get

### Counter-Strike 2

	go get -u github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs

### CS:GO

	go get -u github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs

## Table of Contents

- [Requirements](https://github.com/markus-wa/demoinfocs-golang#requirements)
- [Quickstart Guide](https://github.com/markus-wa/demoinfocs-golang#quickstart-guide)
  - [Example](https://github.com/markus-wa/demoinfocs-golang#example)
  - [More Examples](https://github.com/markus-wa/demoinfocs-golang#more-examples)
  - [Documentation](https://github.com/markus-wa/demoinfocs-golang#documentation)
- [Features](https://github.com/markus-wa/demoinfocs-golang#features)
- [Performance / Benchmarks](https://github.com/markus-wa/demoinfocs-golang#performance--benchmarks)
- [Versioning](https://github.com/markus-wa/demoinfocs-golang#versioning)
- [Projects Using demoinfocs-golang](https://github.com/markus-wa/demoinfocs-golang#projects-using-demoinfocs-golang)
- [Development](https://github.com/markus-wa/demoinfocs-golang#development)
  - [Debugging](https://github.com/markus-wa/demoinfocs-golang#debugging)
  - [Testing](https://github.com/markus-wa/demoinfocs-golang#testing)
  - [Generating Interfaces](https://github.com/markus-wa/demoinfocs-golang#generating-interfaces)
  - [Generating Protobuf Code](https://github.com/markus-wa/demoinfocs-golang#generating-protobuf-code)
  - [Git Hooks](https://github.com/markus-wa/demoinfocs-golang#git-hooks)
- [Acknowledgements](https://github.com/markus-wa/demoinfocs-golang#acknowledgements)
- [License](https://github.com/markus-wa/demoinfocs-golang#license) (MIT)

## Requirements

This library requires at least `go 1.20` to run.
You can download the latest version of Go [here](https://golang.org/).

## Quickstart Guide

1. Download and install the latest version of Go [from golang.org](https://golang.org/dl/) or via your favourite package manager

2. Create a new Go Modules project

```terminal
mkdir my-project
cd my-project
go mod init my-project
go get -u github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs
```

3. Create a `main.go` file with the example below

4. Run `go run main.go`

### Example

This is a simple example on how to handle game events using this library.
It prints all kills in a given demo (killer, weapon, victim, was it a wallbang/headshot?) by registering a handler for [`events.Kill`](https://godoc.org/github.com/markus-wa/demoinfocs-golang/events#Kill).

Check out the [godoc of the `events` package](https://godoc.org/github.com/markus-wa/demoinfocs-golang/events) for some information about the other available events and their purpose.

```go
package main

import (
	"fmt"
	"log"
	"os"

	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

func main() {
	f, err := os.Open("/path/to/demo.dem")
	if err != nil {
		log.Panic("failed to open demo file: ", err)
	}
	defer f.Close()

	p := dem.NewParser(f)
	defer p.Close()

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
		log.Panic("failed to parse demo: ", err)
	}
}
```

#### Sample Output

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

### More Examples

Check out the [examples](examples) folder for more examples, like [how to generate heatmaps](examples/heatmap) like this one:

<img alt="sample heatmap" src="https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/master/examples/heatmap/heatmap.jpg" width="50%">

### Documentation

The full API documentation is available here on [pkg.go.dev](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs).

## Features

* Game events (kills, shots, round starts/ends, footsteps etc.) - [docs](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events?tab=doc) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/print-events)
* Tracking of game-state (players, teams, grenades, ConVars etc.) - [docs](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs?tab=doc#GameState)
* Grenade projectiles / trajectories - [docs](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs?tab=doc#GameState.GrenadeProjectiles) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/nade-trajectories)
* Access to entities, server-classes & data-tables - [docs](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables?tab=doc#ServerClasses) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/entities)
* Access to all net-messages - [docs](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs?tab=doc#NetMessageCreator) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/net-messages)
* Chat & console messages <sup id="achat1">1</sup> - [docs](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events?tab=doc#ChatMessage) / [example](https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/print-events)
* Matchmaking ranks (official MM demos only) - [docs](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events?tab=doc#RankUpdate)
* Full POV demo support
* Support for encrypted net-messages (if the [decryption key](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4@master/pkg/demoinfocs#ParserConfig) is provided)
* JavaScript (browser / Node.js) support via WebAssembly - [example](https://github.com/markus-wa/demoinfocs-wasm)
* [Easy debugging via build-flags](#debugging)
* Built with performance & concurrency in mind

1. <small id="f1">In MM demos the chat is encrypted, so [`ParserConfig.NetMessageDecryptionKey`](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4@master/pkg/demoinfocs#ParserConfig) needs to be set - see also [`MatchInfoDecryptionKey()`](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v4@master/pkg/demoinfocs#MatchInfoDecryptionKey).</small>

## Performance / Benchmarks

Two of the top priorities of this parser are performance and concurrency.

Here are some benchmark results from a system with an Intel i7 6700k CPU and a SSD disk running Windows 10 and a demo with 85'000 frames.

### Overview

|Benchmark|Description|Average Duration|Speed|
|-|-|-|-|
|`BenchmarkConcurrent`|Read and parse 8 demos concurrently|2.06 s (per 8 demos)|~ 1 h 25 min of gameplay per second|
|`BenchmarkDemoInfoCs`|Read demo from drive and parse|0.89 s|~ 25 min of gameplay per second|
|`BenchmarkInMemory`|Read demo from memory and parse|0.88 s|~ 25 min of gameplay per second|

### Raw Output

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

## Services, Projects & Companies Using demoinfocs-golang

- [noesis.gg](https://www.noesis.gg/) - A suite of explorative tools to help you analyze and improve your CS2 performance
- [esportal.com](https://esportal.com/) - An alternative Matchmaking service that aims to provide a friendly environment free from trolls and misbehaving individuals
- [pglesports.com](https://www.pglesports.com/) - Premier eSports tournaments and circuits for massive audiences
- [hltv.org](https://www.hltv.org/) - Leading Counter-Strike site featuring news, demos, pictures, statistics, on-site coverage and more
- [refrag.gg](https://refrag.gg/) - The world's premier CS2 training tool
- [esportslab.gg](https://esportslab.gg/) - Building ML/AI tools for professional esports players
- [scope.gg](https://scope.gg/) - Analytical and advisory service for advanced CS2 players
- [PureSkill.gg](https://pureskill.gg/) - An automated coach to help you get better at CS2
- [cs2lens.com](https://www.cs2lens.com/) - Professional CS2 demo replayer and analysis tool
- [awpy](https://github.com/pnxenopoulos/awpy) - A wrapper for the Golang parser in Python

### CS:GO projects (may no longer work with CS2)

- [cs-demo-minifier](https://github.com/markus-wa/cs-demo-minifier) - Converts demos to JSON, MessagePack and more
- [csgo_spray_pattern_plotter](https://github.com/o40/csgo_spray_pattern_plotter) - A tool to extract and plot spray patterns from CS:GO replays
- [CS:GO Player Skill Prediction](https://drive.google.com/file/d/1JXIB57BA2XBTYVLSy6Xg_5nfL6dWyDmG/view) - Machine learning master thesis by [@quancore](https://github.com/quancore) about predicting player performance
- [csgoverview](https://github.com/Linus4/csgoverview) - A 2D demo replay tool for CS:GO
- [csgo-coach-bug-detector](https://github.com/softarn/csgo-coach-bug-detector) - Detects the abuse of an exploit used by some team coaches in professional matches
- [megaclan3000](https://github.com/megaclan3000/megaclan3000) - A CS:GO stats page for clans with recent matches and player statistics

If your project is using this library feel free to submit a PR or send a message via [Discord](https://discord.gg/eTVBgKeHnh) to be included in the list.

## Useful Tools & Libraries

- [csgo-centrifuge](https://github.com/saiko-tech/csgo-centrifuge) - Get historic radar overview images to make 2D replays & heatmaps accurate on all map versions
- [head-position-model](https://github.com/David-Durst/head-position-model) - Approximate the player's head position (rather than just the camera position)

## Development

### Debugging

You can use the build tag `debugdemoinfocs` to print out debugging information - such as game events or unhandled demo-messages - during the parsing process.<br>

e.g.

    go run -tags debugdemoinfocs examples/print-events/print_events.go -demo example.dem

Side-note: The tag isn't called `debug` to avoid naming conflicts with other libs (and underscores in tags don't work, apparently).

To change the default debugging behavior, Go's `ldflags` parameter can be used. Example for additionally printing out all server-classes with their properties: `-ldflags="-X 'github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs.debugServerClasses=YES'"`.

e.g.

    go run -tags debugdemoinfocs -ldflags="-X 'github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs.debugServerClasses=YES'" examples/print-events/print_events.go -demo example.dem

Check out `debug_on.go` for any other settings that can be changed.

### Testing

#### Unit Tests

For any new features, [Test Driven Development](https://medium.com/@pierreprinetti/test-driven-development-in-go-baeab5adb468) should be practiced where possible.
However, due to some design flaws in some parts of the code it's currently not always practical to do so.

Running unit tests:

    scripts/unit-tests.sh
    # or (identical)
    go test -short ./...

#### Regression Tests

For the full regression suite you will need to download the test demo-set.

Prerequisites:
- [Git LFS](https://git-lfs.github.com) must be installed
- [`7z`](https://www.7-zip.org/) must be in your `PATH` environment variable (`p7zip` or `p7zip-full` package on most Linux distros)

Downloading demos + running regression tests:

    scripts/regression-tests.sh

#### Updating the `default.golden` File

The file [`test/default.golden`](https://github.com/markus-wa/demoinfocs-golang/blob/master/test/default.golden) file contains a serialized output of all expected game events in `test/cs-demos/default.dem`.

If there is a change to game events (new fields etc.) it is necessary to update this file so the regression tests pass.
To update it you can run the following command:

	go test -run TestDemoInfoCs -update

Please don't update the `.golden` file if you are not sure it's required. Maybe the failing CI is just pointing out a regression.

### Generating Interfaces

We generate interfaces such as `GameState` from structs to make it easier to keep docs in synch over structs and interfaces.
For this we use [@vburenin](https://github.com/vburenin)'s [`ifacemaker`](https://github.com/vburenin/ifacemaker) tool.

You can download the latest version [here](https://github.com/vburenin/ifacemaker/releases).
After adding it to your `PATH` you can use `scripts/generate-interfaces.sh` to update interfaces.

### Generating Protobuf Code

Should you need to re-generate the protobuf generated code in the `msg` package, you will need the following tools:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

Make sure both are inside your `PATH` variable.

After installing these use `go generate ./msg` to generate the protobuf code. If you're on Windows you'll need to run go generate from CMD, not Bash.

### Git Hooks

To install some (optional, but quite handy) `pre-commit` and `pre-push` hooks, you can run the following script.

    scripts/git-hooks/link-git-hooks.sh

#### `pre-commit`:
- check if [interfaces have been updated](#generating-interfaces)
- build the code
- run unit tests

#### `pre-push`:
- run regression tests

## Acknowledgements

This library was originally based on <a href="https://github.com/ValveSoftware/csgo-demoinfo" rel="external">Valve's demoinfogo</a> and <a href="https://github.com/StatsHelix/demoinfo" rel="external">SatsHelix's demoinfo</a>.

For Counter-Strike 2, [dotabuff/manta](https://github.com/dotabuff/manta) was an amazing resource for how to parse Source 2 demos and CS2 support would not have been possible without it.<br>
I would also like to specifically thank [@akiver](https://github.com/akiver) & [@LaihoE](https://github.com/LaihoE) for their brilliant help with CS2.

And a very special thanks goes out to all the [⭐contributors⭐](https://github.com/markus-wa/demoinfocs-golang/graphs/contributors)️, be it in the form of PRs, issues or anything else.

Further shoutouts go to:

- [@JuhaKiili](https://github.com/JuhaKiili) for financial contributions in the form of bug bounties
- [@PGL-ESPORTS](https://github.com/PGL-ESPORTS), [@esportalgroup](https://github.com/esportalgroup), [@refrag](https://github.com/refrag) & [@pureskillgg](https://github.com/pureskillgg) for offering past & present consulting work that keeps my lights on

## License

This project is licensed under the [MIT license](LICENSE.md).

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fmarkus-wa%2Fdemoinfocs-golang?ref=badge_large)
