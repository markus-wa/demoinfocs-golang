package demoinfocs_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/markus-wa/godispatch"

	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

const csDemosPath = "test/cs-demos"
const demSetPath = csDemosPath + "/set"
const defaultDemPath = csDemosPath + "/default.dem"
const unexpectedEndOfDemoPath = csDemosPath + "/unexpected_end_of_demo.dem"

func init() {
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

	p := dem.NewParser(f)

	fmt.Println("Parsing header")
	h, err := p.ParseHeader()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Header: %v - FrameRate()=%.2f frames/s ; FrameTime()=%.1fms\n", h, h.FrameRate(), h.FrameTime()*1000)
	h2 := p.Header()
	if h != h2 {
		t.Errorf("Headers returned by ParseHeader() & Header(), respectively, aren't equal; ParseHeader(): %v - Header(): %v", h, h2)
	}

	fmt.Println("Registering handlers")
	gs := p.GameState()
	p.RegisterEventHandler(func(e events.RoundEndedEvent) {
		var winner *dem.TeamState
		var loser *dem.TeamState
		var winnerSide string
		switch e.Winner {
		case common.TeamTerrorists:
			winner = gs.TState()
			loser = gs.CTState()
			winnerSide = "T"
		case common.TeamCounterTerrorists:
			winner = gs.CTState()
			loser = gs.TState()
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
		fmt.Printf("Round finished: score=%d:%d ; winnerSide=%s ; clanName=%q ; teamId=%d ; teamFlag=%s ; ingameTime=%.1fs ; progress=%.1f%% ; tick=%d ; frame=%d\n", winner.Score()+1, loser.Score(), winnerSide, winnerClan, winnerId, winnerFlag, ingameTime, progressPercent, ingameTick, currentFrame)
		if len(winnerClan) == 0 || winnerId == 0 || len(winnerFlag) == 0 || ingameTime == 0 || progressPercent == 0 || ingameTick == 0 || currentFrame == 0 {
			t.Error("Unexprected default value, check output of last round")
		}
	})
	// Check some things at match start
	p.RegisterEventHandler(func(events.MatchStartedEvent) {
		participants := gs.Participants()
		players := gs.PlayingParticipants()
		if len(participants) <= len(players) {
			// We know the default demo has spectators
			t.Error("Expected more participants than players (spectators)")
		}
		if nPlayers := len(players); nPlayers != 10 {
			// We know there should be 10 players at match start in the default demo
			t.Error("Expected 10 players; got", nPlayers)
		}
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
	handlerID = p.RegisterEventHandler(func(events.TickDoneEvent) {
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

func TestConcurrent(t *testing.T) {
	var i int64
	runner := func() {
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

		n := atomic.AddInt64(&i, 1)
		fmt.Printf("Starting runner %d\n", n)

		ts := time.Now()

		err = p.ParseToEnd()
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("Runner %d took %s\n", n, time.Since(ts))
	}

	var wg sync.WaitGroup
	for j := 0; j < 2; j++ {
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
						t.Errorf("Failed to parse '%s/%s': %s\n", demSetPath, name, pErr)
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
		func() {
			f, err := os.Open(defaultDemPath)
			if err != nil {
				b.Fatal(err)
			}
			defer f.Close()

			p := dem.NewParser(f)

			_, err = p.ParseHeader()
			if err != nil {
				b.Fatal(err)
			}

			ts := time.Now()

			err = p.ParseToEnd()
			if err != nil {
				b.Fatal(err)
			}

			b.Logf("Took %s\n", time.Since(ts))
		}()
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

		ts := time.Now()

		err = p.ParseToEnd()
		if err != nil {
			b.Fatal(err)
		}

		b.Logf("Took %s\n", time.Since(ts))
	}
}
