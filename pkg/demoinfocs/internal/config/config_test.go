package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	// same values as the former demoinfocs.DefaultParserConfig
	assert.Equal(t, Config{
		MsgQueueBufferSize: -1,
		UserCmdParsing:     UserCmdParsingButtonsOnly,
	}, Default())

	// the zero value of UserCmdParsingMode is the default
	var zero Config
	assert.Equal(t, UserCmdParsingButtonsOnly, zero.UserCmdParsing)
	assert.Equal(t, FormatFile, zero.Format)
}

func TestNew(t *testing.T) {
	assert.Equal(t, Default(), New[func(*Config)](nil))

	cfg := New([]func(*Config){
		func(c *Config) { c.EagerVelocityTracking = true },
		func(c *Config) { c.EagerVelocityTracking = false },
		func(c *Config) { c.MsgQueueBufferSize = 100 },
	})

	assert.False(t, cfg.EagerVelocityTracking, "last option should win")
	assert.Equal(t, 100, cfg.MsgQueueBufferSize)
	assert.Equal(t, Default().UserCmdParsing, cfg.UserCmdParsing)
}
