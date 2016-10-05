package main

import (
	"fmt"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	dt "github.com/markus-wa/demoinfocs-golang/dt"
	st "github.com/markus-wa/demoinfocs-golang/st"
	"io"
	"os"
)

const (
	MaxEditctBits = 11
	indexMask     = ((1 << MaxEditctBits) - 1)
	maxEntities   = (1 << MaxEditctBits)
	MaxPlayers    = 64
	MaxWeapons    = 64
)

func main() {
	d, _ := os.Open("C:\\Dev\\demo.dem")
	p := NewParser(d)
	p.ParseHeader()
	p.parseTick()
}

// FIXME: create struct GameState for all game-state relevant stuff
type Parser struct {
	bitstream            bs.BitReader
	stParser             *st.Parser
	dtParser             *dt.Parser
	currentTick          uint
	ingameTick           uint
	header               *DemoHeader
	rawPlayers           [MaxPlayers]*PlayerInfo
	players              map[int]*Player
	additionalPlayerInfo [MaxPlayers]*AdditionalPlayerInformation
	entities             [maxEntities]*dt.Entity
	modelPreCache        []string               // Used to find out whether a weapon is a p250 or cz for example (same id)
	weapons              [MaxWeapons]*Equipment // Used to remember what a weapon is (p250 / cz etc.)
	tState               *TeamState
	ctState              *TeamState
}

func (p *Parser) Map() string {
	return p.header.MapName()
}

func (p *Parser) Participants() []*Player {
	r := make([]*Player, 0, len(p.players))
	for _, ptcp := range p.players {
		r = append(r, ptcp)
	}
	return r
}

func (p *Parser) PlayingParticipants() []*Player {
	r := make([]*Player, 0, len(p.players))
	for _, ptcp := range p.players {
		if ptcp.Team != Team_Spectators {
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

func (p *Parser) CurrentTick() uint {
	return p.currentTick
}

func (p *Parser) IngameTick() uint {
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
	fmt.Println(h)
	p.header = &h
	return nil
}

func (p *Parser) ParseToEnd() {
	for p.parseNextTick() {
	}
}

func (p *Parser) parseNextTick() bool {
	return false
}

func (p *Parser) parseTick() bool {
	cmd := DemoCommand(p.bitstream.ReadByte())
	switch cmd {
	case DC_Synctick:
		break
	case DC_Stop:
		return false
	case DC_ConsoleCommand:
		return false
	}
	return false
}

func NewParser(demostream io.Reader) *Parser {
	return &Parser{bitstream: bs.NewBitReader(demostream)}
}
