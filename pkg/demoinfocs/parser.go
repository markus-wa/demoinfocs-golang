package demoinfocs

import (
	"fmt"
	"io"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/geo/r3"
	dp "github.com/markus-wa/godispatch"
	"github.com/pkg/errors"

	bit "github.com/markus-wa/demoinfocs-golang/v2/internal/bitread"
	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	msg "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
)

//go:generate ifacemaker -f parser.go -f parsing.go -s parser -i Parser -p demoinfocs -D -y "Parser is an auto-generated interface for Parser, intended to be used when mockability is needed." -c "DO NOT EDIT: Auto generated" -o parser_interface.go

/*
Parser can parse a CS:GO demo.
Creating a new instance is done via NewParser().

To start off you may use Parser.ParseHeader() to parse the demo header
(this can be skipped and will be done automatically if necessary).
Further, Parser.ParseNextFrame() and Parser.ParseToEnd() can be used to parse the demo.

Use Parser.RegisterEventHandler() to receive notifications about events.

Example (without error handling):

	f, _ := os.Open("/path/to/demo.dem")
	p := dem.NewParser(f)
	defer p.Close()
	header := p.ParseHeader()
	fmt.Println("Map:", header.MapName)
	p.RegisterEventHandler(func(e events.BombExplode) {
		fmt.Printf(e.Site, "went BOOM!")
	})
	p.ParseToEnd()

Prints out '{A/B} site went BOOM!' when a bomb explodes.
*/
type parser struct {
	// Important fields

	bitReader                    *bit.BitReader
	stParser                     *st.SendTableParser
	additionalNetMessageCreators map[int]NetMessageCreator // Map of net-message-IDs to NetMessageCreators (for parsing custom net-messages)
	msgQueue                     chan interface{}          // Queue of net-messages
	msgDispatcher                *dp.Dispatcher            // Net-message dispatcher
	gameEventHandler             gameEventHandler
	userMessageHandler           userMessageHandler
	eventDispatcher              *dp.Dispatcher
	currentFrame                 int                // Demo-frame, not ingame-tick
	tickInterval                 float32            // Duration between ticks in seconds
	header                       *common.DemoHeader // Pointer so we can check for nil
	gameState                    *gameState
	demoInfoProvider             demoInfoProvider // Provides demo infos to other packages that the core package depends on
	err                          error            // Contains a error that occurred during parsing if any
	errLock                      sync.Mutex       // Used to sync up error mutations between parsing & handling go-routines

	// Additional fields, mainly caching & tracking things

	bombsiteA            bombsite
	bombsiteB            bombsite
	equipmentMapping     map[*st.ServerClass]common.EquipmentType        // Maps server classes to equipment-types
	rawPlayers           map[int]*playerInfo                             // Maps entity IDs to 'raw' player info
	modelPreCache        []string                                        // Used to find out whether a weapon is a p250 or cz for example (same id)
	triggers             map[int]*boundingBoxInformation                 // Maps entity IDs to triggers (used for bombsites)
	gameEventDescs       map[int32]*msg.CSVCMsg_GameEventListDescriptorT // Maps game-event IDs to descriptors
	grenadeModelIndices  map[int]common.EquipmentType                    // Used to map model indices to grenades (used for grenade projectiles)
	stringTables         []*msg.CSVCMsg_CreateStringTable                // Contains all created sendtables, needed when updating them
	delayedEventHandlers []func()                                        // Contains event handlers that need to be executed at the end of a tick (e.g. flash events because FlashDuration isn't updated before that)
}

// NetMessageCreator creates additional net-messages to be dispatched to net-message handlers.
//
// See also: ParserConfig.AdditionalNetMessageCreators & Parser.RegisterNetMessageHandler()
type NetMessageCreator func() proto.Message

type bombsite struct {
	index  int
	center r3.Vector
}

type boundingBoxInformation struct {
	min r3.Vector
	max r3.Vector
}

func (bbi boundingBoxInformation) contains(point r3.Vector) bool {
	return point.X >= bbi.min.X && point.X <= bbi.max.X &&
		point.Y >= bbi.min.Y && point.Y <= bbi.max.Y &&
		point.Z >= bbi.min.Z && point.Z <= bbi.max.Z
}

// ServerClasses returns the server-classes of this demo.
// These are available after events.DataTablesParsed has been fired.
func (p *parser) ServerClasses() st.ServerClasses {
	return p.stParser.ServerClasses()
}

// Header returns the DemoHeader which contains the demo's metadata.
// Only possible after ParserHeader() has been called.
func (p *parser) Header() common.DemoHeader {
	return *p.header
}

// GameState returns the current game-state.
// It contains most of the relevant information about the game such as players, teams, scores, grenades etc.
func (p *parser) GameState() GameState {
	return p.gameState
}

