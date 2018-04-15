// Package events contains all events that can be sent out from demoinfocs.Parser.
package events

import (
	r3 "github.com/golang/geo/r3"

	common "github.com/markus-wa/demoinfocs-golang/common"
)

// HeaderParsedEvent signals that the header has been parsed.
// Deprecated, use Parser.Header() instead.
type HeaderParsedEvent struct {
	Header common.DemoHeader
}

// TickDoneEvent signals that a tick is done.
type TickDoneEvent struct{}

// MatchStartedEvent signals that the match has started.
type MatchStartedEvent struct{}

// RoundAnnounceMatchStartedEvent signals that the announcement "Match Started" has been displayed.
type RoundAnnounceMatchStartedEvent struct{}

// RoundEndedEvent signals that the last round is over
type RoundEndedEvent struct {
	Message string
	Reason  common.RoundEndReason
	Winner  common.Team
}

// RoundOfficialyEndedEvent signals that the round 'has officially ended', not exactly sure what that is tbh.
type RoundOfficialyEndedEvent struct{}

// RoundMVPEvent signals the announcement of the last rounds MVP.
type RoundMVPEvent struct {
	Player *common.Player
	Reason common.RoundMVPReason
}

// RoundStartedEvent signals that a new round has started.
type RoundStartedEvent struct {
	TimeLimit int
	FragLimit int
	Objective string
}

// WinPanelMatchEvent signals that the 'win panel' has been displayed. I guess that's the final scoreboard.
type WinPanelMatchEvent struct{}

// FinalRoundEvent signals the 30th round, not raised if the match ends before that.
type FinalRoundEvent struct{}

// LastRoundHalfEvent signals the last round of the first half.
type LastRoundHalfEvent struct{}

// FreezetimeEndedEvent signals that the freeze time is over.
type FreezetimeEndedEvent struct{}

// PlayerFootstepEvent occurs when a player makes a footstep
type PlayerFootstepEvent struct {
	Player *common.Player
}

// PlayerTeamChangeEvent occurs when a player swaps teams.
type PlayerTeamChangeEvent struct {
	Player  *common.Player
	NewTeam common.Team
	OldTeam common.Team
	Silent  bool
	IsBot   bool
}

// PlayerJumpEvent signals that a player has jumped.
type PlayerJumpEvent struct {
	Player *common.Player
}

// PlayerKilledEvent signals that a player has been killed.
type PlayerKilledEvent struct {
	Weapon            *common.Equipment
	Victim            *common.Player
	Killer            *common.Player
	Assister          *common.Player
	PenetratedObjects int
	IsHeadshot        bool
}

// BotTakenOverEvent signals that a player took over a bot.
type BotTakenOverEvent struct {
	Taker *common.Player
}

// WeaponFiredEvent signals that a weapon has been fired.
type WeaponFiredEvent struct {
	Shooter *common.Player
	Weapon  *common.Equipment
}

// FIXME: Currently handling NadeEventIf is really annoying. Improve that
// Same with BombEventIf

// NadeEventIf is the interface for all NadeEvents. Used to catch
// the different events with the same handler.
type NadeEventIf interface {
	dummyNade()
}

// NadeEvent contains the common attributes of nade events. Dont register
// handlers on this tho, you want NadeEventIf for that
type NadeEvent struct {
	NadeType common.EquipmentElement
	Position r3.Vector
	Thrower  *common.Player
}

// Make NadeEvents implement NadeEventIf
func (NadeEvent) dummyNade() {}

// HeExplodedEvent signals the explosion of a HE.
type HeExplodedEvent struct {
	NadeEvent
}

// FlashExplodedEvent signals the explosion of a Flash.
type FlashExplodedEvent struct {
	NadeEvent
}

// DecoyStartEvent signals the start of a decoy.
type DecoyStartEvent struct {
	NadeEvent
}

// DecoyEndEvent signals the end of a decoy.
type DecoyEndEvent struct {
	NadeEvent
}

// SmokeStartEvent signals the start of a smoke (pop).
type SmokeStartEvent struct {
	NadeEvent
}

