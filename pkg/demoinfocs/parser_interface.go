// DO NOT EDIT: Auto generated

package demoinfocs

import (
	"context"
	"time"

	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
	dp "github.com/markus-wa/godispatch"
)

// Parser is an auto-generated interface for Parser, intended to be used when mockability is needed.
// Parser can parse a CS2 demo.
// Creating a new instance is done via NewParser().
//
// Parser.ParseNextFrame() and Parser.ParseToEnd() can be used to parse the demo
// (the demo header is parsed automatically when parsing starts).
//
// Use Parser.RegisterEventHandler() to receive notifications about events.
//
// Example (without error handling):
//
//	f, _ := os.Open("/path/to/demo.dem")
//	p := dem.NewParser(f)
//	defer p.Close()
//	p.RegisterEventHandler(func(e events.BombExplode) {
//		fmt.Printf("%c site went BOOM!\n", e.Site)
//	})
//	p.ParseToEnd(context.Background())
//
// Prints out '{A/B} site went BOOM!' when a bomb explodes.
type Parser interface {
	// ServerClasses returns the server-classes of this demo.
	// These are available after events.DataTablesParsed has been fired.
	ServerClasses() st.ServerClasses
	// GameState returns the current game-state.
	// It contains most of the relevant information about the game such as players, teams, scores, grenades etc.
	GameState() GameState
	// CurrentFrame return the number of the current frame, aka. 'demo-tick' (Since demos often have a different tick-rate than the game).
	// Starts with frame 0, should go up to header.PlaybackFrames but might not be the case (usually it's just close to it).
	CurrentFrame() int
	// CurrentTime returns the time elapsed since the start of the demo
	CurrentTime() time.Duration
	// TickRate returns the tick-rate the server ran on during the game.
	//
	// Returns tick rate based on CSVCMsg_ServerInfo if possible.
	// Otherwise returns tick rate based on demo header or -1 if the header info isn't available.
	TickRate() float64
	// TickTime returns the time a single tick takes in seconds.
	//
	// Returns tick time based on CSVCMsg_ServerInfo if possible.
	// Otherwise returns tick time based on demo header or -1 if the header info isn't available.
	TickTime() time.Duration
	// Progress returns the parsing progress from 0 to 1.
	// Where 0 means nothing has been parsed yet and 1 means the demo has been parsed to the end.
	//
	// Might not be 100% correct since it's just based on the reported tick count of the header.
	// May always return 0 if the demo header is corrupt.
	Progress() float32
	// FrameCount returns the total number of frames in the demo, or -1 if it is not yet known.
	//
	// For Source 2 (CS2) demos the total only becomes available once the trailing CDemoFileInfo has
	// been parsed - typically at the very end of ParseToEnd - and is absent from the pre-parse header,
	// so this returns -1 throughout parsing for those demos. Use it to branch cleanly rather than
	// relying on Progress() (which returns 0 while the total is unknown).
	FrameCount() int
	/*
	   RegisterEventHandler registers a handler for game events.

	   The handler must be of type func(<EventType>) where EventType is the kind of event to be handled.
	   To catch all events func(any) can be used.

	   Example:

	   	parser.RegisterEventHandler(func(e events.WeaponFired) {
	   		fmt.Printf("%s fired his %s\n", e.Shooter.Name, e.Weapon.Type)
	   	})

	   Parameter handler has to be of type any because Go generics only work on functions, not methods.

	   Returns an identifier with which the handler can be removed via UnregisterEventHandler().
	*/
	RegisterEventHandler(handler any) dp.HandlerIdentifier
	// UnregisterEventHandler removes a game event handler via identifier.
	//
	// The identifier is returned at registration by RegisterEventHandler().
	UnregisterEventHandler(identifier dp.HandlerIdentifier)
	/*
	   RegisterNetMessageHandler registers a handler for net-messages.

	   The handler must be of type func(*<MessageType>) where MessageType is the kind of net-message to be handled.

	   Parameter handler has to be of type any because Go generics only work on functions, not methods.

	   Returns an identifier with which the handler can be removed via UnregisterNetMessageHandler().

	   See also: RegisterEventHandler()
	*/
	RegisterNetMessageHandler(handler any) dp.HandlerIdentifier
	// UnregisterNetMessageHandler removes a net-message handler via identifier.
	//
	// The identifier is returned at registration by RegisterNetMessageHandler().
	UnregisterNetMessageHandler(identifier dp.HandlerIdentifier)
	// Close closes any open resources used by the Parser (go routines, file handles).
	// This must be called before discarding the Parser to avoid memory leaks.
	// Returns an error if closing of underlying resources fails.
	Close() error
	// ParseToEnd attempts to parse the demo until the end.
	//
	// Parsing is aborted when ctx is done (it may be cancelled from a handler or another goroutine):
	// no further events are dispatched, all handlers are unregistered, net-messages that are already
	// queued are dropped and context.Cause(ctx) is returned, so errors.Is(err, context.Canceled)
	// and errors.Is(err, context.DeadlineExceeded) can be used to detect it.
	// If ctx is already done, ParseToEnd returns right away without parsing anything.
	//
	// See also: ParseNextFrame() for other possible errors.
	ParseToEnd(ctx context.Context) (err error)
	/*
	   ParseNextFrame attempts to parse the next frame / demo-tick (not ingame tick).

	   Returns true unless the demo command 'stop' or an error was encountered.

	   May return ErrUnexpectedEndOfDemo for incomplete / corrupt demos.
	   May panic if the demo is corrupt in some way.

	   ParseNextFrame doesn't take a context.Context: callers that need cancellation
	   should check their context between frames and stop calling ParseNextFrame once it's done.

	   See also: ParseToEnd() for parsing the complete demo in one go (faster).
	*/
	ParseNextFrame() (moreFrames bool, err error)
}
