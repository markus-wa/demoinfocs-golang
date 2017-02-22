package events

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/common"
)

// Header parsed
type HeaderParsedEvent struct {
	Header common.DemoHeader
}

// Tick done
type TickDoneEvent struct{}

// Match started
type MatchStartedEvent struct{}

// Round announce match started
type RoundAnnounceMatchStartedEvent struct{}

// Round ended
type RoundEndedEvent struct {
	Message string
	Reason  common.RoundEndReason
	Winner  common.Team
}

// Round officially ended
type RoundOfficialyEndedEvent struct{}

// Round MVP crowned
type RoundMVPEvent struct {
	Player *common.Player
	Reason common.RoundMVPReason
}

// Round started
type RoundStartedEvent struct {
	TimeLimit int
	FragLimit int
	Objective string
}

// Win panel
type WinPanelMatchEvent struct{}

// 30th round, not raised if the match ends before that
type FinalRoundEvent struct{}

// Last round of half
type LastRoundHalfEvent struct{}

// FreezetimeEnded
type FreezetimeEndedEvent struct{}

// Player / team change event, occurs when a player swaps teams
type PlayerTeamChangeEvent struct {
	Player  *common.Player
	NewTeam common.Team
	OldTeam common.Team
	Silent  bool
	IsBot   bool
}

type PlayerJumpEvent struct {
	Player *common.Player
}

// Player killed
type PlayerKilledEvent struct {
	Weapon            *common.Equipment
	Victim            *common.Player
	Killer            *common.Player
	Assister          *common.Player
	PenetratedObjects int
	IsHeadshot        bool
}

// Bot taken over
type BotTakenOverEvent struct {
	Taker *common.Player
}

// Weapon fired
type WeaponFiredEvent struct {
	Shooter *common.Player
	Weapon  *common.Equipment
}

type NadeEventIf interface {
	dummyNade()
}

// Nade exploded
type NadeEvent struct {
	NadeType common.EquipmentElement
	Position r3.Vector
	Thrower  *common.Player
}

// Make NadeEvents implement NadeEventIf
func (NadeEvent) dummyNade() {}

type HeExplodedEvent struct {
	NadeEvent
}

type FlashExplodedEvent struct {
	NadeEvent
}

type DecoyStartEvent struct {
	NadeEvent
}

type DecoyEndEvent struct {
	NadeEvent
}

type SmokeStartEvent struct {
	NadeEvent
}

type SmokeEndEvent struct {
	NadeEvent
}

type FireNadeStartEvent struct {
	NadeEvent
}

type FireNadeEndEvent struct {
	NadeEvent
}

// Player was flashed
type PlayerFlashedEvent struct {
	Player *common.Player
}

type BombEventIf interface {
	dummyBomb()
}

type BombEvent struct {
	Player *common.Player
	Site   rune
}

func (BombEvent) dummyBomb() {}

type BombBeginPlant struct {
	BombEvent
}

type BombAbortPlant struct {
	BombEvent
}

type BombPlantedEvent struct {
	BombEvent
}

type BombDefusedEvent struct {
	BombEvent
}

type BombExplodedEvent struct {
	BombEvent
}

type BombBeginDefuse struct {
	Defuser *common.Player
	HasKit  bool
}

type BombAbortDefuse struct {
	Defuser *common.Player
	HasKit  bool
}

// Player has been damaged
type PlayerHurtEvent struct {
	Player       *common.Player
	Attacker     *common.Player
	Health       int
	Armor        int
	Weapon       *common.Equipment
	WeaponString string // Wrong for CZ, M4A1-S etc.
	HealthDamage int
	ArmorDamage  int
	Hitgroup     common.Hitgroup
}

// Player connected
type PlayerBindEvent struct {
	Player *common.Player
}

// Player disconnected
type PlayerDisconnectEvent struct {
	Player *common.Player
}

type SayTextEvent struct {
	EntityIndex int
	Text        string
	IsChat      bool
	IsChatAll   bool
}

type SayText2Event struct {
	Sender    *common.Player
	Text      string
	IsChat    bool
	IsChatAll bool
}

type RankUpdateEvent struct {
	SteamId    int64
	RankOld    int
	RankNew    int
	WinCount   int
	RankChange float32
}
