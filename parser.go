package demoinfocs

import (
	"io"
	"sync"

	r3 "github.com/golang/geo/r3"
	dp "github.com/markus-wa/godispatch"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	common "github.com/markus-wa/demoinfocs-golang/common"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// Parser can parse a CS:GO demo.
// Creating a Parser is done via NewParser().
// To start off use Parser.ParseHeader() to parse the demo header.
// After parsing the header Parser.ParseNextFrame() and Parser.ParseToEnd() can be used to parse the demo.
// Use Parser.RegisterEventHandler() to receive notifications about events.
type Parser struct {
	bitReader                    *bit.BitReader
	stParser                     st.SendTableParser
	msgDispatcher                dp.Dispatcher
	additionalNetMessageCreators map[int]NetMessageCreator
	eventDispatcher              dp.Dispatcher
	msgQueue                     chan interface{}
	gameState                    GameState
	currentFrame                 int
	bombsiteA                    bombsite
	bombsiteB                    bombsite
	header                       *common.DemoHeader // Pointer so we can check for nil
	equipmentMapping             map[*st.ServerClass]common.EquipmentElement
	rawPlayers                   map[int]*playerInfo
	entityIDToPlayers            map[int]*common.Player // Temporary storage since we need to map players from entityID to userID later
	additionalPlayerInfo         [maxPlayers]common.AdditionalPlayerInformation
	entities                     map[int]*st.Entity
	modelPreCache                []string                      // Used to find out whether a weapon is a p250 or cz for example (same id)
	weapons                      [maxEntities]common.Equipment // Used to remember what a weapon is (p250 / cz etc.)
	triggers                     map[int]*boundingBoxInformation
	instanceBaselines            map[int][]byte
	preprocessedBaselines        map[int]map[int]st.PropValue
	gameEventDescs               map[int32]*msg.CSVCMsg_GameEventListDescriptorT
	stringTables                 []*msg.CSVCMsg_CreateStringTable
	cancelChan                   chan struct{}
	err                          error
	errLock                      sync.Mutex
}

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

// SendTableParser returns the sendtable parser.
// This is a beta feature and may be changed or replaced without notice.
func (p *Parser) SendTableParser() *st.SendTableParser {
	return &p.stParser
}

// Header returns the DemoHeader which contains the demo's metadata.
func (p *Parser) Header() common.DemoHeader {
	return *p.header
}

// GameState returns the current game-state
func (p *Parser) GameState() *GameState {
	return &p.gameState
}

// Entities returns the available entities
func (p *Parser) Entities() *map[int]*st.Entity {
	return &p.entities
}

// CurrentFrame return the number of the current frame, aka. 'demo-tick' (Since demos often have a different tick-rate than the game).
// Starts with frame 0, should go up to DemoHeader.PlaybackFrames but might not be the case (usually it's just close to it).
func (p *Parser) CurrentFrame() int {
	return p.currentFrame
}

// CurrentTime returns the ingame time in seconds since the start of the demo.
func (p *Parser) CurrentTime() float32 {
	return float32(p.currentFrame) * p.header.FrameTime()
}

// Progress returns the parsing progress from 0 to 1.
// Where 0 means nothing has been parsed yet and 1 means the demo has been parsed to the end.
// Might not actually be reliable since it's just based on the reported tick count of the header.
func (p *Parser) Progress() float32 {
	return float32(p.currentFrame) / float32(p.header.PlaybackFrames)
}

// RegisterEventHandler registers a handler for game events.
// Must be of type func(<EventType>) where EventType is the kind of event that is handled.
// To catch all events func(interface{}) can be used.
// Parameter handler has to be of type interface{} because lolnogenerics.
// Returns a identifier with which the handler can be removed via UnregisterEventHandler().
func (p *Parser) RegisterEventHandler(handler interface{}) dp.HandlerIdentifier {
	return p.eventDispatcher.RegisterHandler(handler)
}

// UnregisterEventHandler removes a game event handler via identifier.
// The identifier is returned at registration by RegisterEventHandler().
func (p *Parser) UnregisterEventHandler(identifier dp.HandlerIdentifier) {
	p.eventDispatcher.UnregisterHandler(identifier)
}

// RegisterNetMessageHandler registers a handler for net-messages.
// Must be of type func(*<EventType>) where EventType is the kind of event that is handled.
// To catch all events func(interface{}) can be used.
// Parameter handler has to be of type interface{} because lolnogenerics.
// Returns a identifier with which the handler can be removed via UnregisterNetMessageHandler().
// This is a beta feature and may be changed or replaced without notice.
func (p *Parser) RegisterNetMessageHandler(handler interface{}) dp.HandlerIdentifier {
	return p.msgDispatcher.RegisterHandler(handler)
}

// UnregisterNetMessageHandler removes a net-message handler via identifier.
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
	// Check out demopacket.go to see which net-messages are already being parsed by default.
	// This is a beta feature and may be changed or replaced without notice.
	AdditionalNetMessageCreators map[int]NetMessageCreator
	// AdditionalEventEmitters contains additional event emitters - either from the fuzzy package or custom ones.
	// This is mainly used to add logic specifically for one type of demo (e.g. Matchmaking, FaceIt etc.).
	// This is a beta feature and may be changed or replaced without notice.
	// See also: package fuzzy for existing emitters with fuzzy-logic that depends on the demo-type.
	AdditionalEventEmitters []EventEmitter
}