// SmokeEndEvent signals the end of a smoke (fade). Not sure if this means
// it started to fade, completely faded or something in between.
type SmokeEndEvent struct {
	NadeEvent
}

// FireNadeStartEvent signals the start of a molly/incendiary.
type FireNadeStartEvent struct {
	NadeEvent
}

// FireNadeEndEvent signals the end of a molly/incendiary.
type FireNadeEndEvent struct {
	NadeEvent
}

// PlayerFlashedEvent signals that a player was flashed.
type PlayerFlashedEvent struct {
	Player *common.Player
}

// BombEventIf is the interface for alll the bomb events. Like NadeEventIf for NadeEvents.
type BombEventIf interface {
	dummyBomb()
}

// BombEvent contains the common attributes of bomb events. Dont register
// handlers on this tho, you want BombEventIf for that.
type BombEvent struct {
	Player *common.Player
	Site   rune
}

// Make BombEvent implement BombEventIf
func (BombEvent) dummyBomb() {}

// BombBeginPlant signals the start of a plant.
type BombBeginPlant struct {
	BombEvent
}

// BombAbortPlant signals the abortion of a plant.
type BombAbortPlant struct {
	BombEvent
}

// BombPlantedEvent signals that the bomb has been planted.
type BombPlantedEvent struct {
	BombEvent
}

// BombDefusedEvent signals that the bomb has been defused.
type BombDefusedEvent struct {
	BombEvent
}

// BombExplodedEvent signals that the bomb has exploded.
type BombExplodedEvent struct {
	BombEvent
}

// BombBeginDefuseEvent signals the start of defusing.
type BombBeginDefuseEvent struct {
	Defuser *common.Player
	HasKit  bool
}

// BombAbortDefuseEvent signals that defusing was aborted.
type BombAbortDefuseEvent struct {
	Defuser *common.Player
	HasKit  bool
}

// PlayerHurtEvent signals that a player has been damaged.
type PlayerHurtEvent struct {
	Player       *common.Player
	Attacker     *common.Player
	Health       int
	Armor        int
	Weapon       *common.Equipment
	WeaponString string // Wrong for CZ, M4A1-S etc.
	HealthDamage int
	ArmorDamage  int
	HitGroup     common.HitGroup
}

// PlayerBindEvent signals that a player has connected.
type PlayerBindEvent struct {
	Player *common.Player
}

// PlayerDisconnectEvent signals that a player has disconnected.
type PlayerDisconnectEvent struct {
	Player *common.Player
}

// SayTextEvent signals a chat message. It contains the raw
// network message data for admin / console messages.
// EntityIndex will probably always be 0
// See ChatMessageEvent and SayText2Event for player chat messages.
type SayTextEvent struct {
	EntityIndex int
	Text        string
	IsChat      bool
	IsChatAll   bool
}

// SayText2Event signals a chat message. It just contains the raw network message
// for player chat messages, ChatMessageEvent may be more interesting.
// Team chat is generally not recorded so IsChatAll will probably always be false.
// See SayTextEvent for admin / console messages.
type SayText2Event struct {
	Sender    *common.Player
	MsgName   string
	Params    []string
	IsChat    bool
	IsChatAll bool
}

// ChatMessageEvent signals a player generated chat message.
// Since team chat is generally not recorded IsChatAll will probably always be false.
// See SayTextEvent for admin / console messages and SayText2Event for raw network packages.
type ChatMessageEvent struct {
	Sender    *common.Player
	Text      string
	IsChatAll bool
}

// RankUpdateEvent signals the new rank. Not sure if this
// only occurs if the rank changed.
type RankUpdateEvent struct {
	SteamID    int64
	RankOld    int
	RankNew    int
	WinCount   int
	RankChange float32
}

// ItemEquipEvent signals an item was equipped
type ItemEquipEvent struct {
	Weapon common.Equipment
	Player *common.Player
}

// ItemPickupEvent signals an item was bought or picked up.
type ItemPickupEvent struct {
	Weapon common.Equipment
	Player *common.Player
}

// ItemDropEvent signals an item was dropped
type ItemDropEvent struct {
	Weapon common.Equipment
	Player *common.Player
}
