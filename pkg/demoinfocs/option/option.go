/*
Package option provides the options for creating a demoinfocs.Parser.

Options are applied in order on top of the defaults, so the last one wins if an option is passed more than once.

CSTV broadcast settings (timeout, HTTP client) aren't parser options: pass them to cstv.NewReader()
and give the *cstv.Reader to demoinfocs.NewParser().

Example:

	p := demoinfocs.NewParser(f,
		option.MsgQueueBufferSize(0), // parse sequentially
		option.UserCmdParsing(option.UserCmdParsingFull),
		option.NetMessageDecryptionKey(key),
	)
	defer p.Close()

	err := p.ParseToEnd(ctx)
*/
package option

import (
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/config"
)

// Option configures a demoinfocs.Parser, see the functions of this package.
type Option func(*config.Config)

// NetMessageCreator creates additional net-messages to be dispatched to net-message handlers.
//
// See also: AdditionalNetMessageCreators() & demoinfocs.Parser.RegisterNetMessageHandler()
type NetMessageCreator = config.NetMessageCreator

// UserCmdParsingMode controls how CSVCMsg_UserCommands messages are handled, see UserCmdParsing().
type UserCmdParsingMode = config.UserCmdParsingMode

const (
	// UserCmdParsingButtonsOnly tracks buttonstate1 for PlayerButtonsStateUpdate events.
	// It is the default.
	UserCmdParsingButtonsOnly = config.UserCmdParsingButtonsOnly
	// UserCmdParsingFull reconstructs and dispatches complete UserCmd events.
	UserCmdParsingFull = config.UserCmdParsingFull
	// UserCmdParsingDisabled skips user-command parsing and preserves legacy properties.
	UserCmdParsingDisabled = config.UserCmdParsingDisabled
)

// MsgQueueBufferSize sets the size of the internal net-message queue.
// For large demos, fast i/o and slow CPUs higher numbers are suggested and vice versa.
// The buffer size can easily be in the hundred-thousands to low millions for the best performance.
// A negative value (the default) makes the Parser automatically decide the buffer size when parsing the header,
// based on the number of ticks in the demo (number of ticks = buffer size).
// Zero enforces sequential parsing.
func MsgQueueBufferSize(size int) Option {
	return func(c *config.Config) {
		c.MsgQueueBufferSize = size
	}
}

// AdditionalNetMessageCreators maps net-message-IDs to creators (instantiators).
// The creators should return a new instance of the correct protobuf-message type (from the msg package).
// Interesting net-message-IDs can easily be discovered with the build-tag 'debugdemoinfocs'; when looking for 'UnhandledMessage'.
// Check out the demoinfocs package's parsing.go to see which net-messages are already being parsed by default.
// This is a beta feature and may be changed or replaced without notice.
func AdditionalNetMessageCreators(creators map[int]NetMessageCreator) Option {
	return func(c *config.Config) {
		c.AdditionalNetMessageCreators = creators
	}
}

// IgnoreErrBombsiteIndexNotFound tells the parser to not return an error when a bombsite-index from a game-event is not found in the demo.
// See https://github.com/markus-wa/demoinfocs-golang/issues/314
func IgnoreErrBombsiteIndexNotFound(ignore bool) Option {
	return func(c *config.Config) {
		c.IgnoreErrBombsiteIndexNotFound = ignore
	}
}

// NetMessageDecryptionKey tells the parser how to decrypt certain encrypted net-messages (e.g. chat messages in Valve matchmaking demos).
// See demoinfocs.MatchInfoDecryptionKey() on how to retrieve the key from `match730_*.dem.info` files.
func NetMessageDecryptionKey(key []byte) Option {
	return func(c *config.Config) {
		c.NetMessageDecryptionKey = key
	}
}

// DisableMimicSource1Events tells the parser to not mimic Source 1 game events for Source 2 demos.
// Unfortunately Source 2 demos *may* not contain Source 1 game events, that's why the parser will try to mimic them.
// Mimicking is enabled by default.
func DisableMimicSource1Events(disable bool) Option {
	return func(c *config.Config) {
		c.DisableMimicSource1Events = disable
	}
}

// Source2FallbackGameEventListBin sets a fallback game event list protobuf message for Source 2 demos.
// It's used when the game event list is not found in the demo file.
// This can happen due to a CS2 bug with sv_hibernate_when_empty.
func Source2FallbackGameEventListBin(bin []byte) Option {
	return func(c *config.Config) {
		c.Source2FallbackGameEventListBin = bin
	}
}

// IgnorePacketEntitiesPanic tells the parser to skip PacketEntities net-messages that can't be decoded
// instead of failing with a demoinfocs.DecodeError.
// This is required as a workaround for some POV demos that seem to contain rare PacketEntities parsing issues.
// The skipped messages are reported as ParserWarn events (WarnTypePacketEntitiesPanic) with the DecodeError in Err.
// Panics of handlers (e.g. entity callbacks) aren't ignored, see demoinfocs.PanicError.
func IgnorePacketEntitiesPanic(ignore bool) Option {
	return func(c *config.Config) {
		c.IgnorePacketEntitiesPanic = ignore
	}
}

// UserCmdParsing controls CSVCMsg_UserCommands parsing.
// It defaults to UserCmdParsingButtonsOnly.
func UserCmdParsing(mode UserCmdParsingMode) Option {
	return func(c *config.Config) {
		c.UserCmdParsing = mode
	}
}

// EagerVelocityTracking makes the parser record every player's position from the start on demos that
// don't network the player velocity (recorded before the CS2 "AnimGraph 2" update, network protocol
// < 14089).
//
// On those demos Player.Velocity() derives the velocity from position deltas for everybody but the
// recording player of POV demos, whose velocity is networked. Without this option, positions are only
// recorded for players Player.Velocity() has been called for, starting with that call (calls return a
// zero vector until the entity snapshot after it has been applied), and a ParserWarn
// (WarnTypeVelocityTrackingStartedLazily) is dispatched once. With it, the velocity is available for every
// player once their second entity snapshot has been applied, at the cost of recording positions for all
// players.
// It has no effect (and no cost) on newer demos, where the velocity is networked. Disabled by default.
func EagerVelocityTracking(eager bool) Option {
	return func(c *config.Config) {
		c.EagerVelocityTracking = eager
	}
}
