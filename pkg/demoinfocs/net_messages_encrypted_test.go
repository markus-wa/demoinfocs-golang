package demoinfocs

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/markus-wa/ice-cipher-go/pkg/ice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
)

var testDecryptionKey = []byte("0123456789ABCDEF")

// bitWriter writes bits LSB-first, matching the bit reader.
type bitWriter struct {
	buf   []byte
	nBits int
}

func (w *bitWriter) writeBits(v uint64, n int) {
	for i := 0; i < n; i++ {
		if w.nBits%8 == 0 {
			w.buf = append(w.buf, 0)
		}

		if v&(1<<i) != 0 {
			w.buf[len(w.buf)-1] |= 1 << (w.nBits % 8)
		}

		w.nBits++
	}
}

// writeUBitInt is the inverse of BitReader.ReadUBitInt for values < 256.
func (w *bitWriter) writeUBitInt(v uint64) {
	if v < 16 {
		w.writeBits(v, 6)

		return
	}

	w.writeBits(v&15|16, 6)
	w.writeBits(v>>4, 4)
}

func (w *bitWriter) writeVarInt(v uint64) {
	for _, b := range protowire.AppendVarint(nil, v) {
		w.writeBits(uint64(b), 8)
	}
}

// innerNetMessage frames a protobuf message the way encrypted net-messages carry it.
func innerNetMessage(t *testing.T, msgType int, m proto.Message, sizeOverride int) []byte {
	t.Helper()

	b, err := proto.Marshal(m)
	require.NoError(t, err)

	size := len(b)
	if sizeOverride >= 0 {
		size = sizeOverride
	}

	w := &bitWriter{}
	w.writeUBitInt(uint64(msgType))
	w.writeVarInt(uint64(size))

	for _, x := range b {
		w.writeBits(uint64(x), 8)
	}

	return w.buf
}

// encryptNetMessage builds the plaintext layout [padding len][padding][BE uint32 len][inner]
// so that it is a whole number of ICE blocks, and encrypts it.
func encryptNetMessage(key []byte, inner []byte) []byte {
	const headerLen = 1 + 4

	padding := (8 - (headerLen+len(inner))%8) % 8

	plain := []byte{byte(padding)}
	plain = append(plain, bytes.Repeat([]byte{0xAA}, padding)...)
	plain = binary.BigEndian.AppendUint32(plain, uint32(len(inner)))
	plain = append(plain, inner...)

	k := ice.NewKey(encryptedNetMessageKeyType, key)
	enc := make([]byte, len(plain))

	for i := 0; i < len(plain); i += 8 {
		k.Encrypt(plain[i:i+8], enc[i:i+8])
	}

	return enc
}

type encryptedTestResult struct {
	sayText []*msg.CUserMessageSayText2
	warns   []events.ParserWarn
	err     error
}

func parseEncrypted(t *testing.T, parserKey []byte, data *msg.CSVCMsg_EncryptedData) encryptedTestResult {
	t.Helper()

	p := NewParser(rand.Reader, option.NetMessageDecryptionKey(parserKey)).(*parser)

	var res encryptedTestResult

	p.RegisterNetMessageHandler(func(m *msg.CUserMessageSayText2) { res.sayText = append(res.sayText, m) })
	p.RegisterEventHandler(func(w events.ParserWarn) { res.warns = append(res.warns, w) })

	assert.NotPanics(t, func() { p.handleEncryptedData(data) })

	res.err = p.error()

	return res
}

func sayTextPayload(t *testing.T, sizeOverride int) []byte {
	t.Helper()

	return innerNetMessage(t, int(msg.EBaseUserMessages_UM_SayText2), &msg.CUserMessageSayText2{
		Messagename: proto.String("Cstrike_Chat_All"),
		Param1:      proto.String("player"),
		Param2:      proto.String("gg"),
	}, sizeOverride)
}

func TestEncryptedNetMessage_CorrectKeyDispatchesInnerMessage(t *testing.T) {
	data := &msg.CSVCMsg_EncryptedData{
		Encrypted: encryptNetMessage(testDecryptionKey, sayTextPayload(t, -1)),
		KeyType:   proto.Int32(encryptedNetMessageKeyType),
	}

	res := parseEncrypted(t, testDecryptionKey, data)

	require.Len(t, res.sayText, 1)
	assert.Equal(t, "gg", res.sayText[0].GetParam2())
	assert.Empty(t, res.warns)
	assert.NoError(t, res.err)
}

func TestEncryptedNetMessage_WrongKeyWarnsAndKeepsParsing(t *testing.T) {
	data := &msg.CSVCMsg_EncryptedData{
		Encrypted: encryptNetMessage(testDecryptionKey, sayTextPayload(t, -1)),
		KeyType:   proto.Int32(encryptedNetMessageKeyType),
	}

	res := parseEncrypted(t, []byte("FEDCBA9876543210"), data)

	assert.Empty(t, res.sayText)
	require.Len(t, res.warns, 1)
	assert.Equal(t, events.WarnType(events.WarnTypeCantReadEncryptedNetMessage), res.warns[0].Type)
	assert.NoError(t, res.err, "a wrong key must not abort parsing")
}

func TestEncryptedNetMessage_MissingKeyWarns(t *testing.T) {
	data := &msg.CSVCMsg_EncryptedData{
		Encrypted: encryptNetMessage(testDecryptionKey, sayTextPayload(t, -1)),
		KeyType:   proto.Int32(encryptedNetMessageKeyType),
	}

	res := parseEncrypted(t, nil, data)

	assert.Empty(t, res.sayText)
	require.Len(t, res.warns, 1)
	assert.Equal(t, events.WarnType(events.WarnTypeMissingNetMessageDecryptionKey), res.warns[0].Type)
	assert.NoError(t, res.err)
}

func TestEncryptedNetMessage_MalformedPayloadsWarn(t *testing.T) {
	cases := map[string][]byte{
		"empty":                     {},
		"shorter than header":       {0x01, 0x02, 0x03},
		"inner size exceeds data":   encryptNetMessage(testDecryptionKey, sayTextPayload(t, 500)),
		"inner header past end":     encryptNetMessage(testDecryptionKey, []byte{0x3F}), // UBitInt needs 28 more bits
		"inner proto is garbage":    encryptNetMessage(testDecryptionKey, innerGarbageProto(t)),
		"ciphertext not block size": append(encryptNetMessage(testDecryptionKey, sayTextPayload(t, -1)), 0x00),
	}

	for name, enc := range cases {
		t.Run(name, func(t *testing.T) {
			res := parseEncrypted(t, testDecryptionKey, &msg.CSVCMsg_EncryptedData{
				Encrypted: enc,
				KeyType:   proto.Int32(encryptedNetMessageKeyType),
			})

			assert.Empty(t, res.sayText)
			require.Len(t, res.warns, 1)
			assert.Equal(t, events.WarnType(events.WarnTypeCantReadEncryptedNetMessage), res.warns[0].Type)
			assert.NoError(t, res.err)
		})
	}
}

// innerGarbageProto frames bytes that are not a valid CUserMessageSayText2.
func innerGarbageProto(t *testing.T) []byte {
	t.Helper()

	w := &bitWriter{}
	w.writeUBitInt(uint64(msg.EBaseUserMessages_UM_SayText2))

	garbage := []byte{0x0A, 0xFF, 0xFF, 0xFF, 0xFF, 0x0F} // field 1, length-delimited, absurd length
	w.writeVarInt(uint64(len(garbage)))

	for _, x := range garbage {
		w.writeBits(uint64(x), 8)
	}

	return w.buf
}
