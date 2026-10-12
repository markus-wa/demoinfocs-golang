// Package config holds the Parser configuration shared by the demoinfocs and option packages.
// It's internal so the configuration can only be changed through the option package.
package config

import (
	"google.golang.org/protobuf/proto"
)

// Format is the format of the demo stream (".dem" file or live CSTV broadcast).
// It's not configurable: demoinfocs.NewParser() derives it from the reader type (*cstv.Reader or not).
type Format byte

const (
	FormatFile Format = iota
	FormatCSTVBroadcast
)

// UserCmdParsingMode controls how CSVCMsg_UserCommands messages are handled.
// Exported as option.UserCmdParsingMode, see there for the documentation.
type UserCmdParsingMode uint8

const (
	UserCmdParsingButtonsOnly UserCmdParsingMode = iota
	UserCmdParsingFull
	UserCmdParsingDisabled
)

// NetMessageCreator creates additional net-messages to be dispatched to net-message handlers.
// Exported as option.NetMessageCreator, see there for the documentation.
type NetMessageCreator func() proto.Message

// Config contains the configuration of a Parser, see the option package for the documentation of the fields.
type Config struct {
	MsgQueueBufferSize              int
	AdditionalNetMessageCreators    map[int]NetMessageCreator
	IgnoreErrBombsiteIndexNotFound  bool
	NetMessageDecryptionKey         []byte
	DisableMimicSource1Events       bool
	Source2FallbackGameEventListBin []byte
	IgnorePacketEntitiesPanic       bool
	Format                          Format
	UserCmdParsing                  UserCmdParsingMode
	EagerVelocityTracking           bool
}

// Default returns the configuration options are applied on top of.
func Default() Config {
	return Config{
		MsgQueueBufferSize: -1,
		UserCmdParsing:     UserCmdParsingButtonsOnly,
	}
}

// New returns the default configuration with opts applied in order.
func New[O ~func(*Config)](opts []O) Config {
	cfg := Default()

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}
