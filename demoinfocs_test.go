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

	proto "github.com/gogo/protobuf/proto"
	dispatch "github.com/markus-wa/godispatch"

	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	fuzzy "github.com/markus-wa/demoinfocs-golang/fuzzy"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
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
		var winner *dem.TeamState
		var loser *dem.TeamState
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
		winnerClan := winner.ClanName()
		winnerId := winner.ID()
		winnerFlag := winner.Flag()
		ingameTime := p.CurrentTime()
		progressPercent := p.Progress() * 100
		ingameTick := gs.IngameTick()
		currentFrame := p.CurrentFrame()
		// Score + 1 for winner because it hasn't actually been updated yet
		fmt.Printf("Round finished: score=%d:%d ; winnerSide=%s ; clanName=%q ; teamId=%d ; teamFlag=%s ; ingameTime=%s ; progress=%.1f%% ; tick=%d ; frame=%d\n", winner.Score()+1, loser.Score(), winnerSide, winnerClan, winnerId, winnerFlag, ingameTime, progressPercent, ingameTick, currentFrame)
		if len(winnerClan) == 0 || winnerId == 0 || len(winnerFlag) == 0 || ingameTime == 0 || progressPercent == 0 || ingameTick == 0 || currentFrame == 0 {
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
	var done int64
	go func() {
		// 5 minute timeout (for a really slow machine with race condition testing)
		timer := time.NewTimer(time.Minute * 5)
		<-timer.C
		if atomic.LoadInt64(&done) == 0 {
			t.Error("Parsing timeout")
			p.Cancel()
			timer.Reset(time.Second * 1)
			<-timer.C
			t.Fatal("Parser locked up for more than one second after cancellation")
		}
	}()

	frameByFrameCount := 1000
	fmt.Printf("Parsing frame by frame (%d frames)\n", frameByFrameCount)
	for i := 0; i < frameByFrameCount; i++ {
		ok, err := p.ParseNextFrame()
		if err != nil {
			t.Fatal(err)
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

	atomic.StoreInt64(&done, 1)
	fmt.Printf("Took %s\n", time.Since(ts))
}

func TestUnexpectedEndOfDemo(t *testing.T) {
	f, err := os.Open(unexpectedEndOfDemoPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p := dem.NewParser(f)
	_, err = p.ParseHeader()
	if err != nil {
		t.Fatal(err)
	}

	err = p.ParseToEnd()
	if err != dem.ErrUnexpectedEndOfDemo {
		t.Fatal("Parsing cancelled but error was not ErrUnexpectedEndOfDemo:", err)
	}
}

func TestValveMatchmakingFuzzyEmitters(t *testing.T) {
	f, err := os.Open(valveMatchmakingDemoPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cfg := dem.DefaultParserConfig
	cfg.AdditionalEventEmitters = []dem.EventEmitter{new(fuzzy.ValveMatchmakingTeamSwitchEmitter)}

	p := dem.NewParserWithConfig(f, cfg)
	_, err = p.ParseHeader()
	if err != nil {
		t.Fatal(err)
	}

	teamSwitchDone := false
	tScoreBeforeSwap, ctScoreBeforeSwap := -1, -1
	p.RegisterEventHandler(func(ev events.RoundEnd) {
		switch ev.Winner {
		case common.TeamTerrorists:
			tScoreBeforeSwap = p.GameState().TeamTerrorists().Score() + 1

		case common.TeamCounterTerrorists:
			ctScoreBeforeSwap = p.GameState().TeamCounterTerrorists().Score() + 1
		}
	})

	p.RegisterEventHandler(func(fuzzy.TeamSwitchEvent) {
		teamSwitchDone = true
		if tScoreBeforeSwap != p.GameState().TeamCounterTerrorists().Score() {
			t.Error("T-Score before swap != CT-Score after swap")
		}
		if ctScoreBeforeSwap != p.GameState().TeamTerrorists().Score() {
			t.Error("CT-Score before swap != T-Score after swap")
		}
	})

	err = p.ParseToEnd()
	if err != nil {
		t.Fatal(err)
	}

	if !teamSwitchDone {
		t.Fatal("TeamSwitchEvent not received")
	}
}

func TestCancelParseToEnd(t *testing.T) {
	f, err := os.Open(defaultDemPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p := dem.NewParser(f)
	_, err = p.ParseHeader()
	if err != nil {
		t.Fatal(err)
	}

	maxTicks := 100
	var tix int

	var handlerID dispatch.HandlerIdentifier
	handlerID = p.RegisterEventHandler(func(events.TickDone) {
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
		t.Error("TickDoneEvent handler was triggered after being unregistered")
	}
}

func TestInvalidFileType(t *testing.T) {
	invalidDemoData := make([]byte, 2048)
	rand.Read(invalidDemoData)

	p := dem.NewParser(bytes.NewBuffer(invalidDemoData))

	_, err := p.ParseHeader()
	if err != dem.ErrInvalidFileType {
		t.Fatal("Invalid demo but error was not ErrInvalidFileType:", err)
	}
}

func TestHeaderNotParsed(t *testing.T) {
	f, err := os.Open(defaultDemPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p := dem.NewParser(f)

	_, err = p.ParseNextFrame()
	if err != dem.ErrHeaderNotParsed {
		t.Fatal("Tried to parse tick before header but error was not ErrHeaderNotParsed:", err)
	}

	err = p.ParseToEnd()
	if err != dem.ErrHeaderNotParsed {
		t.Fatal("Tried to parse tick before header but error was not ErrHeaderNotParsed:", err)
	}
}

func TestConcurrent(t *testing.T) {
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

	_, err = p.ParseHeader()
	if err != nil {
		tb.Fatal(err)
	}

	err = p.ParseToEnd()
	if err != nil {
		tb.Fatal(err)
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
				_, err = p.ParseHeader()
				if err != nil {
					t.Error(err)
					return
				}

				err = p.ParseToEnd()
				if err != nil {
					t.Error(err)
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

		_, err = p.ParseHeader()
		if err != nil {
			b.Fatal(err)
		}

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
