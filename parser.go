package demoinfocs

import (
	"fmt"
	"io"
	"os"
	"sync"

	dp "github.com/markus-wa/godispatch"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	common "github.com/markus-wa/demoinfocs-golang/common"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// TODO?: create struct GameState for all game-state relevant stuff

// Parser can parse a CS:GO demo.
// Creating a Parser is done via NewParser().
// To start off use Parser.ParseHeader() to parse the demo header.
// After parsing the header Parser.ParseNextFrame() and Parser.ParseToEnd() can be used to parse the demo.
// Use Parser.RegisterEventHandler() to receive notifications about events.
type Parser struct {
	bitReader             *bit.BitReader
	stParser              st.Parser
	msgDispatcher         dp.Dispatcher
	eventDispatcher       dp.Dispatcher
	msgQueue              chan interface{}
	currentFrame          int
	ingameTick            int
	header                *common.DemoHeader // Pointer so we can check for nil
	equipmentMapping      map[*st.ServerClass]common.EquipmentElement
	rawPlayers            [maxPlayers]*common.PlayerInfo
	players               map[int]*common.Player
	connectedPlayers      map[int]*common.Player
	additionalPlayerInfo  [maxPlayers]common.AdditionalPlayerInformation
	entities              map[int]*st.Entity
	modelPreCache         []string                      // Used to find out whether a weapon is a p250 or cz for example (same id)
	weapons               [maxEntities]common.Equipment // Used to remember what a weapon is (p250 / cz etc.)
	tState                TeamState
	ctState               TeamState
	bombsiteA             bombsiteInfo
	bombsiteB             bombsiteInfo
	triggers              []*boundingBoxInformation
	instanceBaselines     map[int][]byte
	preprocessedBaselines map[int]map[int]st.PropValue
	gehDescriptors        map[int32]*msg.CSVCMsg_GameEventListDescriptorT
	stringTables          []*msg.CSVCMsg_CreateStringTable
	cancelChan            chan struct{}
	warn                  WarnHandler
	err                   error
	errLock               sync.Mutex
}

// Map returns the map name. E.g. de_dust2 or de_inferno.
// Deprecated, use Header().MapName instead.
func (p *Parser) Map() string {
	return p.header.MapName
}

// Header returns the DemoHeader which contains the demo's metadata.
func (p *Parser) Header() common.DemoHeader {
	return *p.header
}

// Participants returns all connected players.
// This includes spectators.
func (p *Parser) Participants() []*common.Player {
	r := make([]*common.Player, 0, len(p.connectedPlayers))
	for _, ptcp := range p.connectedPlayers {
		r = append(r, ptcp)
	}
	return r
}

// PlayingParticipants returns all players that aren't spectating.
func (p *Parser) PlayingParticipants() []*common.Player {
	r := make([]*common.Player, 0, len(p.connectedPlayers))
	for _, ptcp := range p.connectedPlayers {
		// FIXME: Why do we have to check for nil here???
		if ptcp != nil && ptcp.Team != common.TeamSpectators {
			r = append(r, ptcp)
		}
	}
	return r
}

// FrameRate returns the frame rate of the demo (frames / demo-ticks per second).
// Not necessarily the tick-rate the server ran on during the game.
// VolvoPlx128TixKTnxBye
func (p *Parser) FrameRate() float32 {
	return float32(p.header.PlaybackFrames) / p.header.PlaybackTime
}

// FrameTime returns the time a frame / demo-tick takes in seconds.
func (p *Parser) FrameTime() float32 {
	return p.header.PlaybackTime / float32(p.header.PlaybackFrames)
}

// Progress returns the parsing progress from 0 to 1.
// Where 0 means nothing has been parsed yet and 1 means the demo has been parsed to the end.
// Might not actually be reliable since it's just based on the reported tick count of the header.
func (p *Parser) Progress() float32 {
	return float32(p.currentFrame) / float32(p.header.PlaybackFrames)
}

// CurrentFrame return the number of the current frame, aka. 'demo-tick' (Since demos often have a different tick-rate than the game).
// Starts with frame 0, should go up to DemoHeader.PlaybackFrames but might not be the case (usually it's just close to it).
func (p *Parser) CurrentFrame() int {
	return p.currentFrame
}

// IngameTick returns the latest actual tick number of the server during the game.
// Watch out, I've seen this return wonky negative numbers at the start of demos.
func (p *Parser) IngameTick() int {
	return p.ingameTick
}

// CurrentTime returns the ingame time in seconds since the start of the demo.
func (p *Parser) CurrentTime() float32 {
	return float32(p.currentFrame) * p.FrameTime()
}

// RegisterEventHandler registers a handler for game events.
// Must be of type func(<EventType>) where EventType is the kind of event that is handled.
// To catch all events func(interface{}) can be used.
// Parameter handler has to be of type interface{} because lolnogenerics.
func (p *Parser) RegisterEventHandler(handler interface{}) {
	p.eventDispatcher.RegisterHandler(handler)
}

// CTState returns the TeamState of the CT team.
// Make sure you handle swapping sides properly if you keep the reference.
func (p *Parser) CTState() *TeamState {
	return &p.ctState
}

// TState returns the TeamState of the T team.
// Make sure you handle swapping sides properly if you keep the reference.
func (p *Parser) TState() *TeamState {
	return &p.tState
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

// TeamState contains a team's ID, score, clan name & country flag.
type TeamState struct {
	id       int
	score    int
	clanName string
	flag     string
}

// ID returns the team-ID.
// This stays the same even after switching sides.
func (ts TeamState) ID() int {
	return ts.id
}

// Score returns the team's number of rounds won.
func (ts TeamState) Score() int {
	return ts.score
}

// ClanName returns the team's clan name.
func (ts TeamState) ClanName() string {
	return ts.clanName
}

// Flag returns the team's country flag.
func (ts TeamState) Flag() string {
	return ts.flag
}

// TODO: Maybe we should use a channel instead of that WarnHandler stuff

// WarnHandler is a function that handles warnings of a Parser.
type WarnHandler func(string)

// WarnToStdErr is a WarnHandler that prints all warnings to standard error output.
func WarnToStdErr(warning string) {
	fmt.Fprintln(os.Stderr, warning)
}

// TODO: Change the New* methods (names + parameters)

// NewParser creates a new Parser on the basis of an io.Reader
// - like os.File or bytes.Reader - that reads demo data.
// Any warnings that don't stop the Parser from doing it's job
// will be passed to the warnHandler if it's not nil.
func NewParser(demostream io.Reader, warnHandler WarnHandler) *Parser {
	return NewParserWithBufferSize(demostream, -1, warnHandler)
}

// NewParserWithBufferSize returns a new Parser with a custom msgQueue buffer size.
// For large demos, fast i/o and slow CPUs higher numbers are suggested and vice versa.
// The buffer size can easily be in the hundred-thousands to low millions for the best performance.
// A negative value will make the Parser automatically decide the buffer size during ParseHeader()
// based on the number of ticks in the demo (nubmer of ticks = buffer size).
// See also: NewParser()
func NewParserWithBufferSize(demostream io.Reader, msgQueueBufferSize int, warnHandler WarnHandler) *Parser {
	var p Parser
	// Init parser
	p.bitReader = bit.NewLargeBitReader(demostream)
	p.instanceBaselines = make(map[int][]byte)
	p.preprocessedBaselines = make(map[int]map[int]st.PropValue)
	p.equipmentMapping = make(map[*st.ServerClass]common.EquipmentElement)
	p.players = make(map[int]*common.Player)
	p.connectedPlayers = make(map[int]*common.Player)
	p.entities = make(map[int]*st.Entity)
	p.cancelChan = make(chan struct{}, 1)
	p.warn = warnHandler

	// Attach proto msg handlers
	p.msgDispatcher.RegisterHandler(p.handlePacketEntities)
	p.msgDispatcher.RegisterHandler(p.handleGameEventList)
	p.msgDispatcher.RegisterHandler(p.handleGameEvent)
	p.msgDispatcher.RegisterHandler(p.handleCreateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUpdateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUserMessage)
	p.msgDispatcher.RegisterHandler(p.handleFrameParsed)
	p.msgDispatcher.RegisterHandler(p.handleIngameTickNumber)

	if msgQueueBufferSize >= 0 {
		p.initMsgQueue(msgQueueBufferSize)
	}
	return &p
}

func (p *Parser) initMsgQueue(buf int) {
	p.msgQueue = make(chan interface{}, buf)
	p.msgDispatcher.AddQueues(p.msgQueue)
}
