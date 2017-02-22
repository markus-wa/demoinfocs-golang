package demoinfocs_test

import (
	"fmt"
	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/events"
	"os"
	"reflect"
	"testing"
	"time"
)

const demPath = "test/demo.dem"

func handleDetails(e interface{}) {
	n := reflect.TypeOf(e).Name()
	if len(n) > 0 && n != "TickDoneEvent" {
		fmt.Println(n, e)
	}
}

func TestDemoInfoCs(t *testing.T) {
	f, _ := os.Open(demPath)
	defer f.Close()

	p := dem.NewParser(f)

	fmt.Println("Parsing header")
	p.ParseHeader()

	fmt.Println("Registering handlers")
	var tState *dem.TeamState
	var oldScore int
	p.RegisterEventHandler(func(events.TickDoneEvent) {
		if tState != nil && oldScore != tState.Score() {
			fmt.Println("T-side score: ", tState.Score())
			oldScore = tState.Score()
		}
	})
	tState = p.TState()

	ts := time.Now()
	cancel := false
	go func() {
		timer := time.NewTimer(time.Second * 8)
		<-timer.C
		cancel = true
		timer = time.NewTimer(time.Second * 2)
		<-timer.C
		t.Fatal("Parsing timeout")
	}()

	fmt.Println("Parsing to end")
	p.ParseToEnd(&cancel)

	fmt.Println("Took", time.Since(ts).Nanoseconds()/1000/1000, "ms")
}

func TestCancelParseToEnd(t *testing.T) {
	runTest(func(p *dem.Parser) {
		p.ParseHeader()
		var tix int = 0
		var cancel bool
		p.RegisterEventHandler(func(events.TickDoneEvent) {
			tix++
			if tix == 100 {
				cancel = true
			} else if tix > 100 {
				t.Fatal("Parsing continued after cancellation")
			}
		})
		defer func() { recover() }()
		p.ParseToEnd(&cancel)
	})
}

func runTest(test func(*dem.Parser)) {
	f, _ := os.Open(demPath)
	defer f.Close()

	test(dem.NewParser(f))
}

func BenchmarkDemoInfoCs(b *testing.B) {
	fmt.Println("Parsing sample demo", b.N, "times")
	for i := 0; i < b.N; i++ {
		runDemoInfoCsBenchmark()
	}
}

func runDemoInfoCsBenchmark() {
	f, _ := os.Open(demPath)
	defer f.Close()

	p := dem.NewParser(f)
	p.ParseHeader()

	ts := time.Now()
	p.ParseToEnd(nil)
	fmt.Println("Took", time.Since(ts).Nanoseconds()/1000/1000, "ms")
}
