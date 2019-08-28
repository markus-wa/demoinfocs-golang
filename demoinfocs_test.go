package demoinfocs_test

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	dispatch "github.com/markus-wa/godispatch"
	"github.com/stretchr/testify/assert"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

const csDemosPath = "test/cs-demos"
const demSetPath = csDemosPath + "/set"
const defaultDemPath = csDemosPath + "/default.dem"
const unexpectedEndOfDemoPath = csDemosPath + "/unexpected_end_of_demo.dem"

var concurrentDemos int

func init() {
	flag.IntVar(&concurrentDemos, "concurrentdemos", 2, "The `number` of current demos")
	flag.Parse()

	if _, err := os.Stat(defaultDemPath); err != nil {
		panic(fmt.Sprintf("Failed to read test demo %q", defaultDemPath))
	}
}

var update = flag.Bool("update", false, "update .golden files")

func TestDemoInfoCs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	f, err := os.Open(defaultDemPath)
	assertions := assert.New(t)
	assertions.NoError(err, "error opening demo %q", defaultDemPath)
	defer mustClose(t, f)

	p := dem.NewParserWithConfig(f, dem.ParserConfig{
		MsgQueueBufferSize: 1000,
		AdditionalNetMessageCreators: map[int]dem.NetMessageCreator{
			4: func() proto.Message { return new(msg.CNETMsg_Tick) },
		},
	})

	var actual bytes.Buffer
	p.RegisterEventHandler(func(e interface{}) {
		actual.WriteString(fmt.Sprintf("%#v\n", e))
	})

	fmt.Println("Parsing header")
	h, err := p.ParseHeader()
	assertions.NoError(err, "error returned by Parser.ParseHeader()")
	assertions.Equal(h, p.Header(), "values returned by ParseHeader() and Header() don't match")
	fmt.Printf("Header: %v - FrameRate()=%.2f frames/s ; FrameTime()=%s ; TickRate()=%.2f frames/s ; TickTime()=%s\n", h, h.FrameRate(), h.FrameTime(), h.TickRate(), h.TickTime())

	fmt.Println("Registering handlers")
	gs := p.GameState()
	p.RegisterEventHandler(func(e events.RoundEnd) {
		var winner *common.TeamState
		var loser *common.TeamState
		var winnerSide string
		switch e.Winner {
		case common.TeamTerrorists:
			winner = gs.TeamTerrorists()
			loser = gs.TeamCounterTerrorists()
			winnerSide = "T"
		case common.TeamCounterTerrorists:
			winner = gs.TeamCounterTerrorists()
			loser = gs.TeamTerrorists()
			winnerSide = "CT"
		default:
			// Probably match medic or something similar
			fmt.Println("Round finished: No winner (tie)")
			return
		}
		winnerClan := winner.ClanName
		winnerID := winner.ID
		winnerFlag := winner.Flag
		ingameTime := p.CurrentTime()
		progressPercent := p.Progress() * 100
		ingameTick := gs.IngameTick()
		currentFrame := p.CurrentFrame()
		// Score + 1 for winner because it hasn't actually been updated yet
		fmt.Printf("Round finished: score=%d:%d ; winnerSide=%s ; clanName=%q ; teamId=%d ; teamFlag=%s ; ingameTime=%s ; progress=%.1f%% ; tick=%d ; frame=%d\n", winner.Score+1, loser.Score, winnerSide, winnerClan, winnerID, winnerFlag, ingameTime, progressPercent, ingameTick, currentFrame)
		if len(winnerClan) == 0 || winnerID == 0 || len(winnerFlag) == 0 || ingameTime == 0 || progressPercent == 0 || ingameTick == 0 || currentFrame == 0 {
			t.Error("Unexprected default value, check output of last round")
		}
	})

	// Check some things at match start
	p.RegisterEventHandler(func(events.MatchStart) {
		participants := gs.Participants()
		all := participants.All()
		players := participants.Playing()

		// We know the default demo has spectators
		assertions.False(len(all) <= len(players), "expected more participants than players (spectators)")

		// the default demo has 10 players (5 Ts + 5 CTs) at match start (this doesn't have to be the case for all demos)
		assertions.Len(players, 10, "expected 10 players")
		assertions.Len(participants.TeamMembers(common.TeamTerrorists), 5, "expected 5 terrorists")
		assertions.Len(participants.TeamMembers(common.TeamCounterTerrorists), 5, "expected 5 CTs")
	})

	// Regression test for grenade projectiles not being deleted at the end of the round (issue #42)
	p.RegisterEventHandler(func(events.RoundStart) {
		assertions.Len(p.GameState().GrenadeProjectiles(), 0, "expected 0 GrenadeProjectiles at the start of the round")
		assertions.Len(p.GameState().Infernos(), 0, "expected 0 Infernos at the start of the round")
	})

	// Net-message stuff
	var netTickHandlerID dispatch.HandlerIdentifier
	netTickHandlerID = p.RegisterNetMessageHandler(func(tick *msg.CNETMsg_Tick) {
		fmt.Println("Net-message tick handled, unregistering - tick:", tick.Tick)
		p.UnregisterNetMessageHandler(netTickHandlerID)
	})

	ts := time.Now()

	frameByFrameCount := 1000
	fmt.Printf("Parsing frame by frame (%d frames)\n", frameByFrameCount)
	for i := 0; i < frameByFrameCount; i++ {
		ok, err := p.ParseNextFrame()
		assertions.NoError(err, "error occurred in ParseNextFrame()")
		assertions.True(ok, "parser reported end of demo after less than %d frames", frameByFrameCount)
	}

	fmt.Println("Parsing to end")
	err = p.ParseToEnd()
	assertions.NoError(err, "error occurred in ParseToEnd()")

	fmt.Printf("Took %s\n", time.Since(ts))

	assertGolden(assertions, "default", actual.Bytes())
}

func TestUnexpectedEndOfDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	f := openFile(t, unexpectedEndOfDemoPath)
	defer mustClose(t, f)

	p := dem.NewParser(f)

	err := p.ParseToEnd()
	assert.Equal(t, dem.ErrUnexpectedEndOfDemo, err, "parsing cancelled but error was not ErrUnexpectedEndOfDemo")
}

func TestCancelParseToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	f := openFile(t, defaultDemPath)
	defer mustClose(t, f)

	p := dem.NewParser(f)

	maxTicks := 100
	var tix int

	var handlerID dispatch.HandlerIdentifier
	handlerID = p.RegisterEventHandler(func(events.FrameDone) {
		tix++
		if tix == maxTicks {
			p.Cancel()
			p.UnregisterEventHandler(handlerID)
		}
	})

	err := p.ParseToEnd()
	assert.Equal(t, dem.ErrCancelled, err, "parsing cancelled but error was not ErrCancelled")
	assert.True(t, tix == maxTicks, "FrameDone handler was not triggered the correct amount of times")
}

func TestInvalidFileType(t *testing.T) {
	invalidDemoData := make([]byte, 2048)
	_, err := rand.Read(invalidDemoData)
	assert.NoError(t, err, "failed to create/read random data")

	msgWrongError := "invalid demo but error was not ErrInvalidFileType"

	p := dem.NewParser(bytes.NewBuffer(invalidDemoData))
	_, err = p.ParseHeader()
	assert.Equal(t, dem.ErrInvalidFileType, err, msgWrongError)

	p = dem.NewParser(bytes.NewBuffer(invalidDemoData))
	_, err = p.ParseNextFrame()
	assert.Equal(t, dem.ErrInvalidFileType, err, msgWrongError)

	p = dem.NewParser(bytes.NewBuffer(invalidDemoData))
	err = p.ParseToEnd()
	assert.Equal(t, dem.ErrInvalidFileType, err, msgWrongError)
}

func TestConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	t.Logf("Running concurrency test with %d demos\n", concurrentDemos)

	var i int64
	runner := func() {
		n := atomic.AddInt64(&i, 1)
		fmt.Printf("Starting concurrent runner %d\n", n)

		ts := time.Now()

		parseDefaultDemo(t)

		fmt.Printf("Runner %d took %s\n", n, time.Since(ts))
	}

	runConcurrently(runner)
}

func parseDefaultDemo(tb testing.TB) {
	f := openFile(tb, defaultDemPath)
	defer mustClose(tb, f)

	p := dem.NewParser(f)

	err := p.ParseToEnd()
	assert.NoError(tb, err, "ParseToEnd() returned an error")
}

func runConcurrently(runner func()) {
	var wg sync.WaitGroup
	for i := 0; i < concurrentDemos; i++ {
		wg.Add(1)
		go func() { runner(); wg.Done() }()
	}
	wg.Wait()
}

func TestDemoSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	dems, err := ioutil.ReadDir(demSetPath)
	assert.NoError(t, err, "failed to list directory %q", demSetPath)

	for _, d := range dems {
		name := d.Name()
		if strings.HasSuffix(name, ".dem") {
			fmt.Printf("Parsing '%s/%s'\n", demSetPath, name)
			func() {
				f := openFile(t, fmt.Sprintf("%s/%s", demSetPath, name))
				defer mustClose(t, f)

				defer func() {
					assert.Nil(t, recover(), "parsing of '%s/%s' paniced", demSetPath, name)
				}()

				p := dem.NewParser(f)

				err = p.ParseToEnd()
				assert.Nil(t, err, "parsing of '%s/%s' failed", demSetPath, name)
			}()
		}
	}
}

func BenchmarkDemoInfoCs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseDefaultDemo(b)
	}
}

func BenchmarkInMemory(b *testing.B) {
	f := openFile(b, defaultDemPath)
	defer mustClose(b, f)

	inf, err := f.Stat()
	assert.NoError(b, err, "failed to stat file %q", defaultDemPath)

	d := make([]byte, inf.Size())
	n, err := f.Read(d)
	assert.NoError(b, err, "failed to read file %q", defaultDemPath)
	assert.Equal(b, int64(n), inf.Size(), "byte count not as expected")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := dem.NewParser(bytes.NewReader(d))

		err = p.ParseToEnd()
		assert.NoError(b, err, "ParseToEnd() returned an error")
	}
}

func BenchmarkConcurrent(b *testing.B) {
	b.Logf("Running concurrency benchmark with %d demos\n", concurrentDemos)

	for i := 0; i < b.N; i++ {
		runConcurrently(func() { parseDefaultDemo(b) })
	}
}

func openFile(tb testing.TB, file string) *os.File {
	f, err := os.Open(file)
	assert.NoError(tb, err, "failed to open file %q", file)
	return f
}

func assertGolden(assertions *assert.Assertions, testCase string, actual []byte) {
	// fmt adds pointer addresses when printing with %v, we need to remove them for the comparison
	actual = removePointers(actual)

	goldenFile := fmt.Sprintf("test/%s.golden", testCase)
	if *update {
		f, err := os.OpenFile(goldenFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		assertions.NoError(err, "error creating/opening %q", goldenFile)

		w := gzip.NewWriter(f)
		assertions.NoError(err, "error writing/updating %q", goldenFile)

		_, err = w.Write(actual)
		assertions.NoError(err, "error writing gzip data to %q", goldenFile)

		mustCloseAssert(assertions, w, f)
		assertions.NoError(err, "error closing gzip writer for %q", goldenFile)
	} else {
		f, err := os.Open(goldenFile)
		assertions.NoError(err, "error opening %q", goldenFile)

		gzipReader, err := gzip.NewReader(f)
		assertions.NoError(err, "error creating gzip reader for %q", goldenFile)

		expected, err := ioutil.ReadAll(gzipReader)
		assertions.NoError(err, "error reading gzipped data from %q", goldenFile)

		mustCloseAssert(assertions, gzipReader, f)

		if !assert.ObjectsAreEqual(expected, actual) {
			expectedFile := fmt.Sprintf("test/%s.expected", testCase)
			actualFile := fmt.Sprintf("test/%s.actual", testCase)
			writeFile(assertions, expectedFile, expected)
			writeFile(assertions, actualFile, actual)
			assertions.Fail("data did not match contents of %q; please check the diff between %q and %q", goldenFile, expectedFile, actualFile)
		}
	}
}

func removePointers(s []byte) []byte {
	r := regexp.MustCompile(`\(0x[\da-f]{10}\)`)
	return r.ReplaceAll(s, []byte("(non-nil)"))
}

func writeFile(assertions *assert.Assertions, file string, data []byte) {
	err := ioutil.WriteFile(file, data, 0755)
	assertions.NoError(err, "failed to write to file %q", file)
}

func mustClose(tb testing.TB, closables ...io.Closer) {
	mustCloseAssert(assert.New(tb), closables...)
}

func mustCloseAssert(assertions *assert.Assertions, closables ...io.Closer) {
	for _, c := range closables {
		assertions.NoError(c.Close(), "failed to close file, reader or writer")
	}
}
