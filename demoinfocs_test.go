package demoinfocs_test

import (
	"bytes"
	"fmt"
	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/events"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const demPath = "test/demo.dem"

func TestDemoInfoCs(t *testing.T) {
	f, err := os.Open(demPath)
	defer f.Close()
	if err != nil {
		t.Fatal(err)
	}

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
	var done int64
	go func() {
		timer := time.NewTimer(time.Minute * 2)
		<-timer.C
		if atomic.LoadInt64(&done) == 0 {
			cancel = true
			timer.Reset(time.Second * 1)
			<-timer.C
			t.Fatal("Parsing timeout")
		}
	}()

	fmt.Println("Parsing to end")
	p.ParseToEnd(&cancel)

	atomic.StoreInt64(&done, 1)
	fmt.Printf("Took %s\n", time.Since(ts))
}

func TestCancelParseToEnd(t *testing.T) {
	f, err := os.Open(demPath)
	defer f.Close()
	if err != nil {
		t.Fatal(err)
	}

	p := dem.NewParser(f)
	p.ParseHeader()

	maxTicks := 100
	var tix int
	var cancel bool

	p.RegisterEventHandler(func(events.TickDoneEvent) {
		tix++
		if tix == maxTicks {
			cancel = true
		} else if tix > maxTicks {
			t.Fatal("Parsing continued after cancellation")
		}
	})

	defer func() { recover() }()
	p.ParseToEnd(&cancel)
}

func TestConcurrent(t *testing.T) {
	var i int64
	runner := func() {
		f, _ := os.Open(demPath)
		defer f.Close()

		p := dem.NewParser(f)
		p.ParseHeader()

		n := atomic.AddInt64(&i, 1)
		fmt.Printf("Starting runner %d\n", n)

		ts := time.Now()
		p.ParseToEnd(nil)
		fmt.Printf("Runner %d took %s\n", n, time.Since(ts))
	}

	var wg sync.WaitGroup
	for j := 0; j < 2; j++ {
		wg.Add(1)
		go func() { runner(); wg.Done() }()
	}
	wg.Wait()
}

func BenchmarkDemoInfoCs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		func() {
			f, err := os.Open(demPath)
			defer f.Close()
			if err != nil {
				b.Fatal(err)
			}

			p := dem.NewParser(f)
			p.ParseHeader()

			ts := time.Now()
			p.ParseToEnd(nil)
			b.Logf("Took %s\n", time.Since(ts))
		}()
	}
}

func BenchmarkInMemory(b *testing.B) {
	f, err := os.Open(demPath)
	defer f.Close()
	if err != nil {
		b.Fatal(err)
	}

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
		p.ParseHeader()

		ts := time.Now()
		p.ParseToEnd(nil)
		b.Logf("Took %s\n", time.Since(ts))
	}
}
