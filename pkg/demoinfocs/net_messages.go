package demoinfocs

import (
	"bytes"
	"encoding/binary"
	"fmt"

	bit "github.com/markus-wa/demoinfocs-golang/v5/internal/bitread"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"github.com/markus-wa/go-unassert"
	"github.com/markus-wa/ice-cipher-go/pkg/ice"
	"google.golang.org/protobuf/proto"
)

func (p *parser) onEntity(e sendtables.Entity, op sendtables.EntityOp) error {
	if op&sendtables.EntityOpCreated > 0 {
		p.gameState.entities[e.ID()] = e
	} else if op&sendtables.EntityOpDeleted > 0 {
		delete(p.gameState.entities, e.ID())
	}

	return nil
}

func (p *parser) handleSetConVar(setConVar *msg.CNETMsg_SetConVar) {
	updated := make(map[string]string)
	for _, cvar := range setConVar.Convars.Cvars {
		updated[cvar.GetName()] = cvar.GetValue()
		p.gameState.rules.conVars[cvar.GetName()] = cvar.GetValue()
	}

	p.eventDispatcher.Dispatch(events.ConVarsUpdated{
		UpdatedConVars: updated,
	})
}

func (p *parser) handleServerInfo(srvInfo *msg.CSVCMsg_ServerInfo) {
	// srvInfo.MapCrc might be interesting as well
	p.tickInterval = srvInfo.GetTickInterval()

	p.eventDispatcher.Dispatch(events.TickRateInfoAvailable{
		TickRate: p.TickRate(),
		TickTime: p.TickTime(),
	})
}

func (p *parser) handleMessageSayText(msg *msg.CUserMessageSayText) {
	p.eventDispatcher.Dispatch(events.SayText{
		EntIdx:    int(msg.GetPlayerindex()),
		IsChat:    msg.GetChat(),
		IsChatAll: false,
		Text:      msg.GetText(),
	})
}

func (p *parser) handleMessageSayText2(msg *msg.CUserMessageSayText2) {
	p.eventDispatcher.Dispatch(events.SayText2{
		EntIdx:    int(msg.GetEntityindex()),
		IsChat:    msg.GetChat(),
		IsChatAll: false,
		MsgName:   msg.GetMessagename(),
		Params:    []string{msg.GetParam1(), msg.GetParam2(), msg.GetParam3(), msg.GetParam4()},
	})

	switch msg.GetMessagename() {
	case "Cstrike_Chat_All", "Cstrike_Chat_AllSpec":
		fallthrough
	case "Cstrike_Chat_AllDead":
		sender := p.gameState.playersByEntityID[int(msg.GetEntityindex())]

		p.eventDispatcher.Dispatch(events.ChatMessage{
			Sender:    sender,
			Text:      msg.GetParam2(),
			IsChatAll: false,
		})

	case "#CSGO_Coach_Join_T": // Ignore these
	case "#CSGO_Coach_Join_CT":
	case "#CSGO_No_Longer_Coach":
	case "#Cstrike_Name_Change":
	case "Cstrike_Chat_T_Loc":
	case "Cstrike_Chat_CT_Loc":
	case "Cstrike_Chat_T_Dead":
	case "Cstrike_Chat_CT_Dead":

	default:
		errMsg := fmt.Sprintf("skipped sending ChatMessageEvent for SayText2 with unknown MsgName %q", msg.GetMessagename())

		p.eventDispatcher.Dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}
}

func (p *parser) handleServerRankUpdate(msg *msg.CCSUsrMsg_ServerRankUpdate) {
	for _, v := range msg.RankUpdate {
		steamID32 := uint32(v.GetAccountId())
		player, ok := p.gameState.playersBySteamID32[steamID32]
		if !ok {
			errMsg := fmt.Sprintf("rank update for unknown player with SteamID32=%d", steamID32)

			p.eventDispatcher.Dispatch(events.ParserWarn{Message: errMsg})
			unassert.Error(errMsg)
		}

		p.eventDispatcher.Dispatch(events.RankUpdate{
			SteamID32:  v.GetAccountId(),
			RankOld:    int(v.GetRankOld()),
			RankNew:    int(v.GetRankNew()),
			WinCount:   int(v.GetNumWins()),
			RankChange: v.GetRankChange(),
			Player:     player,
		})
	}
}