// CurrentFrame return the number of the current frame, aka. 'demo-tick' (Since demos often have a different tick-rate than the game).
// Starts with frame 0, should go up to DemoHeader.PlaybackFrames but might not be the case (usually it's just close to it).
func (p *parser) CurrentFrame() int {
	return p.currentFrame
}

// CurrentTime returns the time elapsed since the start of the demo
func (p *parser) CurrentTime() time.Duration {
	return time.Duration(float32(p.gameState.ingameTick) * p.tickInterval * float32(time.Second))
}

// TickRate returns the tick-rate the server ran on during the game.
//
// Returns tick rate based on CSVCMsg_ServerInfo if possible.
// Otherwise returns tick rate based on demo header or -1 if the header info isn't available.
func (p *parser) TickRate() float64 {
	if p.tickInterval != 0 {
		return 1.0 / float64(p.tickInterval)
	}

	if p.header != nil {
		return legacyTickRate(*p.header)
	}

	return -1
}

func legacyTickRate(h common.DemoHeader) float64 {
	if h.PlaybackTime == 0 {
		return 0
	}

	return float64(h.PlaybackTicks) / h.PlaybackTime.Seconds()
}

// TickTime returns the time a single tick takes in seconds.
//
// Returns tick time based on CSVCMsg_ServerInfo if possible.
// Otherwise returns tick time based on demo header or -1 if the header info isn't available.
func (p *parser) TickTime() time.Duration {
	if p.tickInterval != 0 {
		return time.Duration(float32(time.Second) * p.tickInterval)
	}

	if p.header != nil {
		return legayTickTime(*p.header)
	}

	return -1
}

func legayTickTime(h common.DemoHeader) time.Duration {
	if h.PlaybackTicks == 0 {
		return 0
	}

	return time.Duration(h.PlaybackTime.Nanoseconds() / int64(h.PlaybackTicks))
}

// Progress returns the parsing progress from 0 to 1.
// Where 0 means nothing has been parsed yet and 1 means the demo has been parsed to the end.
//
// Might not be 100% correct since it's just based on the reported tick count of the header.
// May always return 0 if the demo header is corrupt.
func (p *parser) Progress() float32 {
	if p.header == nil || p.header.PlaybackFrames == 0 {
		return 0
	}

	return float32(p.currentFrame) / float32(p.header.PlaybackFrames)
}

/*
RegisterEventHandler registers a handler for game events.

The handler must be of type func(<EventType>) where EventType is the kind of event to be handled.
To catch all events func(interface{}) can be used.

Example:

	parser.RegisterEventHandler(func(e events.WeaponFired) {
		fmt.Printf("%s fired his %s\n", e.Shooter.Name, e.Weapon.Type)
	})

Parameter handler has to be of type interface{} because lolnogenerics.

Returns a identifier with which the handler can be removed via UnregisterEventHandler().
*/
func (p *parser) RegisterEventHandler(handler interface{}) dp.HandlerIdentifier {
	return p.eventDispatcher.RegisterHandler(handler)
}

// UnregisterEventHandler removes a game event handler via identifier.
//
// The identifier is returned at registration by RegisterEventHandler().
func (p *parser) UnregisterEventHandler(identifier dp.HandlerIdentifier) {
	p.eventDispatcher.UnregisterHandler(identifier)
}

/*
RegisterNetMessageHandler registers a handler for net-messages.

The handler must be of type func(*<MessageType>) where MessageType is the kind of net-message to be handled.

Returns a identifier with which the handler can be removed via UnregisterNetMessageHandler().

See also: RegisterEventHandler()
*/
func (p *parser) RegisterNetMessageHandler(handler interface{}) dp.HandlerIdentifier {
	return p.msgDispatcher.RegisterHandler(handler)
}

// UnregisterNetMessageHandler removes a net-message handler via identifier.
//
// The identifier is returned at registration by RegisterNetMessageHandler().
func (p *parser) UnregisterNetMessageHandler(identifier dp.HandlerIdentifier) {
	p.msgDispatcher.UnregisterHandler(identifier)
}

// Close closes any open resources used by the Parser (go routines, file handles).
// This must be called before discarding the Parser to avoid memory leaks.
// Returns an error if closing of underlying resources fails.
func (p *parser) Close() error {
	p.msgDispatcher.RemoveAllQueues()

	if p.bitReader != nil {
		err := p.bitReader.Close()
		if err != nil {
			return errors.Wrap(err, "failed to close BitReader")
		}
	}

	return nil
}

func (p *parser) error() error {
	p.errLock.Lock()
	err := p.err
	p.errLock.Unlock()

	return err
}

func (p *parser) setError(err error) {
	if err == nil {
		return
	}

	p.errLock.Lock()

	if p.err != nil {
		p.errLock.Unlock()

		return
	}

	p.err = err

	p.errLock.Unlock()
}

