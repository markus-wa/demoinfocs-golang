package demoinfocs_test

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	dispatch "github.com/markus-wa/godispatch"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

const csDemosPath = "cs-demos"
const demSetPath = csDemosPath + "/set"
const defaultDemPath = csDemosPath + "/default.dem"
const unexpectedEndOfDemoPath = csDemosPath + "/unexpected_end_of_demo.dem"
const valveMatchmakingDemoPath = csDemosPath + "/valve_matchmaking.dem"

var concurrentDemos int

func init() {
	flag.IntVar(&concurrentDemos, "concurrentdemos", 2, "The `number` of current demos")
	flag.Parse()

	if _, err := os.Stat(defaultDemPath); err != nil {
		panic(fmt.Sprintf("Failed to read test demo %q", defaultDemPath))
	}
}

func TestDemoInfoCs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	f, err := os.Open(defaultDemPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p := dem.NewParserWithConfig(f, dem.ParserConfig{
		MsgQueueBufferSize: 1000,
		AdditionalNetMessageCreators: map[int]dem.NetMessageCreator{
			4: func() proto.Message { return new(msg.CNETMsg_Tick) },
		},
	})

	fmt.Println("Parsing header")
	h, err := p.ParseHeader()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Header: %v - FrameRate()=%.2f frames/s ; FrameTime()=%s ; TickRate()=%.2f frames/s ; TickTime()=%s\n", h, h.FrameRate(), h.FrameTime(), h.TickRate(), h.TickTime())
	h2 := p.Header()
	if h != h2 {
		t.Errorf("Headers returned by ParseHeader() & Header(), respectively, aren't equal; ParseHeader(): %v - Header(): %v", h, h2)
	}

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
		if len(all) <= len(players) {
			// We know the default demo has spectators
			t.Error("Expected more participants than players (spectators)")
		}
		if nPlayers := len(players); nPlayers != 10 {
			// We know there should be 10 players at match start in the default demo
			t.Error("Expected 10 players; got", nPlayers)
		}
		if nTerrorists := len(participants.TeamMembers(common.TeamTerrorists)); nTerrorists != 5 {
			// We know there should be 5 terrorists at match start in the default demo
			t.Error("Expected 5 terrorists; got", nTerrorists)
		}
		if nCTs := len(participants.TeamMembers(common.TeamCounterTerrorists)); nCTs != 5 {
			// We know there should be 5 CTs at match start in the default demo
			t.Error("Expected 5 CTs; got", nCTs)
		}
	})

	// Regression test for grenade projectiles not being deleted at the end of the round (issue #42)
	p.RegisterEventHandler(func(events.RoundStart) {
		if nProjectiles := len(p.GameState().GrenadeProjectiles()); nProjectiles > 0 {
			t.Error("Expected 0 GrenadeProjectiles at the start of the round, got", nProjectiles)
		}

		if nInfernos := len(p.GameState().Infernos()); nInfernos > 0 {
			t.Error("Expected 0 Infernos at the start of the round, got", nInfernos)
		}
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
		ok, errFrame := p.ParseNextFrame()
		if errFrame != nil {
			t.Fatal(errFrame)
		}
		if !ok {
			t.Fatalf("Parser reported end of demo after less than %d frames", frameByFrameCount)
		}
	}

	fmt.Println("Parsing to end")
	err = p.ParseToEnd()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Took %s\n", time.Since(ts))
}

func TestUnexpectedEndOfDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	f, err := os.Open(unexpectedEndOfDemoPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p := dem.NewParser(f)

	err = p.ParseToEnd()
	if err != dem.ErrUnexpectedEndOfDemo {
		t.Fatal("Parsing cancelled but error was not ErrUnexpectedEndOfDemo:", err)
	}
}

func TestCancelParseToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	f, err := os.Open(defaultDemPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

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

	err = p.ParseToEnd()
	if err != dem.ErrCancelled {
		t.Error("Parsing cancelled but error was not ErrCancelled:", err)
	}
	if tix > maxTicks {
		t.Error("FrameDone handler was triggered after being unregistered")
	}
}

func TestInvalidFileType(t *testing.T) {
	invalidDemoData := make([]byte, 2048)
	_, err := rand.Read(invalidDemoData)
	if err != nil {
		t.Fatal("Failed to read random data:", err)
	}

	msgWrongError := "Invalid demo but error was not ErrInvalidFileType:"

	p := dem.NewParser(bytes.NewBuffer(invalidDemoData))
	_, err = p.ParseHeader()
	if err != dem.ErrInvalidFileType {
		t.Fatal("ParseHeader():", msgWrongError, err)
	}

	p = dem.NewParser(bytes.NewBuffer(invalidDemoData))
	_, err = p.ParseNextFrame()
	if err != dem.ErrInvalidFileType {
		t.Fatal("ParseNextFrame():", msgWrongError, err)
	}

	p = dem.NewParser(bytes.NewBuffer(invalidDemoData))
	err = p.ParseToEnd()
	if err != dem.ErrInvalidFileType {
		t.Fatal("ParseToEnd():", msgWrongError, err)
	}
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
	f, err := os.Open(defaultDemPath)
	if err != nil {
		tb.Fatal(err)
	}
	defer f.Close()

	p := dem.NewParser(f)

	err = p.ParseToEnd()
	if err != nil {
		tb.Fatal("Parsing failed:", err)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range dems {
		name := d.Name()
		if strings.HasSuffix(name, ".dem") {
			fmt.Printf("Parsing '%s/%s'\n", demSetPath, name)
			func() {
				var f *os.File
				f, err = os.Open(demSetPath + "/" + name)
				if err != nil {
					t.Error(err)
				}
				defer f.Close()

				defer func() {
					pErr := recover()
					if pErr != nil {
						t.Errorf("Parsing of '%s/%s' paniced: %s\n", demSetPath, name, pErr)
					}
				}()

				p := dem.NewParser(f)

				err = p.ParseToEnd()
				if err != nil {
					t.Errorf("Parsing of '%s/%s' failed: %s\n", demSetPath, name, err)
					return
				}
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
	f, err := os.Open(defaultDemPath)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	inf, err := f.Stat()
	if err != nil {
		b.Fatal(err)
	}

	d := make([]byte, inf.Size())
	n, err := f.Read(d)
	if err != nil || int64(n) != inf.Size() {
		b.Fatal(fmt.Sprintf("Expected %d bytes, got %d", inf.Size(), n), err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := dem.NewParser(bytes.NewReader(d))

		err = p.ParseToEnd()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrent(b *testing.B) {
	b.Logf("Running concurrency benchmark with %d demos\n", concurrentDemos)

	for i := 0; i < b.N; i++ {
		runConcurrently(func() { parseDefaultDemo(b) })
	}
}
