package sendtables

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/fuzzcheck"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

// The files in testdata/s2 were extracted from s2.dem (test/cs-demos/s2/s2.dem):
//   - sendtables.bin: the data of DEM_SendTables (the schema, see Parser.ParsePacket())
//   - classinfo.pb: DEM_ClassInfo
//   - baselines.bin: the instance baselines (varints class ID and size, followed by the data)
//   - packetentities.bin: the first 20 svc_PacketEntities messages, see readPacketEntities()
const s2MaxClasses = 227

// s2Testdata is read once, it's used by every input of the fuzz targets.
var s2Testdata = sync.OnceValues(func() (map[string][]byte, error) {
	m := map[string][]byte{}

	for _, name := range []string{"sendtables.bin", "classinfo.pb", "baselines.bin", "packetentities.bin"} {
		f, err := os.Open("testdata/s2/" + name + ".gz")
		if err != nil {
			return nil, err
		}

		r, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}

		m[name], err = io.ReadAll(r)

		_ = f.Close()

		if err != nil {
			return nil, err
		}
	}

	return m, nil
})

func s2File(tb testing.TB, name string) []byte {
	tb.Helper()

	m, err := s2Testdata()
	require.NoError(tb, err)

	return m[name]
}

// readPacketEntities decodes a stream of svc_PacketEntities messages:
// varint updated entries, a byte that's 1 for delta updates, varint data size, followed by the entity data.
// The stream ends at the first message that's incomplete.
func readPacketEntities(b []byte) (msgs []*msg.CSVCMsg_PacketEntities) {
	r := bytes.NewReader(b)

	for {
		updates, err := binary.ReadUvarint(r)
		if err != nil {
			return msgs
		}

		delta, err := r.ReadByte()
		if err != nil {
			return msgs
		}

		size, err := binary.ReadUvarint(r)
		if err != nil || size > uint64(r.Len()) {
			return msgs
		}

		data := make([]byte, size)
		_, _ = r.Read(data)

		msgs = append(msgs, &msg.CSVCMsg_PacketEntities{
			UpdatedEntries: proto.Int32(int32(updates)),
			LegacyIsDelta:  proto.Bool(delta == 1),
			EntityData:     data,
		})
	}
}

// newS2Parser returns a parser with the schema, classes and baselines of s2.dem, using the given schema data.
func newS2Parser(tb testing.TB, sendTables []byte) (*Parser, error) {
	tb.Helper()

	p := NewParser()

	require.NoError(tb, p.OnServerInfo(&msg.CSVCMsg_ServerInfo{MaxClasses: proto.Int32(s2MaxClasses)}))

	err := p.ParsePacket(sendTables)
	if err != nil {
		return nil, err
	}

	var classInfo msg.CDemoClassInfo

	require.NoError(tb, proto.Unmarshal(s2File(tb, "classinfo.pb"), &classInfo))
	require.NoError(tb, p.OnDemoClassInfo(&classInfo))

	r := bytes.NewReader(s2File(tb, "baselines.bin"))

	for r.Len() > 0 {
		classID, err := binary.ReadUvarint(r)
		require.NoError(tb, err)

		size, err := binary.ReadUvarint(r)
		require.NoError(tb, err)

		data := make([]byte, size)
		_, err = io.ReadFull(r, data)
		require.NoError(tb, err)

		p.SetInstanceBaseline(int(classID), data)
	}

	// bind a property of each new entity, so update handlers run
	p.OnEntity(func(e *Entity, op EntityOp) error {
		if op&EntityOpCreated != 0 {
			for _, prop := range e.Properties() {
				prop.OnUpdate(func(PropertyValue) {})

				break
			}
		}

		return nil
	})

	return p, nil
}

// applyPacketEntities applies the messages, it returns the first error.
// Failures can be panics (e.g. of ParsePacket()), they're returned as error by the boundaries in package demoinfocs.
func applyPacketEntities(p *Parser, msgs []*msg.CSVCMsg_PacketEntities) (err error) {
	defer func() {
		if recover() != nil {
			err = errPanicked
		}
	}()

	for _, m := range msgs {
		err = p.OnPacketEntities(m)
		if err != nil {
			return err
		}
	}

	return nil
}

var errPanicked = errors.New("panicked")

// The real data decodes without errors.
func TestS2Testdata(t *testing.T) {
	t.Parallel()

	p, err := newS2Parser(t, s2File(t, "sendtables.bin"))
	require.NoError(t, err)

	msgs := readPacketEntities(s2File(t, "packetentities.bin"))
	require.Len(t, msgs, 20)
	require.NoError(t, applyPacketEntities(p, msgs))
	require.Len(t, p.entities, 161)
}

// FuzzSendTables parses a fuzzed schema, and applies the first svc_PacketEntities message of s2.dem with it.
//
//	go test -tags unassert_panic -run '^$' -fuzz '^FuzzSendTables$' -fuzztime 15m ./pkg/demoinfocs/sendtables
//
// Not parallel, see fuzzcheck.AllocatedBytes().
func FuzzSendTables(f *testing.F) {
	f.Add(s2File(f, "sendtables.bin"))

	first := readPacketEntities(s2File(f, "packetentities.bin"))[:1]

	f.Fuzz(func(t *testing.T, sendTables []byte) {
		fuzzcheck.Run(t, len(sendTables), func() {
			var p *Parser

			func() {
				defer func() { _ = recover() }()

				p, _ = newS2Parser(t, sendTables)
			}()

			if p != nil {
				_ = applyPacketEntities(p, first)
			}
		})
	})
}

// packetEntitiesParser is the parser of FuzzPacketEntities, it's created once per process (parsing the schema is slow)
// and reset for each input.
var packetEntitiesParser struct {
	sync.Mutex
	p *Parser
}

// FuzzPacketEntities applies fuzzed svc_PacketEntities messages (see readPacketEntities()) with the schema of s2.dem.
//
//	go test -tags unassert_panic -run '^$' -fuzz '^FuzzPacketEntities$' -fuzztime 15m ./pkg/demoinfocs/sendtables
//
// Not parallel, see fuzzcheck.AllocatedBytes().
func FuzzPacketEntities(f *testing.F) {
	stream := s2File(f, "packetentities.bin")
	f.Add(stream)

	first := readPacketEntities(stream)[0]
	firstOnly := binary.AppendUvarint(nil, uint64(first.GetUpdatedEntries()))
	firstOnly = append(firstOnly, 0) // not a delta
	firstOnly = binary.AppendUvarint(firstOnly, uint64(len(first.GetEntityData())))
	f.Add(append(firstOnly, first.GetEntityData()...))

	f.Fuzz(func(t *testing.T, data []byte) {
		packetEntitiesParser.Lock()
		defer packetEntitiesParser.Unlock()

		if packetEntitiesParser.p == nil {
			p, err := newS2Parser(t, s2File(t, "sendtables.bin"))
			require.NoError(t, err)

			packetEntitiesParser.p = p
		}

		p := packetEntitiesParser.p
		p.entities = make(map[int32]*Entity)
		p.entityFullPackets = 0
		p.entityBudget = newEntityBudget()

		msgs := readPacketEntities(data)

		fuzzcheck.Run(t, len(data), func() {
			_ = applyPacketEntities(p, msgs)
		})
	})
}
