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
	player *common.Player
	reason common.RoundMVPReason
}

func (e RoundMVPEvent) Player() *common.Player {
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
	player  *common.Player
	newTeam common.Team
	oldTeam common.Team
	silent  bool
	isBot   bool
}

func (e PlayerTeamChangeEvent) Player() *common.Player {
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
	weapon            *common.Equipment
	victim            *common.Player
	killer            *common.Player
	assister          *common.Player
	penetratedObjects int
	isHeadshot        bool
}

func (e PlayerKilledEvent) Weapon() *common.Equipment {
	return e.weapon
}

func (e PlayerKilledEvent) Victim() *common.Player {
	return e.victim
}

func (e PlayerKilledEvent) Killer() *common.Player {
	return e.killer
}

func (e PlayerKilledEvent) Assister() *common.Player {
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
	taker *common.Player
}

func (e *BotTakenOverEvent) Taker() *common.Player {
	return e.taker
}

// Weapon fired
type WeaponFiredEvent struct {
	shooter *common.Player
	weapon  *common.Equipment
}

func (e WeaponFiredEvent) Shooter() *common.Player {
	return e.shooter
}

func (e WeaponFiredEvent) Weapon() *common.Equipment {
	return e.weapon
}

// Nade exploded
// TODO: velocity vector
type NadeEvent struct {
	nadeType common.EquipmentElement
	position r3.Vector
	thrower  *common.Player
}

func (e NadeEvent) NadeType() common.EquipmentElement {
	return e.nadeType
}

func (e NadeEvent) Position() r3.Vector {
	return e.position
}

func (e NadeEvent) Thrower() *common.Player {
	return e.thrower
}

func (e NadeEvent) IBPPlayer() *common.Player {
	return e.thrower
}

// Flash exploded
type FlashEvent struct {
	NadeEvent
	flashedPlayers []*common.Player
}

func (e FlashEvent) FlashedPlayers() []*common.Player {
	return e.flashedPlayers
}

// Bomb event (planted or exploded???)
type BombEvent struct {
	planter *common.Player
	site    rune
}

func (e BombEvent) Planter() *common.Player {
	return e.planter
}

func (e BombEvent) Site() rune {
	return e.site
}

// Bomb defused????? (Used to be BombDeEvent)
type BombDefusedEvent struct {
	defuser *common.Player
	site    rune
}

func (e BombDefusedEvent) Defuser() *common.Player {
	return e.defuser
}

func (e BombDefusedEvent) Site() rune {
	return e.site
}

type PlayerHurtEvent struct {
	player       *common.Player
	attacker     *common.Player
	health       int
	armor        int
	weapon       *common.Equipment
	weaponString string // Wrong for CZ, M4A1-S etc.
	healthDamage int
	armorDamage  int
	hitgroup     common.Hitgroup
}

func (e PlayerHurtEvent) Player() *common.Player {
	return e.player
}

func (e PlayerHurtEvent) Attacker() *common.Player {
	return e.attacker
}

func (e PlayerHurtEvent) Health() int {
	return e.health
}

func (e PlayerHurtEvent) Armor() int {
	return e.armor
}

func (e PlayerHurtEvent) Weapon() *common.Equipment {
	return e.weapon
}

func (e PlayerHurtEvent) WeaponString() string {
	return e.weaponString
}

func (e PlayerHurtEvent) HealthDamage() int {
	return e.healthDamage
}

func (e PlayerHurtEvent) ArmorDamage() int {
	return e.armorDamage
}

func (e PlayerHurtEvent) Hitgroup() common.Hitgroup {
	return e.hitgroup
}

type PlayerBindEvent struct {
	player *common.Player
}

func (e PlayerBindEvent) Player() *common.Player {
	return e.player
}

type PlayerDisconnectEvent struct {
	player *common.Player
}

func (e PlayerDisconnectEvent) Player() *common.Player {
	return e.player
}

type SayTextEvent struct {
	entityIndex int
	text        string
	isChat      bool
	isChatAll   bool
}

func (e SayTextEvent) EntityIndex() int {
	return e.entityIndex
}

func (e SayTextEvent) Text() string {
	return e.text
}

func (e SayTextEvent) IsChat() bool {
	return e.isChat
}

func (e SayTextEvent) IsChatAll() bool {
	return e.isChatAll
}

type SayText2Event struct {
	sender    *common.Player
	text      string
	isChat    bool
	isChatAll bool
}

func (e SayText2Event) EntityIndex() *common.Player {
	return e.sender
}

func (e SayText2Event) Text() string {
	return e.text
}

func (e SayText2Event) IsChat() bool {
	return e.isChat
}

func (e SayText2Event) IsChatAll() bool {
	return e.isChatAll
}

type RankUpdateEvent struct {
	steamId    int64
	rankOld    int
	rankNew    int
	winCount   int
	rankChange float32
}

func (e RankUpdateEvent) SteamId() int64 {
	return e.steamId
}

func (e RankUpdateEvent) RankOld() int {
	return e.rankOld
}

func (e RankUpdateEvent) RankNew() int {
	return e.rankNew
}

func (e RankUpdateEvent) WinCount() int {
	return e.winCount
}

func (e RankUpdateEvent) RankChange() float32 {
	return e.rankChange
}
