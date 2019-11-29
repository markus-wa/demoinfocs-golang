package demoinfocs

import (
	"io"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/geo/r3"
	dp "github.com/markus-wa/godispatch"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/msg"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

//go:generate ifacemaker -f parser.go -f parsing.go -s Parser -i IParser -p demoinfocs -D -y "IParser is an auto-generated interface for Parser, intended to be used when mockability is needed." -c "DO NOT EDIT: Auto generated" -o parser_interface.go

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
	header := p.ParseHeader()
	fmt.Println("Map:", header.MapName)
	p.RegisterEventHandler(func(e events.BombExplode) {
		fmt.Printf(e.Site, "went BOOM!")
	})
	p.ParseToEnd()

Prints out '{A/B} site went BOOM!' when a bomb explodes.
*/
type Parser struct {
	// Important fields

	bitReader                    *bit.BitReader
	stParser                     *st.SendTableParser
	additionalNetMessageCreators map[int]NetMessageCreator // Map of net-message-IDs to NetMessageCreators (for parsing custom net-messages)
	msgQueue                     chan interface{}          // Queue of net-messages
	msgDispatcher                dp.Dispatcher             // Net-message dispatcher
	gameEventHandler             gameEventHandler
	userMessageHandler           userMessageHandler
	eventDispatcher              dp.Dispatcher
	currentFrame                 int                // Demo-frame, not ingame-tick
	header                       *common.DemoHeader // Pointer so we can check for nil
	gameState                    *GameState
	demoInfoProvider             demoInfoProvider // Provides demo infos to other packages that the core package depends on
	cancelChan                   chan struct{}    // Non-anime-related, used for aborting the parsing
	err                          error            // Contains a error that occurred during parsing if any
	errLock                      sync.Mutex       // Used to sync up error mutations between parsing & handling go-routines

	// Additional fields, mainly caching & tracking things

	bombsiteA            bombsite
	bombsiteB            bombsite
	equipmentMapping     map[*st.ServerClass]common.EquipmentElement     // Maps server classes to equipment-types
	rawPlayers           map[int]*playerInfo                             // Maps entity IDs to 'raw' player info
	additionalPlayerInfo [maxPlayers]common.AdditionalPlayerInformation  // Maps entity IDs to additional player info (scoreboard info)
	modelPreCache        []string                                        // Used to find out whether a weapon is a p250 or cz for example (same id)
	weapons              [maxEntities]common.Equipment                   // Used to remember what a weapon is (p250 / cz etc.)
	triggers             map[int]*boundingBoxInformation                 // Maps entity IDs to triggers (used for bombsites)
	gameEventDescs       map[int32]*msg.CSVCMsg_GameEventListDescriptorT // Maps game-event IDs to descriptors
	grenadeModelIndices  map[int]common.EquipmentElement                 // Used to map model indices to grenades (used for grenade projectiles)
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
func (p *Parser) ServerClasses() st.ServerClasses {
	return p.stParser.ServerClasses()
}

// Header returns the DemoHeader which contains the demo's metadata.
// Only possible after ParserHeader() has been called.
func (p *Parser) Header() common.DemoHeader {
	return *p.header
}

// GameState returns the current game-state.
// It contains most of the relevant information about the game such as players, teams, scores, grenades etc.
func (p *Parser) GameState() IGameState {
	return p.gameState
}

// CurrentFrame return the number of the current frame, aka. 'demo-tick' (Since demos often have a different tick-rate than the game).
// Starts with frame 0, should go up to DemoHeader.PlaybackFrames but might not be the case (usually it's just close to it).
func (p *Parser) CurrentFrame() int {
	return p.currentFrame
}

// CurrentTime returns the time elapsed since the start of the demo
func (p *Parser) CurrentTime() time.Duration {
	return time.Duration(p.currentFrame) * p.header.FrameTime()
}

// Progress returns the parsing progress from 0 to 1.
// Where 0 means nothing has been parsed yet and 1 means the demo has been parsed to the end.
//
// Might not be 100% correct since it's just based on the reported tick count of the header.
func (p *Parser) Progress() float32 {
	return float32(p.currentFrame) / float32(p.header.PlaybackFrames)
}

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
func (p *Parser) RegisterEventHandler(handler interface{}) dp.HandlerIdentifier {
	return p.eventDispatcher.RegisterHandler(handler)
}

// UnregisterEventHandler removes a game event handler via identifier.
//
// The identifier is returned at registration by RegisterEventHandler().
func (p *Parser) UnregisterEventHandler(identifier dp.HandlerIdentifier) {
	p.eventDispatcher.UnregisterHandler(identifier)
}

/*
RegisterNetMessageHandler registers a handler for net-messages.

The handler must be of type func(*<MessageType>) where MessageType is the kind of net-message to be handled.

Returns a identifier with which the handler can be removed via UnregisterNetMessageHandler().

See also: RegisterEventHandler()
*/
func (p *Parser) RegisterNetMessageHandler(handler interface{}) dp.HandlerIdentifier {
	return p.msgDispatcher.RegisterHandler(handler)
}

// UnregisterNetMessageHandler removes a net-message handler via identifier.
//
// The identifier is returned at registration by RegisterNetMessageHandler().
func (p *Parser) UnregisterNetMessageHandler(identifier dp.HandlerIdentifier) {
	p.msgDispatcher.UnregisterHandler(identifier)
}

func (p *Parser) error() (err error) {
	p.errLock.Lock()
	err = p.err
	p.errLock.Unlock()
	return
}

func (p *Parser) setError(err error) {
	if err != nil {
		p.errLock.Lock()
		p.err = err
		p.errLock.Unlock()
	}
}

// NewParser creates a new Parser with the default configuration.
// The demostream io.Reader (e.g. os.File or bytes.Reader) must provide demo data in the '.DEM' format.
//
// See also: NewCustomParser() & DefaultParserConfig
func NewParser(demostream io.Reader) *Parser {
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
func NewParserWithConfig(demostream io.Reader, config ParserConfig) *Parser {
	var p Parser

	// Init parser
	p.bitReader = bit.NewLargeBitReader(demostream)
	p.stParser = st.NewSendTableParser()
	p.equipmentMapping = make(map[*st.ServerClass]common.EquipmentElement)
	p.rawPlayers = make(map[int]*playerInfo)
	p.triggers = make(map[int]*boundingBoxInformation)
	p.cancelChan = make(chan struct{}, 1)
	p.gameState = newGameState()
	p.grenadeModelIndices = make(map[int]common.EquipmentElement)
	p.gameEventHandler = newGameEventHandler(&p)
	p.userMessageHandler = newUserMessageHandler(&p)
	p.demoInfoProvider = demoInfoProvider{parser: &p}

	// Attach proto msg handlers
	p.msgDispatcher.RegisterHandler(p.handlePacketEntities)
	p.msgDispatcher.RegisterHandler(p.handleGameEventList)
	p.msgDispatcher.RegisterHandler(p.handleGameEvent)
	p.msgDispatcher.RegisterHandler(p.handleCreateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUpdateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUserMessage)
	p.msgDispatcher.RegisterHandler(p.handleSetConVar)
	p.msgDispatcher.RegisterHandler(p.handleFrameParsed)
	p.msgDispatcher.RegisterHandler(p.gameState.handleIngameTickNumber)

	if config.MsgQueueBufferSize >= 0 {
		p.initMsgQueue(config.MsgQueueBufferSize)
	}

	p.additionalNetMessageCreators = config.AdditionalNetMessageCreators

	return &p
}

func (p *Parser) initMsgQueue(buf int) {
	p.msgQueue = make(chan interface{}, buf)
	p.msgDispatcher.AddQueues(p.msgQueue)
}

type demoInfoProvider struct {
	parser *Parser
}

func (p demoInfoProvider) IngameTick() int {
	return p.parser.gameState.IngameTick()
}

func (p demoInfoProvider) TickRate() float64 {
	// TODO: read tickRate from CVARs as fallback
	return p.parser.header.TickRate()
}

func (p demoInfoProvider) FindPlayerByHandle(handle int) *common.Player {
	return p.parser.gameState.Participants().FindByHandle(handle)
}
