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

const (
	msgHeaderNotParsed = "Tried to parse tick before parsing header"
)

// ParseHeader attempts to parse the header of the demo.
// Returns error if the filestamp (first 8 bytes) doesn't match HL2DEMO.
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
		return h, errors.New("Invalid File-Type; expecting HL2DEMO in the first 8 bytes")
	}

	// Initialize queue if the buffer size wasn't specified, the amount of ticks
	// seems to be a good indicator of how many events we'll get
	if p.msgQueue == nil {
		p.initMsgQueue(h.PlaybackTicks)
	}

	p.header = &h
	// TODO: Deprecated, remove this + HeaderParsedEvent in 1.0.0
	p.eventDispatcher.Dispatch(events.HeaderParsedEvent{Header: h})
	return h, nil
}

// Parsing errors
var (
	// ErrCancelled signals that parsing was cancelled via Parser.Cancel()
	ErrCancelled error = errors.New("Parsing was cancelled before it finished (ErrCancelled)")

	// ErrUnexpectedEndOfDemo signals that the demo is incomplete / corrupt -
	// these demos may still be useful, check the how far the parser got.
	ErrUnexpectedEndOfDemo error = errors.New("Demo stream ended unexpectedly (ErrUnexpectedEndOfDemo)")
)

// ParseToEnd attempts to parse the demo until the end.
// Aborts and returns ErrCancelled if Cancel() is called before the end.
// May return ErrUnexpectedEndOfDemo for incomplete / corrupt demos.
// May panic if the demo is corrupt in some way.
func (p *Parser) ParseToEnd() (err error) {
	defer func() {
		if err == nil {
			err = recoverFromUnexpectedEOF(recover())
		}
	}()

	if p.header == nil {
		panic(msgHeaderNotParsed)
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

// Cancel aborts ParseToEnd(). All information that was already read
// up to this point will still be used (and new events may still be sent).
func (p *Parser) Cancel() {
	p.cancelChan <- struct{}{}
}

// ParseNextFrame attempts to parse the next frame / demo-tick (not ingame tick).
// Returns true unless the demo command 'stop' or an error was encountered.
// May return ErrUnexpectedEndOfDemo for incomplete / corrupt demos.
// Panics if header hasn't been parsed yet - see Parser.ParseHeader().
func (p *Parser) ParseNextFrame() (b bool, err error) {
	defer func() {
		if err == nil {
			err = recoverFromUnexpectedEOF(recover())
		}
	}()

	if p.header == nil {
		panic(msgHeaderNotParsed)
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

func (p *Parser) parseFrame() bool {
	cmd := demoCommand(p.bitReader.ReadSingleByte())

	// Send ingame tick number update
	p.msgQueue <- ingameTickNumber(p.bitReader.ReadSignedInt(32))

	// Skip 'player slot'
	p.bitReader.ReadSingleByte()

	switch cmd {
	case dcSynctick:
		// Ignore

	case dcStop:
		return false

	case dcConsoleCommand:
		// Skip
		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.bitReader.EndChunk()

	case dcDataTables:
		p.msgDispatcher.SyncQueues(p.msgQueue)

		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.stParser.ParsePacket(p.bitReader)
		p.bitReader.EndChunk()

		p.mapEquipment()
		p.bindEntities()

	case dcStringTables:
		p.msgDispatcher.SyncQueues(p.msgQueue)

		p.parseStringTables()

	case dcUserCommand:
		// Skip
		p.bitReader.ReadInt(32)
		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.bitReader.EndChunk()

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
