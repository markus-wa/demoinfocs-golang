package main

import (
	"fmt"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/dp"
	"github.com/markus-wa/demoinfocs-golang/dt"
	"github.com/markus-wa/demoinfocs-golang/st"
	"io"
	"os"
	"time"
)

const (
	MaxEditctBits = 11
	indexMask     = ((1 << MaxEditctBits) - 1)
	maxEntities   = (1 << MaxEditctBits)
	MaxPlayers    = 64
	MaxWeapons    = 64
)

const MaxOsPath = 260

func main() {
	d, _ := os.Open("C:\\Dev\\demo.dem")
	p := NewParser(d)
	p.ParseHeader()
	ts := time.Now().Unix()
	p.ParseToEnd()
	fmt.Println("took", time.Now().Unix()-ts)
}

// FIXME: create struct GameState for all game-state relevant stuff
type Parser struct {
	bitstream            bs.BitReader
	dtParser             *dt.Parser
	currentTick          int
	ingameTick           int
	header               *DemoHeader
	rawPlayers           [MaxPlayers]*common.PlayerInfo
	players              map[int]*common.Player
	additionalPlayerInfo [MaxPlayers]*common.AdditionalPlayerInformation
	entities             [maxEntities]*dt.Entity
	modelPreCache        []string                      // Used to find out whether a weapon is a p250 or cz for example (same id)
	weapons              [MaxWeapons]*common.Equipment // Used to remember what a weapon is (p250 / cz etc.)
	tState               *common.TeamState
	ctState              *common.TeamState
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

func (p *Parser) ParseHeader() error {
	fmt.Println("tryna parse dat header")
	h := DemoHeader{}
	h.filestamp = p.bitstream.ReadCString(8)
	h.protocol = p.bitstream.ReadSignedInt(32)
	h.networkProtocol = p.bitstream.ReadSignedInt(32)
	h.serverName = p.bitstream.ReadCString(MaxOsPath)
	h.clientName = p.bitstream.ReadCString(MaxOsPath)
	h.mapName = p.bitstream.ReadCString(MaxOsPath)
	h.gameDirectory = p.bitstream.ReadCString(MaxOsPath)
	h.playbackTime = p.bitstream.ReadFloat()
	h.playbackTicks = p.bitstream.ReadSignedInt(32)
	h.playbackFrames = p.bitstream.ReadSignedInt(32)
	h.signonLength = p.bitstream.ReadSignedInt(32)
	if h.filestamp != "HL2DEMO" {
		panic("Shit's fucked mate (Invalid File-Type; expecting HL2DEMO)")
	}
	fmt.Println("Header: ", h)
	p.header = &h
	return nil
}

func (p *Parser) ParseToEnd() {
	for p.parseNextTick() {
	}
}

func (p *Parser) parseNextTick() bool {
	if p.header == nil {
		panic("Tried to parse tick before parsing header")
	}
	b := p.parseTick()
	// FIXME: we should do some more stuff here
	return b
}

func (p *Parser) parseTick() bool {
	cmd := DemoCommand(p.bitstream.ReadByte())

	// Tick number
	p.ingameTick = p.bitstream.ReadSignedInt(32)
	// Skip 'player slot'
	p.bitstream.ReadByte()

	p.currentTick++

	switch cmd {
	case DC_Synctick:
		// Ignore
	case DC_Stop:
		fmt.Println("We done it boys")
		return false

	case DC_ConsoleCommand:
		// Skip
		p.bitstream.BeginChunk(p.bitstream.ReadSignedInt(32) * 8)
		p.bitstream.EndChunk()

	case DC_DataTables:
		p.bitstream.BeginChunk(p.bitstream.ReadSignedInt(32) * 8)
		p.bitstream.EndChunk()

	case DC_StringTables:
		p.bitstream.BeginChunk(p.bitstream.ReadSignedInt(32) * 8)
		st.ParsePacket(p.bitstream)
		p.bitstream.EndChunk()

	case DC_UserCommand:
		// Skip
		p.bitstream.ReadInt(32)
		p.bitstream.BeginChunk(p.bitstream.ReadSignedInt(32) * 8)
		p.bitstream.EndChunk()

	case DC_Signon:
		fallthrough
	case DC_Packet:
		p.parseDemoPacket()
	default:
		panic("Canny handle it anymoe (command unknown)")
	}
	return true
}

func (p *Parser) parseDemoPacket() {
	// Booooring
	parseCommandInfo(p.bitstream)
	p.bitstream.ReadInt(32) // SeqNrIn
	p.bitstream.ReadInt(32) // SeqNrOut

	p.bitstream.BeginChunk(p.bitstream.ReadSignedInt(32) * 8)
	dp.ParsePacket(p.bitstream)
	p.bitstream.EndChunk()
}

func NewParser(demostream io.Reader) *Parser {
	return &Parser{bitstream: bs.NewBitReader(demostream)}
}
