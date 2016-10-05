package main

import (
	"github.com/golang/geo/r3"
)

// Header parsed
type HeaderParsedEvent struct {
	header *DemoHeader
}

func (hpe HeaderParsedEvent) Header() DemoHeader {
	return *hpe.header
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
	reason  RoundEndReason
	winner  Team
}

func (e RoundEndedEvent) Message() string {
	return e.message
}

func (e RoundEndedEvent) Reason() RoundEndReason {
	return e.reason
}

func (e RoundEndedEvent) Winner() Team {
	return e.winner
}

// Round officially ended
type RoundOfficialyEndedEvent struct {
}

// Round MVP crowned
type RoundMVPEvent struct {
	player Player
	reason RoundMVPReason
}

func (e RoundMVPEvent) Player() Player {
	return e.player
}

func (e RoundMVPEvent) Reason() RoundMVPReason {
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
	swapped Player
	newTeam Team
	oldTeam Team
	silent  bool
	isBot   bool
}

func (e PlayerTeamChangeEvent) Swapped() Player {
	return e.swapped
}

func (e PlayerTeamChangeEvent) NewTeam() Team {
	return e.newTeam
}

func (e PlayerTeamChangeEvent) OldTeam() Team {
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
	weapon            Equipment
	victim            Player
	killer            Player
	assister          Player
	penetratedObjects int
	isHeadshot        bool
}

func (e PlayerKilledEvent) Weapon() Equipment {
	return e.weapon
}

func (e PlayerKilledEvent) Victim() Player {
	return e.victim
}

func (e PlayerKilledEvent) Killer() Player {
	return e.killer
}

func (e PlayerKilledEvent) Assister() Player {
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
	taker Player
}

func (e BotTakenOverEvent) Taker() Player {
	return e.taker
}

// Weapon fired
type WeaponFiredEvent struct {
	shooter Player
	weapon  Equipment
}

func (e WeaponFiredEvent) Shooter() Player {
	return e.shooter
}

func (e WeaponFiredEvent) Weapon() Equipment {
	return e.weapon
}

// Nade exploded
// TODO: velocity vector
type NadeEvent struct {
	nadeType EquipmentElement
	position r3.Vector
	thrower  Player
}

func (e NadeEvent) NadeType() EquipmentElement {
	return e.nadeType
}

func (e NadeEvent) Position() r3.Vector {
	return e.position
}

func (e NadeEvent) Thrower() Player {
	return e.thrower
}

func (e NadeEvent) IBPPlayer() Player {
	return e.thrower
}

// Flash exploded
type FlashEvent struct {
	nadeEvent      NadeEvent
	flashedPlayers []Player
}

func (e FlashEvent) NadeEvent() NadeEvent {
	return e.nadeEvent
}

func (e FlashEvent) FlashedPlayers() []Player {
	return e.flashedPlayers
}

// Bomb event (planted or exploded???)
type BombEvent struct {
	planter Player
	site    rune
}

func (e BombEvent) Planter() Player {
	return e.planter
}

func (e BombEvent) Site() rune {
	return e.site
}

// Bomb defused????? (Used to be BombDeEvent)
type BombDefusedEvent struct {
	defuser Player
	site    rune
}

func (e BombDefusedEvent) Defuser() Player {
	return e.defuser
}

func (e BombDefusedEvent) Site() rune {
	return e.site
}
