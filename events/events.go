package events

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/common"
)

// Header parsed
type HeaderParsedEvent struct {
}

// Tick done
type TickDoneEvent struct {
}

// Match started
type MatchStartedEvent struct {
}

// Round announce match started
type RoundAnnounceMatchStartedEvent struct {
}

// Round ended
type RoundEndedEvent struct {
	message string
	reason  common.RoundEndReason
	winner  common.Team
}

func (e RoundEndedEvent) Message() string {
	return e.message
}

func (e RoundEndedEvent) Reason() common.RoundEndReason {
	return e.reason
}

func (e RoundEndedEvent) Winner() common.Team {
	return e.winner
}

// Round officially ended
type RoundOfficialyEndedEvent struct {
}

// Round MVP crowned
type RoundMVPEvent struct {
	player common.Player
	reason common.RoundMVPReason
}

func (e RoundMVPEvent) Player() common.Player {
	return e.player
}

func (e RoundMVPEvent) Reason() common.RoundMVPReason {
	return e.reason
}

// Round started
type RoundStartedEvent struct {
	timeLimit int
	fragLimit int
	objective string
}

func (e RoundStartedEvent) TimeLimit() int {
	return e.timeLimit
}

func (e RoundStartedEvent) FragLimit() int {
	return e.fragLimit
}

func (e RoundStartedEvent) Objective() string {
	return e.objective
}

// Win panel
type WinPanelMatchEvent struct {
}

// Final round
type FinalRoundEvent struct {
}

// Last round of half
type LastRoundHalfEvent struct {
}

// FreezetimeEnded
type FreezetimeEndedEvent struct {
}

// Player / team change event, occurs when a player swaps teams
type PlayerTeamChangeEvent struct {
	player  common.Player
	newTeam common.Team
	oldTeam common.Team
	silent  bool
	isBot   bool
}

func (e PlayerTeamChangeEvent) Player() common.Player {
	return e.player
}

func (e PlayerTeamChangeEvent) NewTeam() common.Team {
	return e.newTeam
}

func (e PlayerTeamChangeEvent) OldTeam() common.Team {
	return e.oldTeam
}

func (e PlayerTeamChangeEvent) Silent() bool {
	return e.silent
}

func (e PlayerTeamChangeEvent) IsBot() bool {
	return e.isBot
}

// Player killed
type PlayerKilledEvent struct {
	weapon            common.Equipment
	victim            common.Player
	killer            common.Player
	assister          common.Player
	penetratedObjects int
	isHeadshot        bool
}

func (e PlayerKilledEvent) Weapon() common.Equipment {
	return e.weapon
}

func (e PlayerKilledEvent) Victim() common.Player {
	return e.victim
}

func (e PlayerKilledEvent) Killer() common.Player {
	return e.killer
}

func (e PlayerKilledEvent) Assister() common.Player {
	return e.assister
}

func (e PlayerKilledEvent) PenetratedObjects() int {
	return e.penetratedObjects
}

func (e PlayerKilledEvent) IsHeadshot() bool {
	return e.isHeadshot
}

// Bot taken over
type BotTakenOverEvent struct {
	taker common.Player
}

func (e BotTakenOverEvent) Taker() common.Player {
	return e.taker
}

// Weapon fired
type WeaponFiredEvent struct {
	shooter common.Player
	weapon  common.Equipment
}

func (e WeaponFiredEvent) Shooter() common.Player {
	return e.shooter
}

func (e WeaponFiredEvent) Weapon() common.Equipment {
	return e.weapon
}

// Nade exploded
// TODO: velocity vector
type NadeEvent struct {
	nadeType common.EquipmentElement
	position r3.Vector
	thrower  common.Player
}

func (e NadeEvent) NadeType() common.EquipmentElement {
	return e.nadeType
}

func (e NadeEvent) Position() r3.Vector {
	return e.position
}

func (e NadeEvent) Thrower() common.Player {
	return e.thrower
}

func (e NadeEvent) IBPPlayer() common.Player {
	return e.thrower
}

// Flash exploded
type FlashEvent struct {
	nadeEvent      NadeEvent
	flashedPlayers []common.Player
}

func (e FlashEvent) NadeEvent() NadeEvent {
	return e.nadeEvent
}

func (e FlashEvent) FlashedPlayers() []common.Player {
	return e.flashedPlayers
}

// Bomb event (planted or exploded???)
type BombEvent struct {
	planter common.Player
	site    rune
}

func (e BombEvent) Planter() common.Player {
	return e.planter
}

func (e BombEvent) Site() rune {
	return e.site
}

// Bomb defused????? (Used to be BombDeEvent)
type BombDefusedEvent struct {
	defuser common.Player
	site    rune
}

func (e BombDefusedEvent) Defuser() common.Player {
	return e.defuser
}

func (e BombDefusedEvent) Site() rune {
	return e.site
}
