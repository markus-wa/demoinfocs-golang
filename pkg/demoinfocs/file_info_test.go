package demoinfocs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	msg "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

func TestParser_HandleFileInfo_MissingFields(t *testing.T) {
	p := newParser()

	assert.NotPanics(t, func() { p.handleFileInfo(&msg.CDemoFileInfo{}) })
	assert.Zero(t, p.header.PlaybackTicks)
	assert.Zero(t, p.header.PlaybackFrames)
	assert.Zero(t, p.header.PlaybackTime)

	p.handleFileInfo(&msg.CDemoFileInfo{
		PlaybackTicks:  proto.Int32(128),
		PlaybackFrames: proto.Int32(64),
		PlaybackTime:   proto.Float32(2),
	})

	assert.Equal(t, 128, p.header.PlaybackTicks)
	assert.Equal(t, 64, p.header.PlaybackFrames)
	assert.Equal(t, 2*time.Second, p.header.PlaybackTime)
}