// NewParser creates a new Parser with the default configuration.
// The demostream io.Reader (e.g. os.File or bytes.Reader) must provide demo data in the '.DEM' format.
//
// See also: NewCustomParser() & DefaultParserConfig
func NewParser(demostream io.Reader) Parser {
	return NewParserWithConfig(demostream, DefaultParserConfig)
}

// ParserConfig contains the configuration for creating a new Parser.
type ParserConfig struct {
	// MsgQueueBufferSize defines the size of the internal net-message queue.
	// For large demos, fast i/o and slow CPUs higher numbers are suggested and vice versa.
	// The buffer size can easily be in the hundred-thousands to low millions for the best performance.
	// A negative value will make the Parser automatically decide the buffer size during ParseHeader()
	// based on the number of ticks in the demo (nubmer of ticks = buffer size);
	// this is the default behavior for DefaultParserConfig.
	// Zero enforces sequential parsing.
	MsgQueueBufferSize int

	// AdditionalNetMessageCreators maps net-message-IDs to creators (instantiators).
	// The creators should return a new instance of the correct protobuf-message type (from the msg package).
	// Interesting net-message-IDs can easily be discovered with the build-tag 'debugdemoinfocs'; when looking for 'UnhandledMessage'.
	// Check out parsing.go to see which net-messages are already being parsed by default.
	// This is a beta feature and may be changed or replaced without notice.
	AdditionalNetMessageCreators map[int]NetMessageCreator
}

// DefaultParserConfig is the default Parser configuration used by NewParser().
var DefaultParserConfig = ParserConfig{
	MsgQueueBufferSize: -1,
}

// NewParserWithConfig returns a new Parser with a custom configuration.
//
// See also: NewParser() & ParserConfig
func NewParserWithConfig(demostream io.Reader, config ParserConfig) Parser {
	var p parser

	// Init parser
	p.bitReader = bit.NewLargeBitReader(demostream)
	p.stParser = st.NewSendTableParser()
	p.equipmentMapping = make(map[*st.ServerClass]common.EquipmentType)
	p.rawPlayers = make(map[int]*playerInfo)
	p.triggers = make(map[int]*boundingBoxInformation)
	p.demoInfoProvider = demoInfoProvider{parser: &p}
	p.gameState = newGameState(p.demoInfoProvider)
	p.grenadeModelIndices = make(map[int]common.EquipmentType)
	p.gameEventHandler = newGameEventHandler(&p)
	p.userMessageHandler = newUserMessageHandler(&p)

	dispatcherCfg := dp.Config{
		PanicHandler: func(v interface{}) {
			p.setError(fmt.Errorf("%v\nstacktrace:\n%s", v, debug.Stack()))
		},
	}
	p.msgDispatcher = dp.NewDispatcherWithConfig(dispatcherCfg)
	p.eventDispatcher = dp.NewDispatcherWithConfig(dispatcherCfg)

	// Attach proto msg handlers
	p.msgDispatcher.RegisterHandler(p.handlePacketEntities)
	p.msgDispatcher.RegisterHandler(p.handleGameEventList)
	p.msgDispatcher.RegisterHandler(p.handleGameEvent)
	p.msgDispatcher.RegisterHandler(p.handleCreateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUpdateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUserMessage)
	p.msgDispatcher.RegisterHandler(p.handleSetConVar)
	p.msgDispatcher.RegisterHandler(p.handleFrameParsed)
	p.msgDispatcher.RegisterHandler(p.handleServerInfo)
	p.msgDispatcher.RegisterHandler(p.gameState.handleIngameTickNumber)

	if config.MsgQueueBufferSize >= 0 {
		p.initMsgQueue(config.MsgQueueBufferSize)
	}

	p.additionalNetMessageCreators = config.AdditionalNetMessageCreators

	return &p
}

func (p *parser) initMsgQueue(buf int) {
	p.msgQueue = make(chan interface{}, buf)
	p.msgDispatcher.AddQueues(p.msgQueue)
}

type demoInfoProvider struct {
	parser *parser
}

func (p demoInfoProvider) IngameTick() int {
	return p.parser.gameState.IngameTick()
}

func (p demoInfoProvider) TickRate() float64 {
	return p.parser.TickRate()
}

func (p demoInfoProvider) FindPlayerByHandle(handle int) *common.Player {
	return p.parser.gameState.Participants().FindByHandle(handle)
}

func (p demoInfoProvider) PlayerResourceEntity() st.Entity {
	return p.parser.gameState.playerResourceEntity
}

func (p demoInfoProvider) FindWeaponByEntityID(entityID int) *common.Equipment {
	return p.parser.gameState.weapons[entityID]
}
