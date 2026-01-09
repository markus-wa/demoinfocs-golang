// Demonstrates tracking player actions and button states.
package main

import (
	"fmt"
	"os"

	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

// Run like this: go run player_actions.go -demo /path/to/demo.dem
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	type buttonAction struct {
		mask common.ButtonBitMask
		name string
	}

	buttons := []buttonAction{
		{common.ButtonNone, "None"},
		{common.ButtonAttack, "Attack"},
		{common.ButtonJump, "Jump"},
		{common.ButtonDuck, "Duck"},
		{common.ButtonForward, "Forward"},
		{common.ButtonBack, "Back"},
		{common.ButtonUse, "Use"},
		{common.ButtonTurnLeft, "TurnLeft"},
		{common.ButtonTurnRight, "TurnRight"},
		{common.ButtonMoveLeft, "MoveLeft"},
		{common.ButtonMoveRight, "MoveRight"},
		{common.ButtonAttack2, "Attack2"},
		{common.ButtonReload, "Reload"},
		{common.ButtonSpeed, "Speed"},
		{common.ButtonJoyAutoSprint, "JoyAutoSprint"},
		{common.ButtonUseOrReload, "UseOrReload"},
		{common.ButtonScore, "Score"},
		{common.ButtonZoom, "Zoom"},
		{common.ButtonLookAtWeapon, "LookAtWeapon"},
	}

	p.RegisterEventHandler(func(e events.PlayerButtonsStateUpdate) {
		actions := []string{}
		for _, button := range buttons {
			if e.Player.IsPressingButton(button.mask) {
				actions = append(actions, button.name)
			}
		}

		fmt.Printf("[Tick %d] %s actions: %v (raw state: 0x%x)\n", p.GameState().IngameTick(), e.Player.Name, actions, e.ButtonsState)
	})

	err = p.ParseToEnd()
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
