# Capturing Voice Data

This example shows how to use the library to capture voice chat in CELT format.

See https://gist.github.com/ericek111/abe5829f6e52e4b25b3b97a0efd0b22b for how to play it back.

## Prerequisites

You need to have the following installed:

- Linux (macOS or WSL may work but not tested)
- CS:GO Linux Binaries
- CELT - Audio Codec Library
- Sox - Sound Processing Tools (for conversion to `.wav`)

## Running the example

```terminal
export CSGO_BIN="$STEAM_LIBRARY/steamapps/common/Counter-Strike Global Offensive/bin/linux64"
export CGO_LDFLAGS="-L \"$CSGO_BIN\" -l:vaudio_celt_client.so"
export LD_LIBRARY_PATH=$CSGO_BIN

go run capture_voice.go -demo /path/to/demo

play -t raw -r 22050 -e signed -b 16 -c 1 out.celt
sox -t raw -r 22050 -e signed -b 16 -c 1 -L out.celt out.wav 
```
