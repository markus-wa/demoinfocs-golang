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
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dispatch "github.com/markus-wa/godispatch"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

const (
	testDataPath            = "../../test"
	csDemosPath             = testDataPath + "/cs-demos"
	demSetPath              = csDemosPath + "/set"
	demSetPathS2            = csDemosPath + "/s2"
	defaultDemPath          = csDemosPath + "/default.dem"
	retakeDemPath           = csDemosPath + "/retake_unknwon_bombsite_index.dem"
	unexpectedEndOfDemoPath = csDemosPath + "/unexpected_end_of_demo.dem"
	s2DemPath               = demSetPathS2 + "/s2.dem"
)

var concurrentDemos = flag.Int("concurrentdemos", 2, "The `number` of current demos")

var update = flag.Bool("update", false, "update .golden files")

//nolint:cyclop
func TestDemoInfoCs(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	f, err := os.Open(defaultDemPath)
	assertions := assert.New(t)
	assertions.NoError(err, "error opening demo %q", defaultDemPath)

	defer mustClose(t, f)

	p := demoinfocs.NewParserWithConfig(f, demoinfocs.ParserConfig{
		MsgQueueBufferSize: 1000,
		AdditionalNetMessageCreators: map[int]demoinfocs.NetMessageCreator{
			4: func() proto.Message { return new(msg.CNETMsg_Tick) },
		},
	})

	var actual bytes.Buffer
	p.RegisterEventHandler(func(e any) {
		actual.WriteString(fmt.Sprintf("%#v\n", e))
	})

	t.Log("Parsing header")
	h, err := p.ParseHeader()
	assertions.NoError(err, "error returned by Parser.ParseHeader()")
	assertions.Equal(h, p.Header(), "values returned by ParseHeader() and Header() don't match")
	t.Logf("Header: %v - FrameRate()=%.2f frames/s ; FrameTime()=%s ; TickRate()=%.2f frames/s ; TickTime()=%s\n", h, h.FrameRate(), h.FrameTime(), p.TickRate(), p.TickTime())

	t.Log("Registering handlers")
	gs := p.GameState()
	p.RegisterEventHandler(func(e events.RoundEnd) {
		var (
			winner, loser *common.TeamState
			winnerSide    string
		)

		switch e.Winner { //nolint:exhaustive
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
			t.Log("Round finished: No winner (tie)")

			return
		}

		winnerClan := winner.ClanName()
		winnerID := winner.ID()
		winnerFlag := winner.Flag()
		ingameTime := p.CurrentTime()
		progressPercent := p.Progress() * 100
		ingameTick := gs.IngameTick()
		currentFrame := p.CurrentFrame()

		// Score + 1 for winner because it hasn't actually been updated yet
		t.Logf("Round finished: score=%d:%d ; winnerSide=%s ; clanName=%q ; teamId=%d ; teamFlag=%s ; ingameTime=%s ; progress=%.1f%% ; tick=%d ; frame=%d\n", winner.Score()+1, loser.Score(), winnerSide, winnerClan, winnerID, winnerFlag, ingameTime, progressPercent, ingameTick, currentFrame)
		if len(winnerClan) == 0 || winnerID == 0 || len(winnerFlag) == 0 || ingameTime == 0 || progressPercent == 0 || ingameTick == 0 || currentFrame == 0 {
			t.Error("Unexprected default value, check output of last round")
		}
	})

	// bomb planting checks
	p.RegisterEventHandler(func(begin events.BombPlantBegin) {
		assertions.True(begin.Player.IsPlanting, "Player started planting but IsPlanting is false")
	})
	p.RegisterEventHandler(func(abort events.BombPlantAborted) {
		assertions.False(abort.Player.IsPlanting, "Player aborted planting but IsPlanting is true")
	})
	p.RegisterEventHandler(func(planted events.BombPlanted) {
		assertions.False(planted.Player.IsPlanting, "Player finished planting but IsPlanting is true")
	})

	// airborne checks
	// we don't check RoundStart or RoundFreezetimeEnd since players may spawn airborne
	p.RegisterEventHandler(func(plantBegin events.BombPlantBegin) {
		assertions.False(plantBegin.Player.IsAirborne(), "Player can't be airborne during plant")
	})

	// reload checks
	p.RegisterEventHandler(func(reload events.WeaponReloadBegin) {
		assertions.True(reload.Player.IsReloading, "Player started reloading but IsReloading is false")
	})

	p.RegisterEventHandler(func(start events.RoundFreezetimeEnd) {
		for _, pl := range p.GameState().Participants().All() {
			assertions.False(pl.IsReloading, "Player can't be reloading at the start of the round")
		}
	})

	// PlayerFlashed checks
	p.RegisterEventHandler(func(flashed events.PlayerFlashed) {
		if flashed.Projectile.Owner != flashed.Attacker {
			t.Errorf("PlayerFlashed projectile.Owner != Attacker. tick=%d, owner=%s, attacker=%s\n", p.GameState().IngameTick(), flashed.Projectile.Owner, flashed.Attacker)
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
		assertions.Len(p.GameState().Smokes(), 0, "expected 0 Smokes at the start of the round")
	})

	// Net-message stuff
	var netTickHandlerID dispatch.HandlerIdentifier
	netTickHandlerID = p.RegisterNetMessageHandler(func(tick *msg.CNETMsg_Tick) {
		t.Log("Net-message tick handled, unregistering - tick:", tick.Tick)
		p.UnregisterNetMessageHandler(netTickHandlerID)
	})

	ts := time.Now()

	frameByFrameCount := 1000
	t.Logf("Parsing frame by frame (%d frames)\n", frameByFrameCount)

	for i := 0; i < frameByFrameCount; i++ {
		ok, err := p.ParseNextFrame()
		assertions.NoError(err, "error occurred in ParseNextFrame()")
		assertions.True(ok, "parser reported end of demo after less than %d frames", frameByFrameCount)
	}

	t.Log("Parsing to end")
	err = p.ParseToEnd()
	assertions.NoError(err, "error occurred in ParseToEnd()")

	t.Logf("Took %s\n", time.Since(ts))

	assertGolden(t, assertions, "default", actual.Bytes())
}

func TestS2(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	f, err := os.Open(s2DemPath)
	assertions := assert.New(t)
	assertions.NoError(err, "error opening demo %q", s2DemPath)

	defer mustClose(t, f)

	cfg := demoinfocs.DefaultParserConfig
	cfg.MsgQueueBufferSize = 0

	p := demoinfocs.NewParserWithConfig(f, cfg)

	if *update {
		p.RegisterNetMessageHandler(func(gel *msgs2.CMsgSource1LegacyGameEventList) {
			lo.Must0(os.WriteFile("s2_CMsgSource1LegacyGameEventList.pb.bin", lo.Must(proto.Marshal(gel)), 0600))
		})
	}

	t.Log("Parsing header")
	_, err = p.ParseHeader()
	assertions.NoError(err, "error returned by Parser.ParseHeader()")

	t.Log("Parsing to end")
	err = p.ParseToEnd()
	assertions.NoError(err, "error occurred in ParseToEnd()")
}

func TestEncryptedNetMessages(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	infoF, err := os.Open(csDemosPath + "/match730_003528806449641685104_1453182610_271.dem.info")
	assert.NoError(t, err)

	b, err := ioutil.ReadAll(infoF)
	assert.NoError(t, err)

	k, err := demoinfocs.MatchInfoDecryptionKey(b)
	assert.NoError(t, err)

	f, err := os.Open(csDemosPath + "/match730_003528806449641685104_1453182610_271.dem")
	assert.NoError(t, err)
	defer mustClose(t, f)

	cfg := demoinfocs.DefaultParserConfig
	cfg.NetMessageDecryptionKey = k

	p := demoinfocs.NewParserWithConfig(f, cfg)

	p.RegisterEventHandler(func(message events.ChatMessage) {
		t.Log(message)
	})

	err = p.ParseToEnd()
	assert.NoError(t, err)
}

func TestMatchInfoDecryptionKey_Error(t *testing.T) {
	_, err := demoinfocs.MatchInfoDecryptionKey([]byte{0})
	assert.Error(t, err)
}

func TestRetake_BadBombsiteIndex(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	f := openFile(t, retakeDemPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	err := p.ParseToEnd()
	assert.Error(t, err, demoinfocs.ErrBombsiteIndexNotFound)
}

func TestRetake_IgnoreBombsiteIndexNotFound(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	f := openFile(t, retakeDemPath)
	defer mustClose(t, f)

	cfg := demoinfocs.DefaultParserConfig
	cfg.IgnoreErrBombsiteIndexNotFound = true

	p := demoinfocs.NewParserWithConfig(f, cfg)

	err := p.ParseToEnd()
	assert.NoError(t, err)
}

func TestUnexpectedEndOfDemo(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	f := openFile(t, unexpectedEndOfDemoPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	err := p.ParseToEnd()
	assert.ErrorIs(t, err, demoinfocs.ErrUnexpectedEndOfDemo, "parsing cancelled but error was not ErrUnexpectedEndOfDemo")
}

func TestBadNetMessageDecryptionKey(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	const (
		demPath  = csDemosPath + "/match730_003528806449641685104_1453182610_271.dem"
		infoPath = csDemosPath + "/match730_003449478367177343081_1946274414_112.dem.info"
	)

	infoF, err := os.Open(infoPath)
	assert.NoError(t, err)

	b, err := ioutil.ReadAll(infoF)
	assert.NoError(t, err)

	k, err := demoinfocs.MatchInfoDecryptionKey(b)
	assert.NoError(t, err)

	f, err := os.Open(demPath)
	assert.NoError(t, err)

	defer f.Close()

	cfg := demoinfocs.DefaultParserConfig
	cfg.NetMessageDecryptionKey = k

	p := demoinfocs.NewParserWithConfig(f, cfg)

	var cantReadEncNetMsgWarns []events.ParserWarn

	p.RegisterEventHandler(func(warn events.ParserWarn) {
		if warn.Type == events.WarnTypeCantReadEncryptedNetMessage {
			cantReadEncNetMsgWarns = append(cantReadEncNetMsgWarns, warn)
		}
	})

	err = p.ParseToEnd()
	assert.NoError(t, err)

	assert.NotEmpty(t, cantReadEncNetMsgWarns)
}

func TestParseToEnd_Cancel(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	f := openFile(t, defaultDemPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	var (
		tix      = 0
		maxTicks = 100
	)

	var handlerID dispatch.HandlerIdentifier
	handlerID = p.RegisterEventHandler(func(events.FrameDone) {
		tix++
		if tix == maxTicks {
			p.Cancel()
			p.UnregisterEventHandler(handlerID)
		}
	})

	err := p.ParseToEnd()
	assert.Equal(t, demoinfocs.ErrCancelled, err, "parsing cancelled but error was not ErrCancelled")
	assert.True(t, tix == maxTicks, "FrameDone handler was not triggered the correct amount of times")
}

// See https://github.com/markus-wa/demoinfocs-golang/issues/276
func TestParseToEnd_MultiCancel(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	f := openFile(t, defaultDemPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	var handlerID dispatch.HandlerIdentifier
	handlerID = p.RegisterEventHandler(func(events.FrameDone) {
		p.Cancel()
		p.Cancel()
		p.Cancel()
		p.Cancel()
		p.Cancel()
		p.UnregisterEventHandler(handlerID)
	})

	err := p.ParseToEnd()
	assert.Equal(t, demoinfocs.ErrCancelled, err, "parsing cancelled but error was not ErrCancelled")
}

func TestInvalidFileType(t *testing.T) {
	t.Parallel()

	invalidDemoData := make([]byte, 2048)
	_, err := rand.Read(invalidDemoData)
	assert.NoError(t, err, "failed to create/read random data")

	msgWrongError := "invalid demo but error was not ErrInvalidFileType"

	p := demoinfocs.NewParser(bytes.NewBuffer(invalidDemoData))
	_, err = p.ParseHeader()
	assert.Equal(t, demoinfocs.ErrInvalidFileType, err, msgWrongError)

	p = demoinfocs.NewParser(bytes.NewBuffer(invalidDemoData))
	_, err = p.ParseNextFrame()
	assert.Equal(t, demoinfocs.ErrInvalidFileType, err, msgWrongError)

	p = demoinfocs.NewParser(bytes.NewBuffer(invalidDemoData))
	err = p.ParseToEnd()
	assert.Equal(t, demoinfocs.ErrInvalidFileType, err, msgWrongError)
}

func TestConcurrent(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	t.Logf("Running concurrency test with %d demos\n", *concurrentDemos)

	var i int64
	runner := func() {
		n := atomic.AddInt64(&i, 1)
		t.Logf("Starting concurrent runner %d\n", n)

		ts := time.Now()

		parseDefaultDemo(t)

		t.Logf("Runner %d took %s\n", n, time.Since(ts))
	}

	runConcurrently(runner)
}

func parseDefaultDemo(tb testing.TB) {
	tb.Helper()

	f := openFile(tb, defaultDemPath)
	defer mustClose(tb, f)

	p := demoinfocs.NewParser(f)

	err := p.ParseToEnd()
	assert.NoError(tb, err, "ParseToEnd() returned an error")
}

func runConcurrently(runner func()) {
	var wg sync.WaitGroup

	for i := 0; i < *concurrentDemos; i++ {
		wg.Add(1)

		go func() { runner(); wg.Done() }()
	}

	wg.Wait()
}

func testDemoSet(t *testing.T, path string) {
	dems, err := os.ReadDir(path)
	assert.NoError(t, err, "failed to list directory %q", path)

	for _, d := range dems {
		name := d.Name()
		if strings.HasSuffix(name, ".dem") {
			t.Logf("Parsing '%s/%s'\n", path, name)
			func() {
				f := openFile(t, fmt.Sprintf("%s/%s", path, name))
				defer mustClose(t, f)

				defer func() {
					assert.Nil(t, recover(), "parsing of '%s/%s' panicked", path, name)
				}()

				p := demoinfocs.NewParser(f)

				p.RegisterEventHandler(func(warn events.ParserWarn) {
					switch warn.Type {
					case events.WarnTypeBombsiteUnknown:
						if p.Header().MapName == "de_grind" {
							t.Log("expected known issue with bomb sites on de_grind occurred:", warn.Message)

							return
						}

					case events.WarnTypeTeamSwapPlayerNil:
						t.Log("expected known issue with team swaps occurred:", warn.Message)
						return

					case events.WarnTypeGameEventBeforeDescriptors:
						if strings.Contains(name, "POV-orbit-skytten-vs-cloud9-gfinity15sm1-nuke.dem") {
							t.Log("expected known issue for POV demos occurred:", warn.Message)

							return
						}

					default:
						t.Error("unexpected parser warning occurred:", warn.Message)
					}
				})

				err = p.ParseToEnd()
				assert.NoError(t, err, "parsing of '%s/%s' failed", demSetPath, name)
			}()
		}
	}
}

func TestDemoSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	t.Parallel()

	testDemoSet(t, demSetPath)
}

func TestDemoSetS2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	t.Parallel()

	testDemoSet(t, demSetPathS2)
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
		p := demoinfocs.NewParser(bytes.NewReader(d))

		err = p.ParseToEnd()
		assert.NoError(b, err, "ParseToEnd() returned an error")
	}
}

func BenchmarkConcurrent(b *testing.B) {
	b.Logf("Running concurrency benchmark with %d demos\n", *concurrentDemos)

	for i := 0; i < b.N; i++ {
		runConcurrently(func() { parseDefaultDemo(b) })
	}
}

func openFile(tb testing.TB, file string) *os.File {
	tb.Helper()

	f, err := os.Open(file)
	assert.NoError(tb, err, "failed to open file %q", file)

	return f
}

func assertGolden(tb testing.TB, assertions *assert.Assertions, testCase string, actual []byte) {
	tb.Helper()

	const goldenVerificationGoVersionMin = "go1.12"
	if ver := runtime.Version(); strings.Compare(ver, goldenVerificationGoVersionMin) < 0 {
		tb.Logf("old go version %q detected, skipping golden file verification", ver)
		tb.Logf("need at least version %q to compare against golden file", goldenVerificationGoVersionMin)

		return
	}

	// fmt adds pointer addresses when printing with %v, we need to remove them for the comparison
	actual = removePointers(actual)

	goldenFile := fmt.Sprintf("%s/%s.golden", testDataPath, testCase)

	if *update {
		f, err := os.OpenFile(goldenFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		assertions.NoError(err, "error creating/opening %q", goldenFile)

		w := gzip.NewWriter(f)

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
			expectedFile := fmt.Sprintf("%s/%s.expected", testDataPath, testCase)
			actualFile := fmt.Sprintf("%s/%s.actual", testDataPath, testCase)
			writeFile(assertions, expectedFile, expected)
			writeFile(assertions, actualFile, actual)
			assertions.Fail("mismatch between golden and actual", "data did not match contents of %q; please check the diff between %q and %q", goldenFile, expectedFile, actualFile)
		}
	}
}

func removePointers(s []byte) []byte {
	r := regexp.MustCompile(`\(0x[\da-f]+\)`)

	return r.ReplaceAll(s, []byte("(non-nil)"))
}

func writeFile(assertions *assert.Assertions, file string, data []byte) {
	err := ioutil.WriteFile(file, data, 0600)
	assertions.NoError(err, "failed to write to file %q", file)
}

func mustClose(tb testing.TB, closables ...io.Closer) {
	tb.Helper()

	mustCloseAssert(assert.New(tb), closables...)
}

func mustCloseAssert(assertions *assert.Assertions, closables ...io.Closer) {
	for _, c := range closables {
		assertions.NoError(c.Close(), "failed to close file, reader or writer")
	}
}
