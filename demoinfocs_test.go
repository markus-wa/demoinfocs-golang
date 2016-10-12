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

func handleTickDone(interface{}) {
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

func TestDemoInfoCs(t *testing.T) {
	var demPath string
	if runtime.GOOS == "windows" {
		demPath = "C:\\Dev\\demo.dem"
	} else {
		demPath = "/home/markus/Downloads/demo.dem"
	}
	f, _ := os.Open(demPath)

	p := dem.NewParser(f)
	p.ParseHeader()

	fmt.Println("go")
	if true {
		p.EventDispatcher().Register(reflect.TypeOf(events.TickDoneEvent{}), handleTickDone)
		p.EventDispatcher().Register(reflect.TypeOf(events.BombDefusedEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.BombEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.BotTakenOverEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.FinalRoundEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.FlashEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.FreezetimeEndedEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.WinPanelMatchEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.WeaponFiredEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.SayTextEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.SayText2Event{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.RoundStartedEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.RoundOfficialyEndedEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.RoundMVPEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.RoundEndedEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.LastRoundHalfEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.MatchStartedEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.NadeEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.PlayerBindEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.PlayerDisconnectEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.PlayerHurtEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.PlayerKilledEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.PlayerTeamChangeEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.RankUpdateEvent{}), handle)
		p.EventDispatcher().Register(reflect.TypeOf(events.RoundAnnounceMatchStartedEvent{}), handle)
	}
	tsc = p.TState()
	ts := time.Now()
	p.ParseToEnd(&cancel)
	duration := time.Since(ts)
	fmt.Println("took", duration.Nanoseconds()/1000/1000, "ms")

	f.Close()
}
