package demoinfocs_test

import (
	"fmt"
	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/events"
	"os"
	"reflect"
	"runtime"
	"testing"
	"time"
)

var cancel bool = false
var tsc *dem.TeamState
var tix int = 0
var oldScore int

func handleTickDone(events.TickDoneEvent) {
	tix++
	if tix > 100 {
		//cancel = true
	}
	if tsc != nil && oldScore != tsc.Score() {
		fmt.Println(tsc.Score())
		oldScore = tsc.Score()
	}

}

func handle(interface{}) {}

func handleDetails(e interface{}) {
	n := reflect.TypeOf(e).Name()
	if len(n) > 0 && n != "TickDoneEvent" {
		fmt.Println(n, e)
	}
}

var started bool = false

func handleStart(events.MatchStartedEvent) {
	started = true
}
func handleKill(events.PlayerKilledEvent) {
	if started {
		//k := e.(events.PlayerKilledEvent)
		//fmt.Println(k.Killer, "&", k.Assister, "killed", k.Victim)
		//fmt.Println(*k.Killer, "&", k.Assister, "killed", *k.Victim)
	}
}

func TestDemoInfoCs(t *testing.T) {
	var demPath string
	if runtime.GOOS == "windows" {
		demPath = "C:\\Dev\\demo.dem"
	} else {
		demPath = "/home/markus/Downloads/demo.dem"
	}
	f, _ := os.Open(demPath)
	defer f.Close()

	p := dem.NewParser(f)
	p.ParseHeader()

	fmt.Println("go")
	if true {
		p.EventDispatcher().RegisterHandler(handleTickDone)
		p.EventDispatcher().RegisterHandler(handle)
		//p.EventDispatcher().RegisterHandler(reflect.TypeOf((*interface{})(nil)).Elem(), handleDetails)
		//p.EventDispatcher().RegisterHandler(reflect.TypeOf((*events.BombEventIf)(nil)).Elem(), handleDetails)
		//p.EventDispatcher().RegisterHandler(reflect.TypeOf((*events.NadeEventIf)(nil)).Elem(), handleDetails)
		//p.EventDispatcher().RegisterHandler(reflect.TypeOf((*events.PlayerJumpEvent)(nil)).Elem(), handleDetails)
		//p.EventDispatcher().RegisterHandler(reflect.TypeOf((*events.PlayerDisconnectEvent)(nil)).Elem(), handleDetails)
		p.EventDispatcher().RegisterHandler(handleKill)
		p.EventDispatcher().RegisterHandler(handleStart)
	}
	tsc = p.TState()
	ts := time.Now()
	p.ParseToEnd(&cancel)
	duration := time.Since(ts)
	fmt.Println("took", duration.Nanoseconds()/1000/1000, "ms")
}

func BenchmarkDemoInfoCs(b *testing.B) {
	fmt.Println("Parsing sample demo", b.N, "times")
	var demPath string
	if runtime.GOOS == "windows" {
		demPath = "C:\\Dev\\demo.dem"
	} else {
		demPath = "/home/markus/Downloads/demo.dem"
	}
	for i := 0; i < b.N; i++ {
		runDemoInfoCsBenchmark(demPath)
	}
}

func runDemoInfoCsBenchmark(path string) {
	f, _ := os.Open(path)
	defer f.Close()

	p := dem.NewParser(f)
	p.ParseHeader()
	ts := time.Now()
	p.ParseToEnd(nil)
	duration := time.Since(ts)
	fmt.Println("took", duration.Nanoseconds()/1000/1000, "ms")
}
