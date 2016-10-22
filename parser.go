package demoinfocs

import (
	"github.com/golang/geo/r3"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/msg"
	"github.com/markus-wa/demoinfocs-golang/st"
	"io"
	"reflect"
)

// FIXME: create struct GameState for all game-state relevant stuff
type Parser struct {
	bitreader             bs.BitReader
	stParser              *st.Parser
	eventDispatcher       *eventDispatcher
	eventQueue            chan interface{}
	currentTick           int
	ingameTick            int
	header                *DemoHeader
	equipmentMapping      map[*st.ServerClass]common.EquipmentElement
	rawPlayers            [MaxPlayers]*common.PlayerInfo
	players               map[int]*common.Player
	additionalPlayerInfo  [MaxPlayers]common.AdditionalPlayerInformation
	entities              [maxEntities]*st.Entity
	modelPreCache         []string                       // Used to find out whether a weapon is a p250 or cz for example (same id)
	weapons               [maxEntities]*common.Equipment // Used to remember what a weapon is (p250 / cz etc.)
	tState                *TeamState
	ctState               *TeamState
	bombsiteACenter       r3.Vector
	bombsiteBCenter       r3.Vector
	triggers              []*BoundingBoxInformation
	instanceBaselines     map[int][]byte
	preprocessedBaselines map[int][]*st.RecordedPropertyUpdate
}

func (p *Parser) Map() string {
	return p.header.MapName()
}

func (p *Parser) Participants() []*common.Player {
	r := make([]*common.Player, 0, len(p.players))
	for _, ptcp := range p.players {
		r = append(r, ptcp)
	}
	return r
}

func (p *Parser) PlayingParticipants() []*common.Player {
	r := make([]*common.Player, 0, len(p.players))
	for _, ptcp := range p.players {
		if ptcp.Team != common.Team_Spectators {
			r = append(r, ptcp)
		}
	}
	return r
}

func (p *Parser) TickRate() float32 {
	return float32(p.header.playbackFrames) / p.header.playbackTime
}

func (p *Parser) TickTime() float32 {
	return p.header.playbackTime / float32(p.header.playbackFrames)
}

func (p *Parser) Progress() float32 {
	return float32(p.currentTick) / float32(p.header.playbackFrames)
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

func (p *Parser) EventDispatcher() EventDispatcher {
	return p.eventDispatcher
}

func (p *Parser) CTState() *TeamState {
	return p.ctState
}

func (p *Parser) TState() *TeamState {
	return p.tState
}

func NewParser(demostream io.Reader) *Parser {
	p := Parser{}
	// Init parser
	p.bitreader = bs.NewBitReader(demostream)
	p.stParser = &st.Parser{}
	p.eventDispatcher = &eventDispatcher{}
	p.eventQueue = make(chan interface{})
	p.ctState = &TeamState{}
	p.tState = &TeamState{}
	p.instanceBaselines = make(map[int][]byte)
	p.preprocessedBaselines = make(map[int][]*st.RecordedPropertyUpdate)
	p.equipmentMapping = make(map[*st.ServerClass]common.EquipmentElement)
	p.players = make(map[int]*common.Player)

	// Attach handlers
	p.eventDispatcher.Register(reflect.TypeOf(&msg.CSVCMsg_PacketEntities{}), p.handlePackageEntities)

	go p.eventDispatcher.dispatchQueue(p.eventQueue)
	return &p
}
