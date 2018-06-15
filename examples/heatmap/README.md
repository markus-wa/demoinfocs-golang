# Creating a heatmap

This example shows how to create a heatmap from positions where players fired their weapons from.

## Running the example

`go run heatmap.go -demo /path/to/demo > out.png`

This will create a PNG with dots on all the locations where shots were fired (the heatmap 'overlay').

This doesn't look too interesting on it's own but that can be helped by quickly mapping it to the map overview in an image editing tool (2 min tops, no skills required).

![Resulting heatmap before and after mapping to map overview](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/master/examples/heatmap/heatmap.jpg)
