// DO NOT EDIT: Auto generated

package demoinfocs

import (
	"time"

	"github.com/markus-wa/demoinfocs-golang/common"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
	dp "github.com/markus-wa/godispatch"
)

// IParser is an auto-generated interface for Parser, intended to be used when mockability is needed.
// Parser can parse a CS:GO demo.
// Creating a new instance is done via NewParser().
//
// To start off you may use Parser.ParseHeader() to parse the demo header
// (this can be skipped and will be done automatically if necessary).
// Further, Parser.ParseNextFrame() and Parser.ParseToEnd() can be used to parse the demo.
//
// Use Parser.RegisterEventHandler() to receive notifications about events.
//
// Example (without error handling):
//
// 	f, _ := os.Open("/path/to/demo.dem")
// 	p := dem.NewParser(f)
// 	header := p.ParseHeader()
// 	fmt.Println("Map:", header.MapName)
// 	p.RegisterEventHandler(func(e events.BombExplode) {
// 		fmt.Printf(e.Site, "went BOOM!")
// 	})
// 	p.ParseToEnd()
//
// Prints out '{A/B} site went BOOM!' when a bomb explodes.
type IParser interface {
	// ServerClasses returns the server-classes of this demo.
	// These are available after events.DataTablesParsed has been fired.
	ServerClasses() st.ServerClasses
	// Header returns the DemoHeader which contains the demo's metadata.
	// Only possible after ParserHeader() has been called.
	Header() common.DemoHeader
	// GameState returns the current game-state.
	// It contains most of the relevant information about the game such as players, teams, scores, grenades etc.
	GameState() IGameState
	// CurrentFrame return the number of the current frame, aka. 'demo-tick' (Since demos often have a different tick-rate than the game).
	// Starts with frame 0, should go up to DemoHeader.PlaybackFrames but might not be the case (usually it's just close to it).
	CurrentFrame() int
	// CurrentTime returns the time elapsed since the start of the demo
	CurrentTime() time.Duration
	// Progress returns the parsing progress from 0 to 1.
	// Where 0 means nothing has been parsed yet and 1 means the demo has been parsed to the end.
	//
	// Might not be 100% correct since it's just based on the reported tick count of the header.
	Progress() float32
	/*
	   RegisterEventHandler registers a handler for game events.

	   The handler must be of type func(<EventType>) where EventType is the kind of event to be handled.
	   To catch all events func(interface{}) can be used.

	   Example:

	   	parser.RegisterEventHandler(func(e events.WeaponFired) {
	   		fmt.Printf("%s fired his %s\n", e.Shooter.Name, e.Weapon.Weapon)
	   	})

	   Parameter handler has to be of type interface{} because lolnogenerics.

	   Returns a identifier with which the handler can be removed via UnregisterEventHandler().
	*/
	RegisterEventHandler(handler interface{}) dp.HandlerIdentifier
	// UnregisterEventHandler removes a game event handler via identifier.
	//
	// The identifier is returned at registration by RegisterEventHandler().
	UnregisterEventHandler(identifier dp.HandlerIdentifier)
	/*
	   RegisterNetMessageHandler registers a handler for net-messages.

	   The handler must be of type func(*<MessageType>) where MessageType is the kind of net-message to be handled.

	   Returns a identifier with which the handler can be removed via UnregisterNetMessageHandler().

	   See also: RegisterEventHandler()
	*/
	RegisterNetMessageHandler(handler interface{}) dp.HandlerIdentifier
	// UnregisterNetMessageHandler removes a net-message handler via identifier.
	//
	// The identifier is returned at registration by RegisterNetMessageHandler().
	UnregisterNetMessageHandler(identifier dp.HandlerIdentifier)
	// ParseHeader attempts to parse the header of the demo and returns it.
	// If not done manually this will be called by Parser.ParseNextFrame() or Parser.ParseToEnd().
	//
	// Returns ErrInvalidFileType if the filestamp (first 8 bytes) doesn't match HL2DEMO.
	ParseHeader() (common.DemoHeader, error)
	// ParseToEnd attempts to parse the demo until the end.
	// Aborts and returns ErrCancelled if Cancel() is called before the end.
	//
	// See also: ParseNextFrame() for other possible errors.
	ParseToEnd() (err error)
	// Cancel aborts ParseToEnd().
	// All information that was already read up to this point may still be used (and new events may still be sent out).
	Cancel()
	/*
	   ParseNextFrame attempts to parse the next frame / demo-tick (not ingame tick).

	   Returns true unless the demo command 'stop' or an error was encountered.

	   May return ErrUnexpectedEndOfDemo for incomplete / corrupt demos.
	   May panic if the demo is corrupt in some way.

	   See also: ParseToEnd() for parsing the complete demo in one go (faster).
	*/
	ParseNextFrame() (moreFrames bool, err error)
}
