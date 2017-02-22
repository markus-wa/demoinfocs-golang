package demoinfocs

import (
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/msg"
	"github.com/markus-wa/demoinfocs-golang/st"
	dp "github.com/markus-wa/godispatch"
	"io"
)

// FIXME?: create struct GameState for all game-state relevant stuff
type Parser struct {
	bitreader             bs.BitReader
	stParser              st.Parser
	msgDispatcher         dp.Dispatcher
	eventDispatcher       dp.Dispatcher
	msgQueue              chan interface{}
	currentTick           int
	ingameTick            int
	header                *common.DemoHeader // Pointer so we can check for nil
	equipmentMapping      map[*st.ServerClass]common.EquipmentElement
	rawPlayers            [maxPlayers]*common.PlayerInfo
	players               map[int]*common.Player
	connectedPlayers      map[int]*common.Player
	additionalPlayerInfo  [maxPlayers]common.AdditionalPlayerInformation
	entities              [maxEntities]*st.Entity
	modelPreCache         []string                      // Used to find out whether a weapon is a p250 or cz for example (same id)
	weapons               [maxEntities]common.Equipment // Used to remember what a weapon is (p250 / cz etc.)
	tState                TeamState
	ctState               TeamState
	bombsiteA             bombsiteInfo
	bombsiteB             bombsiteInfo
	triggers              []*BoundingBoxInformation
	instanceBaselines     map[int][]byte
	preprocessedBaselines map[int][]*st.RecordedPropertyUpdate
	gehDescriptors        map[int32]*msg.CSVCMsg_GameEventListDescriptorT
	stringTables          []*msg.CSVCMsg_CreateStringTable
}

func (p *Parser) Map() string {
	return p.header.MapName
}

func (p *Parser) Participants() []*common.Player {
	r := make([]*common.Player, 0, len(p.connectedPlayers))
	for _, ptcp := range p.connectedPlayers {
		r = append(r, ptcp)
	}
	return r
}

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

func (p *Parser) TickRate() float32 {
	return float32(p.header.PlaybackFrames) / p.header.PlaybackTime
}

func (p *Parser) TickTime() float32 {
	return p.header.PlaybackTime / float32(p.header.PlaybackFrames)
}

func (p *Parser) Progress() float32 {
	return float32(p.currentTick) / float32(p.header.PlaybackFrames)
}

func (p *Parser) CurrentTick() int {
	return p.currentTick
}

func (p *Parser) IngameTick() int {
	return p.ingameTick
}

func (p *Parser) CurrentTime() float32 {
	return float32(p.currentTick) * p.TickTime()
}

// RegisterEventHandler registers a handler for game events.
// Must be of type func(<EventType>) where EventType is the kind of event that is handled.
// To catch all events func(interface{}) can be used.
// Parameter handler has to be of type interface{} because go doesn't support generics.
func (p *Parser) RegisterEventHandler(handler interface{}) {
	p.eventDispatcher.RegisterHandler(handler)
}

func (p *Parser) CTState() *TeamState {
	return &p.ctState
}

func (p *Parser) TState() *TeamState {
	return &p.tState
}

// NewParser creates a new Parser on the basis of an io.Reader
// - like os.File or bytes.Reader - that reads demo data.
func NewParser(demostream io.Reader) *Parser {
	var p Parser
	// Init parser
	p.bitreader = bs.NewBitReader(demostream, bs.LargeBuffer)
	p.msgQueue = make(chan interface{}, 8)
	p.instanceBaselines = make(map[int][]byte)
	p.preprocessedBaselines = make(map[int][]*st.RecordedPropertyUpdate)
	p.equipmentMapping = make(map[*st.ServerClass]common.EquipmentElement)
	p.players = make(map[int]*common.Player)
	p.connectedPlayers = make(map[int]*common.Player)

	// Attach proto msg handlers
	p.msgDispatcher.RegisterHandler(p.handlePackageEntities)
	p.msgDispatcher.RegisterHandler(p.handleGameEventList)
	p.msgDispatcher.RegisterHandler(p.handleGameEvent)
	p.msgDispatcher.RegisterHandler(p.handleCreateStringTable)
	p.msgDispatcher.RegisterHandler(p.handleUpdateStringTable)

	p.msgDispatcher.AddQueues(p.msgQueue)
	return &p
}
