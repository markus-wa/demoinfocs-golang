package demoinfocs

import (
	"io"

	bs "github.com/markus-wa/demoinfocs-golang/bitread"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/msg"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
	dp "github.com/markus-wa/godispatch"
)

// TODO?: create struct GameState for all game-state relevant stuff

// Parser can parse a CS:GO demo.
// Creating a Parser is done via NewParser().
// To start off use Parser.ParseHeader() to parse the demo header.
// After parsing the header Parser.ParseNextFrame() and Parser.ParseToEnd() can be used to parse the demo.
// Use Parser.RegisterEventHandler() to receive notifications about events.
type Parser struct {
	bitReader             *bs.BitReader
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
}

// Map returns the map name. E.g. de_dust2 or de_inferno.
func (p *Parser) Map() string {
	return p.header.MapName
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
		if ptcp != nil && ptcp.Team != common.Team_Spectators {
			r = append(r, ptcp)
		}
	}
	return r
}

// FrameRate returns the frame rate of the demo (frames / demo-ticks per second).
// Not necessarily the tick-rate the server ran on during the game.
// VolvoPlx128BitKTnxBye
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

// NewParser creates a new Parser on the basis of an io.Reader
// - like os.File or bytes.Reader - that reads demo data.
func NewParser(demostream io.Reader) *Parser {
	var p Parser
	// Init parser
	p.bitReader = bs.NewLargeBitReader(demostream)
	p.msgQueue = make(chan interface{}, 8)
	p.instanceBaselines = make(map[int][]byte)
	p.preprocessedBaselines = make(map[int]map[int]st.PropValue)
	p.equipmentMapping = make(map[*st.ServerClass]common.EquipmentElement)
	p.players = make(map[int]*common.Player)
	p.connectedPlayers = make(map[int]*common.Player)
	p.entities = make(map[int]*st.Entity)
	p.cancelChan = make(chan struct{}, 1)

	// Attach proto msg handlers
	p.msgDispatcher.RegisterHandler(p.handlePacketEntities)
	p.msgDispatcher.RegisterHandler(p.handleGameEventList)
	p.msgDispatcher.RegisterHandler(p.handleGameEvent)
	p.msgDispatcher.RegisterHandler(p.handleCreateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUpdateStringTable)

	p.msgDispatcher.AddQueues(p.msgQueue)
	return &p
}
