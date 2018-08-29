// Package events contains all events that can be sent out from demoinfocs.Parser.
package events

import (
	r3 "github.com/golang/geo/r3"

	common "github.com/markus-wa/demoinfocs-golang/common"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

// TickDoneEvent signals that a tick is done.
type TickDoneEvent struct{}

// MatchStartedEvent signals that the match has started.
type MatchStartedEvent struct{}

// RoundAnnounceMatchStartedEvent signals that the announcement "Match Started" has been displayed.
type RoundAnnounceMatchStartedEvent struct{}

// RoundEndReason is the type for the various RoundEndReasonXYZ constants.
//
// See RoundEndedEvent.
type RoundEndReason byte

// RoundEndReason constants give information about why a round ended (Bomb defused, exploded etc.).
const (
	RoundEndReasonTargetBombed         RoundEndReason = 1
	RoundEndReasonVIPEscaped           RoundEndReason = 2
	RoundEndReasonVIPKilled            RoundEndReason = 3
	RoundEndReasonTerroristsEscaped    RoundEndReason = 4
	RoundEndReasonCTStoppedEscape      RoundEndReason = 5
	RoundEndReasonTerroristsStopped    RoundEndReason = 6
	RoundEndReasonBombDefused          RoundEndReason = 7
	RoundEndReasonCTWin                RoundEndReason = 8
	RoundEndReasonTerroristsWin        RoundEndReason = 9
	RoundEndReasonDraw                 RoundEndReason = 10
	RoundEndReasonHostagesRescued      RoundEndReason = 11
	RoundEndReasonTargetSaved          RoundEndReason = 12
	RoundEndReasonHostagesNotRescued   RoundEndReason = 13
	RoundEndReasonTerroristsNotEscaped RoundEndReason = 14
	RoundEndReasonVIPNotEscaped        RoundEndReason = 15
	RoundEndReasonGameStart            RoundEndReason = 16
	RoundEndReasonTerroristsSurrender  RoundEndReason = 17
	RoundEndReasonCTSurrender          RoundEndReason = 18
)

// RoundEndedEvent signals that a round just finished.
// Attention: TeamState.Score() won't be up to date yet after this.
// Add +1 to the winner's score as a workaround.
type RoundEndedEvent struct {
	Message string
	Reason  RoundEndReason
	Winner  common.Team
}

// RoundOfficiallyEndedEvent signals that the round 'has officially ended', not exactly sure what that is tbh.
type RoundOfficiallyEndedEvent struct{}

// RoundMVPReason is the type for the various MVPReasonYXZ constants.
//
// See RoundMVPEvent.
type RoundMVPReason byte

// RoundMVPReasons constants give information about why a player got the MVP award.
const (
	MVPReasonMostEliminations RoundMVPReason = 1
	MVPReasonBombDefused      RoundMVPReason = 2
	MVPReasonBombPlanted      RoundMVPReason = 3
)

// RoundMVPEvent signals the announcement of the last rounds MVP.
type RoundMVPEvent struct {
	Player *common.Player
	Reason RoundMVPReason
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

// PlayerFootstepEvent occurs when a player makes a footstep.
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

// NadeEventIf is the interface for all NadeEvents (except NadeProjectile* events).
// Used to catch the different events with the same handler.
type NadeEventIf interface {
	Base() NadeEvent
}

// NadeEvent contains the common attributes of nade events. Dont register
// handlers on this tho, you want NadeEventIf for that
type NadeEvent struct {
	NadeType     common.EquipmentElement
	Position     r3.Vector
	Thrower      *common.Player
	NadeEntityID int
}

// Base returns the NadeEvent itself, used for catching all events with NadeEventIf.
func (ne NadeEvent) Base() NadeEvent {
	return ne
}

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

// NadeProjectileBouncedEvent signals that a nade has just bounced off a wall/floor/ceiling or object.
type NadeProjectileBouncedEvent struct {
	Projectile *common.GrenadeProjectile
	BounceNr   int
}

// NadeProjectileThrownEvent signals that a nade has just been thrown.
// This is different from the WeaponFiredEvent because it's sent out when the projectile entity is created.
type NadeProjectileThrownEvent struct {
	Projectile *common.GrenadeProjectile
}

// NadeProjectileDestroyedEvent signals that a nade entity has been destroyed (i.e. it detonated / expired).
// This is different from the other Nade events because it's sent out when the projectile entity is destroyed.
//
// Mainly useful for getting the full trajectory of the projectile.
type NadeProjectileDestroyedEvent struct {
	Projectile *common.GrenadeProjectile
}

// PlayerFlashedEvent signals that a player was flashed.
type PlayerFlashedEvent struct {
	Player *common.Player
}

// BombEventIf is the interface for all the bomb events. Like NadeEventIf for NadeEvents.
type BombEventIf interface {
	implementsBombEventIf()
}

type bombsite rune

// Bombsite identifiers
const (
	BombsiteA bombsite = 'A'
	BombsiteB bombsite = 'B'
)

// BombEvent contains the common attributes of bomb events. Dont register
// handlers on this tho, you want BombEventIf for that.
type BombEvent struct {
	Player *common.Player
	Site   bombsite
}

// Make BombEvent implement BombEventIf
func (BombEvent) implementsBombEventIf() {}

// BombBeginPlant signals the start of a plant.
type BombBeginPlant struct {
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
	Player *common.Player
	HasKit bool
}

// BombDropEvent signals that the bomb (C4) has been dropped.
type BombDropEvent struct {
	Player   *common.Player
	EntityID int
}

// BombPickupEvent signals that the bomb (C4) has been picked up.
type BombPickupEvent struct {
	Player *common.Player
}

func (BombBeginDefuseEvent) implementsBombEventIf() {}

// HitGroup is the type for the various HitGroupXYZ constants.
//
// See PlayerHurtEvent.
type HitGroup byte

// HitGroup constants give information about where a player got hit.
// e.g. head, chest, legs etc.
const (
	HitGroupGeneric  HitGroup = 0
	HitGroupHead     HitGroup = 1
	HitGroupChest    HitGroup = 2
	HitGroupStomach  HitGroup = 3
	HitGroupLeftArm  HitGroup = 4
	HitGroupRightArm HitGroup = 5
	HitGroupLeftLeg  HitGroup = 6
	HitGroupRightLeg HitGroup = 7
	HitGroupGear     HitGroup = 10
)

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
	HitGroup     HitGroup
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
// EntIdx will probably always be 0
// See ChatMessageEvent and SayText2Event for player chat messages.
type SayTextEvent struct {
	EntIdx    int // Not sure what this is, doesn't seem to be the entity-ID
	Text      string
	IsChat    bool // Not sure, from the net-message
	IsChatAll bool // Seems to always be false, team chat might not be recorded
}

// SayText2Event signals a chat message. It just contains the raw network message.
// For player chat messages, ChatMessageEvent may be more interesting.
// Team chat is generally not recorded so IsChatAll will probably always be false.
// See SayTextEvent for admin / console messages.
type SayText2Event struct {
	EntIdx    int      // Not sure what this is, doesn't seem to be the entity-ID
	MsgName   string   // The message type, e.g. Cstrike_Chat_All for global chat
	Params    []string // The message's parameters, for Cstrike_Chat_All parameter 1 is the player and 2 the message for example
	IsChat    bool     // Not sure, from the net-message
	IsChatAll bool     // Seems to always be false, team chat might not be recorded
}

// ChatMessageEvent signals a player generated chat message.
// Since team chat is generally not recorded IsChatAll will probably always be false.
// See SayTextEvent for admin / console messages and SayText2Event for raw network package data.
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

// ItemEquipEvent signals an item was equipped.
type ItemEquipEvent struct {
	Weapon common.Equipment
	Player *common.Player
}

// ItemPickupEvent signals an item was bought or picked up.
type ItemPickupEvent struct {
	Weapon common.Equipment
	Player *common.Player
}

// ItemDropEvent signals an item was dropped.
type ItemDropEvent struct {
	Weapon common.Equipment
	Player *common.Player
}

// DataTablesParsedEvent signals that the datatables were parsed.
// You can use the Parser.SendTableParser() after this event to register update notification on entities & properties.
type DataTablesParsedEvent struct{}

// StringTableCreatedEvent signals that a string table was created via net message.
// Can be useful for figuring out when player-info is available via Parser.GameState().[Playing]Participants().
// E.g. after the table 'userinfo' has been created the player-data should be available after the next TickDoneEvent.
// The reason it's not immediately available is because we need to do some post-processing to prep that data after a tick has finished.
type StringTableCreatedEvent struct {
	TableName string
}

// ParserWarnEvent signals that a non-fatal problem occurred during parsing.
type ParserWarnEvent struct {
	Message string
}

// GenericGameEvent signals a otherwise unhandled event.
type GenericGameEvent struct {
	Name string
	Data map[string]*msg.CSVCMsg_GameEventKeyT
}

// InfernoStartedEvent signals that the fire of a incendiary or Molotov has just started.
// This is different from the FireNadeStartedEvent because it's sent out when the inferno entity is created instead of on the game-event.
type InfernoStartedEvent struct {
	Inferno *common.Inferno
}

// InfernoExpiredEvent signals that all fire from a incendiary or Molotov has extinguished.
// This is different from the FireNadeExpiredEvent event because it's sent out when the inferno entity is destroyed instead of on the game-event.
//
// Mainly useful for getting the final area of an inferno.
type InfernoExpiredEvent struct {
	Inferno *common.Inferno
}
