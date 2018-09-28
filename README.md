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

## Go Get

	go get -u github.com/markus-wa/demoinfocs-golang

## Example

This is a simple example on how to use the library. It collects all positions where weapons were fired from (using `events.WeaponFire`) and creates a heatmap using [go-heatmap](https://github.com/dustin/go-heatmap).

Check out the [examples](examples) folder for more examples and the [godoc of the `events` package](https://godoc.org/github.com/markus-wa/demoinfocs-golang/events) for some information about the other available events and their purpose.

```go
package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os"

	heatmap "github.com/dustin/go-heatmap"
	schemes "github.com/dustin/go-heatmap/schemes"

	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
	metadata "github.com/markus-wa/demoinfocs-golang/metadata"
)

// Run like this: go run heatmap.go > out.jpg
func main() {
	f, err := os.Open("/path/to/demo.dem")
	checkError(err)
	defer f.Close()

	p := dem.NewParser(f)

	// Parse header (contains map-name etc.)
	header, err := p.ParseHeader()
	checkError(err)

	// Get metadata for the map that's being played
	m := metadata.MapNameToMap[header.MapName]

	// Register handler for WeaponFire, triggered every time a shot is fired
	points := []heatmap.DataPoint{}
	var bounds image.Rectangle
	p.RegisterEventHandler(func(e events.WeaponFire) {
		// Translate positions from in-game coordinates to radar overview
		x, y := m.TranslateScale(e.Shooter.Position.X, e.Shooter.Position.Y)

		// Track bounds to get around the normalization done by the heatmap library
		bounds = updatedBounds(bounds, int(x), int(y))

		// Add shooter's position as datapoint
		// Invert Y since it expects data to be ordered from bottom to top
		points = append(points, heatmap.P(x, y*-1))
	})

	// Parse to end
	err = p.ParseToEnd()
	checkError(err)

	// Get map overview as base image
	fMap, err := os.Open(fmt.Sprintf("../../metadata/maps/%s.jpg", header.MapName))
	checkError(err)

	imgMap, _, err := image.Decode(fMap)
	checkError(err)

	// Create output canvas
	img := image.NewRGBA(imgMap.Bounds())

	draw.Draw(img, imgMap.Bounds(), imgMap, image.ZP, draw.Over)

	// Generate heatmap
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	imgHeatmap := heatmap.Heatmap(image.Rect(0, 0, width, height), points, 15, 128, schemes.AlphaFire)

	// Draw it on top of the overview
	draw.Draw(img, bounds, imgHeatmap, image.ZP, draw.Over)

	// Write to stdout
	jpeg.Encode(os.Stdout, img, &jpeg.Options{
		Quality: 90,
	})
}

func updatedBounds(b image.Rectangle, x, y int) image.Rectangle {
	if b.Min.X > x || b.Min.X == 0 {
		b.Min.X = x
	} else if b.Max.X < x {
		b.Max.X = x
	}

	if b.Min.Y > y || b.Min.Y == 0 {
		b.Min.Y = y
	} else if b.Max.Y < y {
		b.Max.Y = y
	}

	return b
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
```

### Result

Running the above code (`go run heatmap.go > heatmap.png`) will create a JPEG of a radar overview with dots on all the locations where shots were fired.

![Resulting heatmap](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/master/examples/heatmap/heatmap.jpg)

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

One of the top priorities of this parser is performance and concurrency.

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

### Debugging

You can use the build tag `debugdemoinfocs` (i.e. `go test -tags debugdemoinfocs -v`) to print out debugging information - such as game events or unhandled demo-messages - during the parsing process.<br>
Side-note: The tag isn't called `debug` to avoid naming conflicts with other libs (and underscores in tags don't work, apparently).

To change the default debugging behavior Go's `ldflags` paramter can be used. Example for additionally printing out all server-classes with their properties: `-ldflags '-X github.com/markus-wa/demoinfocs-golang.debugServerClasses=YES'`

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
