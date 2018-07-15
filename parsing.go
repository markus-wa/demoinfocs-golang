package demoinfocs

import (
	"errors"
	"fmt"
	"io"

	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

const maxOsPath = 260

const (
	playerWeaponPrefix    = "m_hMyWeapons."
	playerWeaponPrePrefix = "bcc_nonlocaldata."
)

// Parsing errors
var (
	// ErrCancelled signals that parsing was cancelled via Parser.Cancel()
	ErrCancelled = errors.New("Parsing was cancelled before it finished (ErrCancelled)")

	// ErrUnexpectedEndOfDemo signals that the demo is incomplete / corrupt -
	// these demos may still be useful, check how far the parser got.
	ErrUnexpectedEndOfDemo = errors.New("Demo stream ended unexpectedly (ErrUnexpectedEndOfDemo)")

	// ErrInvalidFileType signals that the input isn't a valid CS:GO demo.
	ErrInvalidFileType = errors.New("Invalid File-Type; expecting HL2DEMO in the first 8 bytes")

	// ErrHeaderNotParsed signals that the header hasn't been parsed before attempting to parse a tick.
	ErrHeaderNotParsed = errors.New("Header must be parsed before trying to parse a tick. See Parser.ParseHeader()")
)

// ParseHeader attempts to parse the header of the demo.
//
// Returns ErrInvalidFileType if the filestamp (first 8 bytes) doesn't match HL2DEMO.
func (p *Parser) ParseHeader() (common.DemoHeader, error) {
	var h common.DemoHeader
	h.Filestamp = p.bitReader.ReadCString(8)
	h.Protocol = p.bitReader.ReadSignedInt(32)
	h.NetworkProtocol = p.bitReader.ReadSignedInt(32)
	h.ServerName = p.bitReader.ReadCString(maxOsPath)
	h.ClientName = p.bitReader.ReadCString(maxOsPath)
	h.MapName = p.bitReader.ReadCString(maxOsPath)
	h.GameDirectory = p.bitReader.ReadCString(maxOsPath)
	h.PlaybackTime = p.bitReader.ReadFloat()
	h.PlaybackTicks = p.bitReader.ReadSignedInt(32)
	h.PlaybackFrames = p.bitReader.ReadSignedInt(32)
	h.SignonLength = p.bitReader.ReadSignedInt(32)

	if h.Filestamp != "HL2DEMO" {
		return h, ErrInvalidFileType
	}

	// Initialize queue if the buffer size wasn't specified, the amount of ticks
	// seems to be a good indicator of how many events we'll get
	if p.msgQueue == nil {
		p.initMsgQueue(h.PlaybackTicks)
	}

	p.header = &h

	return h, nil
}

// ParseToEnd attempts to parse the demo until the end.
// Aborts and returns ErrCancelled if Cancel() is called before the end.
//
// See also: ParseNextFrame() for other possible errors.
func (p *Parser) ParseToEnd() (err error) {
	defer func() {
		if err == nil {
			err = recoverFromUnexpectedEOF(recover())
		}
	}()

	if p.header == nil {
		return ErrHeaderNotParsed
	}

	for {
		select {
		case <-p.cancelChan:
			return ErrCancelled

		default:
			if !p.parseFrame() {
				// Make sure all the messages of the demo are handled
				p.msgDispatcher.SyncQueues(p.msgQueue)

				// Close msgQueue
				close(p.msgQueue)

				return p.error()
			}
		}

		if pErr := p.error(); pErr != nil {
			return pErr
		}
	}
}

func recoverFromUnexpectedEOF(r interface{}) error {
	if r != nil {
		if r == io.ErrUnexpectedEOF {
			return ErrUnexpectedEndOfDemo
		}
		panic(r)
	}
	return nil
}

// Cancel aborts ParseToEnd().
// All information that was already read up to this point may still be used (and new events may still be sent out).
func (p *Parser) Cancel() {
	p.cancelChan <- struct{}{}
}

/*
ParseNextFrame attempts to parse the next frame / demo-tick (not ingame tick).

Returns true unless the demo command 'stop' or an error was encountered.
Returns an error if the header hasn't been parsed yet - see Parser.ParseHeader().

May return ErrUnexpectedEndOfDemo for incomplete / corrupt demos.
May panic if the demo is corrupt in some way.

See also: ParseToEnd() for parsing the complete demo in one go (faster).
*/
func (p *Parser) ParseNextFrame() (b bool, err error) {
	defer func() {
		if err == nil {
			err = recoverFromUnexpectedEOF(recover())
		}
	}()

	if p.header == nil {
		return false, ErrHeaderNotParsed
	}

	b = p.parseFrame()

	// Make sure all the messages of the frame are handled
	p.msgDispatcher.SyncQueues(p.msgQueue)

	// Close msgQueue if we are done
	if !b {
		close(p.msgQueue)
	}

	err = p.error()
	return
}

// Demo commands as documented at https://developer.valvesoftware.com/wiki/DEM_Format
type demoCommand byte

const (
	dcSignon         demoCommand = 1
	dcPacket         demoCommand = 2
	dcSynctick       demoCommand = 3
	dcConsoleCommand demoCommand = 4
	dcUserCommand    demoCommand = 5
	dcDataTables     demoCommand = 6
	dcStop           demoCommand = 7
	dcCustomData     demoCommand = 8
	dcStringTables   demoCommand = 9
)

func (p *Parser) parseFrame() bool {
	cmd := demoCommand(p.bitReader.ReadSingleByte())

	// Send ingame tick number update
	p.msgQueue <- ingameTickNumber(p.bitReader.ReadSignedInt(32))

	// Skip 'player slot'
	p.bitReader.Skip(8)

	debugDemoCommand(cmd)

	switch cmd {
	case dcSynctick:
		// Ignore

	case dcStop:
		return false

	case dcConsoleCommand:
		// Skip
		p.bitReader.Skip(p.bitReader.ReadSignedInt(32) << 3)

	case dcDataTables:
		p.msgDispatcher.SyncQueues(p.msgQueue)

		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.stParser.ParsePacket(p.bitReader)
		p.bitReader.EndChunk()

		debugAllServerClasses(p.ServerClasses())

		p.mapEquipment()
		p.bindEntities()

		p.eventDispatcher.Dispatch(events.DataTablesParsedEvent{})

	case dcStringTables:
		p.msgDispatcher.SyncQueues(p.msgQueue)

		p.parseStringTables()

	case dcUserCommand:
		// Skip
		p.bitReader.Skip(32)
		p.bitReader.Skip(p.bitReader.ReadSignedInt(32) << 3)

	case dcSignon:
		fallthrough
	case dcPacket:
		p.parsePacket()

	case dcCustomData:
		// Might as well panic since we'll be way off if we dont skip the whole thing
		panic("Found CustomData but not handled")

	default:
		panic(fmt.Sprintf("Canny handle it anymoe (command %v unknown)", cmd))
	}

	// Queue up some post processing
	p.msgQueue <- frameParsedToken

	return true
}

type frameParsedTokenType struct{}

var frameParsedToken = new(frameParsedTokenType)

func (p *Parser) handleFrameParsed(*frameParsedTokenType) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	for k, rp := range p.rawPlayers {
		// We need to re-map the players from their entityID to their UID.
		// This is necessary because we don't always have the UID when the player connects (or something like that, not really sure tbh).
		if pl := p.entityIDToPlayers[k]; pl != nil {
			pl.Name = rp.name
			pl.SteamID = rp.xuid
			pl.IsBot = rp.isFakePlayer
			pl.AdditionalPlayerInformation = &p.additionalPlayerInfo[pl.EntityID]

			if pl.IsAlive() {
				pl.LastAlivePosition = pl.Position
			}

			if p.gameState.players[rp.userID] == nil {
				p.gameState.players[rp.userID] = pl

				if pl.SteamID != 0 {
					p.eventDispatcher.Dispatch(events.PlayerBindEvent{Player: pl})
				}
			}
		}
	}

	p.currentFrame++
	p.eventDispatcher.Dispatch(events.TickDoneEvent{})
}
