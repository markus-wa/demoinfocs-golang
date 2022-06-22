# Nade trajectory overview

This example shows how to create a overview of grenade trajectories of a match.

:information_source: Uses radar images from `https://radar-overviews.csgo.saiko.tech/<map>/<crc>/radar.png` - see https://github.com/saiko-tech/csgo-centrifuge for more info.

## Running the example

`go run nade_trajectories.go -demo /path/to/demo > out.jpg`

This will create a JPEG with grenade trajectories of the first five rounds. The reason it doesn't do more trajectories is because the image would look quite cluttered otherwise.

![Resulting map overview with trajectories](https://raw.githubusercontent.com/markus-wa/demoinfocs-golang/master/examples/nade-trajectories/nade_trajectories.jpg)
