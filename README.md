# demoinfocs-golang

Is a high performance CS:GO demo parser written in Go based on [Valve's demoinfogo](https://github.com/ValveSoftware/csgo-demoinfo) and [SatsHelix's demoinfo](https://github.com/StatsHelix/demoinfo).

[![GoDoc](https://godoc.org/github.com/markus-wa/demoinfocs-golang?status.svg)](https://godoc.org/github.com/markus-wa/demoinfocs-golang)
[![Build Status](https://travis-ci.org/markus-wa/demoinfocs-golang.svg?branch=master)](https://travis-ci.org/markus-wa/demoinfocs-golang)
[![codecov](https://codecov.io/gh/markus-wa/demoinfocs-golang/branch/master/graph/badge.svg)](https://codecov.io/gh/markus-wa/demoinfocs-golang)
[![Go Report](https://goreportcard.com/badge/github.com/markus-wa/demoinfocs-golang)](https://goreportcard.com/report/github.com/markus-wa/demoinfocs-golang)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE.md)

## Discussions / Chat

You can use gitter to ask questions and discuss ideas about this project.<br>
There are also [some other rooms](https://gitter.im/csgodemos) available around the topic of CS:GO demos.

[![Gitter chat](https://badges.gitter.im/csgodemos/demoinfo-lib.png)](https://gitter.im/csgodemos/demoinfo-lib)

## Go Get

	go get -u github.com/markus-wa/demoinfocs-golang

## Example

This is a simple example on how to use the library. It collects all positions where weapons were fired and creates a heatmap using [go-heatmap](https://github.com/dustin/go-heatmap).

Check out the `examples` folder for more examples and the godoc of the `events` package for some information about the other available events and their purpose.

```go
package main

import (
	"image"
	"image/png"
	"log"
	"os"

	heatmap "github.com/dustin/go-heatmap"
	schemes "github.com/dustin/go-heatmap/schemes"

	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

// Run like this: go run heatmap.go > out.png
func main() {
	f, err := os.Open("/path/to/demo.dem")
	checkErr(err)
	defer f.Close()

	p := dem.NewParser(f)

	// Parse header (contains map-name etc.)
	_, err = p.ParseHeader()
	checkErr(err)

	// Register handler for WeaponFiredEvent, triggered every time a shot is fired
	points := []heatmap.DataPoint{}
	p.RegisterEventHandler(func(e events.WeaponFiredEvent) {
		// Add shooter's position as datapoint
		points = append(points, heatmap.P(e.Shooter.Position.X, e.Shooter.Position.Y))
	})

	// Parse to end
	err = p.ParseToEnd()
	checkErr(err)

	// Generate heatmap and write to standard output
	img := heatmap.Heatmap(image.Rect(0, 0, 1024, 1024), points, 15, 128, schemes.AlphaFire)
	png.Encode(os.Stdout, img)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
```

### Result

Running the above code (`go run heatmap.go > heatmap.png`) will create a PNG with dots on all the locations where shots were fired (the heatmap 'overlay').

This doesn't look too interesting on it's own but that can be helped by quickly mapping it to the map overview in an image editing tool (2 min tops, no skills required).

![Resulting heatmap before and after mapping to map overview](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/master/examples/heatmap/heatmap.jpg)

## Features

* Game events
* Access to entities, server-classes & data-tables
* Access to all net-messages
* Chat & console messages <sup id="achat1">1</sup>
* [Easy debugging](###Debugging) via build-flags
* Built with concurrency in mind

1. <small id="f1">Only for some demos; in MM demos the chat is encrypted for example.</small>

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
pushd cs-demos && git lfs pull && popd
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
