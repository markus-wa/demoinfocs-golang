package demoinfocs

import (
	"net/http"
	"time"
)

// demoFormat is the format of the demo stream (".dem" file or live CSTV broadcast).
// It's set internally: NewCSTVBroadcastParser() selects the CSTV broadcast format.
type demoFormat byte

const (
	demoFormatFile demoFormat = iota
	demoFormatCSTVBroadcast
)

// UserCmdParsingMode controls how CSVCMsg_UserCommands messages are handled.
type UserCmdParsingMode uint8

const (
	// UserCmdParsingButtonsOnly tracks buttonstate1 for PlayerButtonsStateUpdate events.
	// It is the default.
	UserCmdParsingButtonsOnly UserCmdParsingMode = iota
	// UserCmdParsingFull reconstructs and dispatches complete UserCmd events.
	UserCmdParsingFull
	// UserCmdParsingDisabled skips user-command parsing and preserves legacy properties.
	UserCmdParsingDisabled
)

// config contains the configuration of a Parser, see the With*() options.
type config struct {
	MsgQueueBufferSize              int
	AdditionalNetMessageCreators    map[int]NetMessageCreator
	IgnoreErrBombsiteIndexNotFound  bool
	NetMessageDecryptionKey         []byte
	DisableMimicSource1Events       bool
	Source2FallbackGameEventListBin []byte
	IgnorePacketEntitiesPanic       bool
	Format                          demoFormat
	CSTVTimeout                     time.Duration
	HTTPClient                      *http.Client
	UserCmdParsing                  UserCmdParsingMode
	EagerVelocityTracking           bool
}

// defaultConfig returns the configuration options are applied on top of.
func defaultConfig() config {
	return config{
		MsgQueueBufferSize: -1,
		CSTVTimeout:        10 * time.Second,
		UserCmdParsing:     UserCmdParsingButtonsOnly,
	}
}

func newConfig(opts []Option) config {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// Option configures a Parser, see the With*() functions.
type Option func(*config)

// WithMsgQueueBufferSize sets the size of the internal net-message queue.
// For large demos, fast i/o and slow CPUs higher numbers are suggested and vice versa.
// The buffer size can easily be in the hundred-thousands to low millions for the best performance.
// A negative value (the default) makes the Parser automatically decide the buffer size when parsing the header,
// based on the number of ticks in the demo (number of ticks = buffer size).
// Zero enforces sequential parsing.
func WithMsgQueueBufferSize(size int) Option {
	return func(c *config) {
		c.MsgQueueBufferSize = size
	}
}

// WithAdditionalNetMessageCreators maps net-message-IDs to creators (instantiators).
// The creators should return a new instance of the correct protobuf-message type (from the msg package).
// Interesting net-message-IDs can easily be discovered with the build-tag 'debugdemoinfocs'; when looking for 'UnhandledMessage'.
// Check out parsing.go to see which net-messages are already being parsed by default.
// This is a beta feature and may be changed or replaced without notice.
func WithAdditionalNetMessageCreators(creators map[int]NetMessageCreator) Option {
	return func(c *config) {
		c.AdditionalNetMessageCreators = creators
	}
}

// WithIgnoreErrBombsiteIndexNotFound tells the parser to not return an error when a bombsite-index from a game-event is not found in the demo.
// See https://github.com/markus-wa/demoinfocs-golang/issues/314
func WithIgnoreErrBombsiteIndexNotFound(ignore bool) Option {
	return func(c *config) {
		c.IgnoreErrBombsiteIndexNotFound = ignore
	}
}

// WithNetMessageDecryptionKey tells the parser how to decrypt certain encrypted net-messages (e.g. chat messages in Valve matchmaking demos).
// See MatchInfoDecryptionKey() on how to retrieve the key from `match730_*.dem.info` files.
func WithNetMessageDecryptionKey(key []byte) Option {
	return func(c *config) {
		c.NetMessageDecryptionKey = key
	}
}

// WithDisableMimicSource1Events tells the parser to not mimic Source 1 game events for Source 2 demos.
// Unfortunately Source 2 demos *may* not contain Source 1 game events, that's why the parser will try to mimic them.
// Mimicking is enabled by default.
func WithDisableMimicSource1Events(disable bool) Option {
	return func(c *config) {
		c.DisableMimicSource1Events = disable
	}
}

// WithSource2FallbackGameEventListBin sets a fallback game event list protobuf message for Source 2 demos.
// It's used when the game event list is not found in the demo file.
// This can happen due to a CS2 bug with sv_hibernate_when_empty.
func WithSource2FallbackGameEventListBin(bin []byte) Option {
	return func(c *config) {
		c.Source2FallbackGameEventListBin = bin
	}
}

// WithIgnorePacketEntitiesPanic tells the parser to ignore PacketEntities parsing panics.
// This is required as a workaround for some POV demos that seem to contain rare PacketEntities parsing issues.
// The ignored panics are reported as ParserWarn events (WarnTypePacketEntitiesPanic).
func WithIgnorePacketEntitiesPanic(ignore bool) Option {
	return func(c *config) {
		c.IgnorePacketEntitiesPanic = ignore
	}
}

// WithUserCmdParsing controls CSVCMsg_UserCommands parsing.
// It defaults to UserCmdParsingButtonsOnly.
func WithUserCmdParsing(mode UserCmdParsingMode) Option {
	return func(c *config) {
		c.UserCmdParsing = mode
	}
}

// WithEagerVelocityTracking makes the parser record every player's position from the start on demos that
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
func WithEagerVelocityTracking(eager bool) Option {
	return func(c *config) {
		c.EagerVelocityTracking = eager
	}
}

// WithCSTVTimeout sets the timeout for CSTV broadcasts (10s by default).
// It's the maximum time to retry for a response from the CSTV server, using an exponential backoff mechanism, starting at 1s.
// Only used by NewCSTVBroadcastParser() and ParseCSTVBroadcast().
func WithCSTVTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.CSTVTimeout = timeout
	}
}

// WithHTTPClient sets the HTTP client used to fetch CSTV broadcast fragments (http.DefaultClient by default).
// Only used by NewCSTVBroadcastParser() and ParseCSTVBroadcast().
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) {
		c.HTTPClient = client
	}
}