// encryptedNetMessageKeyType is the only key type the parser can decrypt,
// it corresponds to the public key found in `match730_*.dem.info` files.
const encryptedNetMessageKeyType = 2

// decryptNetMessage decrypts an encrypted net-message payload and returns a
// bit reader positioned at the start of the inner message, plus the number of
// payload bytes. ok is false if the message could not be decrypted, in which
// case a ParserWarn has been dispatched and the reader is nil.
// The caller is responsible for pooling the returned reader.
func (p *parser) decryptNetMessage(keyType int32, encrypted []byte) (br *bit.BitReader, nBytesWritten int, ok bool) {
	if keyType != encryptedNetMessageKeyType {
		return nil, 0, false
	}

	if p.decryptionKey == nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Type:    events.WarnTypeMissingNetMessageDecryptionKey,
			Message: "received encrypted net-message but no decryption key is set",
		})

		return nil, 0, false
	}

	k := ice.NewKey(encryptedNetMessageKeyType, p.decryptionKey)
	b := k.DecryptAll(encrypted)

	br = bit.NewSmallBitReader(bytes.NewReader(b))

	const (
		byteLenPadding = 1
		byteLenWritten = 4
	)

	paddingBytes := br.ReadSingleByte()

	if int(paddingBytes) >= len(b)-byteLenPadding-byteLenWritten {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: "encrypted net-message has invalid number of padding bytes",
			Type:    events.WarnTypeCantReadEncryptedNetMessage,
		})

		p.poolBitReader(br)

		return nil, 0, false
	}

	br.Skip(int(paddingBytes) << 3)

	bBytesWritten := br.ReadBytes(byteLenWritten)
	nBytesWritten = int(binary.BigEndian.Uint32(bBytesWritten))

	if len(b) != byteLenPadding+byteLenWritten+int(paddingBytes)+nBytesWritten {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: "encrypted net-message has invalid length",
			Type:    events.WarnTypeCantReadEncryptedNetMessage,
		})

		p.poolBitReader(br)

		return nil, 0, false
	}

	return br, nBytesWritten, true
}

// handleEncryptedData decrypts encrypted net-messages (e.g. chat messages in Valve matchmaking demos)
// and dispatches the inner message as if it had been received in plain text.
// The inner message uses the same framing as CDemoPacket data:
// a UBitInt message type followed by a varint size and the protobuf bytes.
func (p *parser) handleEncryptedData(m *msg.CSVCMsg_EncryptedData) {
	br, nBytesWritten, ok := p.decryptNetMessage(m.GetKeyType(), m.GetEncrypted())
	if !ok {
		return
	}

	defer p.poolBitReader(br)

	t := int32(br.ReadUBitInt()) //nolint:gosec // net-message type IDs are small, same as in handleDemoPacket()
	size := br.ReadVarInt32()

	if int(size) > nBytesWritten {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: "encrypted net-message has invalid inner message size",
			Type:    events.WarnTypeCantReadEncryptedNetMessage,
		})

		return
	}

	msgCreator := msgCreatorForType(t)
	if msgCreator == nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: fmt.Sprintf("unknown encrypted message type: %d", t),
			Type:    events.WarnTypeUnknownProtobufMessage,
		})

		return
	}

	inner := msgCreator()

	err := proto.Unmarshal(br.ReadBytes(int(size)), inner)
	if err != nil {
		p.setError(err)

		return
	}

	p.msgDispatcher.Dispatch(inner)
}