// EventEmitter is the interface to define additional event-emitters.
// The emitters may fire additional events by calling the eventDispatcher function received during registration of the emitter.
// See also: package fuzzy for existing emitters with fuzzy-logic that depends on the demo-type.
type EventEmitter interface {
	Register(parser *Parser, eventDispatcher func(event interface{}))
}

// DefaultParserConfig is the default Parser configuration used by NewParser().
// You may set this variable to a custom configuration to be used by NewParser().
var DefaultParserConfig = ParserConfig{
	MsgQueueBufferSize: -1,
}

// NewParserWithConfig returns a new Parser with a custom configuration.
// See also: NewParser() & ParserConfig
func NewParserWithConfig(demostream io.Reader, config ParserConfig) *Parser {
	var p Parser

	// Init parser
	p.bitReader = bit.NewLargeBitReader(demostream)
	p.instanceBaselines = make(map[int][]byte)
	p.preprocessedBaselines = make(map[int]map[int]st.PropValue)
	p.equipmentMapping = make(map[*st.ServerClass]common.EquipmentElement)
	p.rawPlayers = make(map[int]*playerInfo)
	p.entityIDToPlayers = make(map[int]*common.Player)
	p.entities = make(map[int]*st.Entity)
	p.triggers = make(map[int]*boundingBoxInformation)
	p.cancelChan = make(chan struct{}, 1)
	p.gameState = newGameState()

	// Attach proto msg handlers
	p.msgDispatcher.RegisterHandler(p.handlePacketEntities)
	p.msgDispatcher.RegisterHandler(p.handleGameEventList)
	p.msgDispatcher.RegisterHandler(p.handleGameEvent)
	p.msgDispatcher.RegisterHandler(p.handleCreateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUpdateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUserMessage)
	p.msgDispatcher.RegisterHandler(p.handleFrameParsed)
	p.msgDispatcher.RegisterHandler(p.gameState.handleIngameTickNumber)

	if config.MsgQueueBufferSize >= 0 {
		p.initMsgQueue(config.MsgQueueBufferSize)
	}

	p.additionalNetMessageCreators = config.AdditionalNetMessageCreators

	for _, emitter := range config.AdditionalEventEmitters {
		emitter.Register(&p, p.eventDispatcher.Dispatch)
	}

	return &p
}

func (p *Parser) initMsgQueue(buf int) {
	p.msgQueue = make(chan interface{}, buf)
	p.msgDispatcher.AddQueues(p.msgQueue)
}
