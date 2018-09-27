// Package events contains all events that can be sent out from demoinfocs.Parser.
package events

import (
	r3 "github.com/golang/geo/r3"

	common "github.com/markus-wa/demoinfocs-golang/common"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

// TickDone signals that a tick is done.
type TickDone struct{}

// MatchStart signals that the match has started.
type MatchStart struct{}

// RoundStart signals that a new round has started.
type RoundStart struct {
	TimeLimit int
	FragLimit int
	Objective string
}

// RoundFreezetimeEnd signals that the freeze time is over.
type RoundFreezetimeEnd struct{}

// RoundEndReason is the type for the various RoundEndReasonXYZ constants.
//
// See RoundEnd.
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

// RoundEnd signals that a round just finished.
// Attention: TeamState.Score() won't be up to date yet after this.
// Add +1 to the winner's score as a workaround.
type RoundEnd struct {
	Message string
	Reason  RoundEndReason
	Winner  common.Team
}

// RoundEndOfficial signals that the round has 'officially' ended.
// After RoundEnd and before this players are still able to walk around.
type RoundEndOfficial struct{}

// RoundMVPReason is the type for the various MVPReasonYXZ constants.
//
// See RoundMVPAnnouncement.
type RoundMVPReason byte

// RoundMVPReasons constants give information about why a player got the MVP award.
const (
	MVPReasonMostEliminations RoundMVPReason = 1
	MVPReasonBombDefused      RoundMVPReason = 2
	MVPReasonBombPlanted      RoundMVPReason = 3
)

// RoundMVPAnnouncement signals the announcement of the last rounds MVP.
type RoundMVPAnnouncement struct {
	Player *common.Player
	Reason RoundMVPReason
}

// AnnouncementMatchStarted signals that the announcement "Match Started" has been displayed.
type AnnouncementMatchStarted struct{}

// AnnouncementLastRoundHalf signals the last round of the first half.
type AnnouncementLastRoundHalf struct{}

// AnnouncementFinalRound signals the 30th round, not raised if the match ends before that.
type AnnouncementFinalRound struct{}

// AnnouncementWinPanelMatch signals that the 'win panel' has been displayed. I guess that's the final scoreboard.
type AnnouncementWinPanelMatch struct{}

// Footstep occurs when a player makes a footstep.
type Footstep struct {
	Player *common.Player
}

// PlayerTeamChange occurs when a player swaps teams.
type PlayerTeamChange struct {
	Player  *common.Player
	NewTeam common.Team
	OldTeam common.Team
	Silent  bool
	IsBot   bool
}

// PlayerJump signals that a player has jumped.
type PlayerJump struct {
	Player *common.Player
}

// Kill signals that a player has been killed.
type Kill struct {
	Weapon            *common.Equipment
	Victim            *common.Player
	Killer            *common.Player
	Assister          *common.Player
	PenetratedObjects int
	IsHeadshot        bool
}

// BotTakenOver signals that a player took over a bot.
type BotTakenOver struct {
	Taker *common.Player
}

// WeaponFire signals that a weapon has been fired.
type WeaponFire struct {
	Shooter *common.Player
	Weapon  *common.Equipment
}

// GrenadeEventIf is the interface for all GrenadeEvents (except GrenadeProjectile* events).
// Used to catch the different events with the same handler.
type GrenadeEventIf interface {
	Base() GrenadeEvent
}

// GrenadeEvent contains the common attributes of nade events. Dont register
// handlers on this tho, you want GrenadeEventIf for that
type GrenadeEvent struct {
	GrenadeType     common.EquipmentElement
	Position        r3.Vector
	Thrower         *common.Player
	GrenadeEntityID int
}

// Base returns the GrenadeEvent itself, used for catching all events with GrenadeEventIf.
func (ne GrenadeEvent) Base() GrenadeEvent {
	return ne
}

// HeExplode signals the explosion of a HE.
type HeExplode struct {
	GrenadeEvent
}

// FlashExplode signals the explosion of a Flash.
type FlashExplode struct {
	GrenadeEvent
}

// DecoyStart signals the start of a decoy.
type DecoyStart struct {
	GrenadeEvent
}

// DecoyExpire signals the end of a decoy.
type DecoyExpire struct {
	GrenadeEvent
}

// SmokeStart signals the start of a smoke (pop).
type SmokeStart struct {
	GrenadeEvent
}

// SmokeExpired signals that a smoke as completely faded away.
type SmokeExpired struct {
	GrenadeEvent
}

// FireGrenadeStart signals the start of a molly/incendiary.
type FireGrenadeStart struct {
	GrenadeEvent
}

// FireGrenadeExpired signals that all fires of a molly/incendiary have extinguished.
type FireGrenadeExpired struct {
	GrenadeEvent
}

// GrenadeProjectileBounce signals that a nade has just bounced off a wall/floor/ceiling or object.
type GrenadeProjectileBounce struct {
	Projectile *common.GrenadeProjectile
	BounceNr   int
}

// GrenadeProjectileThrow signals that a nade has just been thrown.
// This is different from the WeaponFired because it's sent out when the projectile entity is created.
type GrenadeProjectileThrow struct {
	Projectile *common.GrenadeProjectile
}

// GrenadeProjectileDestroy signals that a nade entity is being destroyed (i.e. it detonated / expired).
// This is different from the other Grenade events because it's sent out when the projectile entity is destroyed.
//
// Mainly useful for getting the full trajectory of the projectile.
type GrenadeProjectileDestroy struct {
	Projectile *common.GrenadeProjectile
}

// PlayerFlashed signals that a player was flashed.
type PlayerFlashed struct {
	Player *common.Player
}

// BombEventIf is the interface for all the bomb events. Like GrenadeEventIf for GrenadeEvents.
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

// BombPlantBegin signals the start of a plant.
type BombPlantBegin struct {
	BombEvent
}

// BombPlanted signals that the bomb has been planted.
type BombPlanted struct {
	BombEvent
}

// BombDefused signals that the bomb has been defused.
type BombDefused struct {
	BombEvent
}

// BombExplode signals that the bomb has exploded.
type BombExplode struct {
	BombEvent
}

// BombDefuseStart signals the start of defusing.
type BombDefuseStart struct {
	Player *common.Player
	HasKit bool
}

// BombDropped signals that the bomb (C4) has been dropped onto the ground.
// Not fired if it has been dropped to another player (see BombPickup for this).
type BombDropped struct {
	Player   *common.Player
	EntityID int
}

// BombPickup signals that the bomb (C4) has been picked up.
type BombPickup struct {
	Player *common.Player
}

func (BombDefuseStart) implementsBombEventIf() {}

// HitGroup is the type for the various HitGroupXYZ constants.
//
// See PlayerHurt.
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

// PlayerHurt signals that a player has been damaged.
type PlayerHurt struct {
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

// PlayerConnect signals that a player has started connecting.
type PlayerConnect struct {
	Player *common.Player
}

// PlayerDisconnected signals that a player has disconnected.
type PlayerDisconnected struct {
	Player *common.Player
}

// SayText signals a chat message. It contains the raw
// network message data for admin / console messages.
// EntIdx will probably always be 0
// See ChatMessage and SayText2 for player chat messages.
type SayText struct {
	EntIdx    int // Not sure what this is, doesn't seem to be the entity-ID
	Text      string
	IsChat    bool // Not sure, from the net-message
	IsChatAll bool // Seems to always be false, team chat might not be recorded
}

// SayText2 signals a chat message. It just contains the raw network message.
// For player chat messages, ChatMessage may be more interesting.
// Team chat is generally not recorded so IsChatAll will probably always be false.
// See SayText for admin / console messages.
type SayText2 struct {
	EntIdx    int      // Not sure what this is, doesn't seem to be the entity-ID
	MsgName   string   // The message type, e.g. Cstrike_Chat_All for global chat
	Params    []string // The message's parameters, for Cstrike_Chat_All parameter 1 is the player and 2 the message for example
	IsChat    bool     // Not sure, from the net-message
	IsChatAll bool     // Seems to always be false, team chat might not be recorded
}

// ChatMessage signals a player generated chat message.
// Since team chat is generally not recorded IsChatAll will probably always be false.
// See SayText for admin / console messages and SayText2 for raw network package data.
type ChatMessage struct {
	Sender    *common.Player
	Text      string
	IsChatAll bool
}

// RankUpdate signals the new rank. Not sure if this
// only occurs if the rank changed.
type RankUpdate struct {
	SteamID    int64
	RankOld    int
	RankNew    int
	WinCount   int
	RankChange float32
}

// ItemEquip signals an item was equipped.
type ItemEquip struct {
	Weapon common.Equipment
	Player *common.Player
}

// ItemPickup signals an item was bought or picked up.
type ItemPickup struct {
	Weapon common.Equipment
	Player *common.Player
}

// ItemDrop signals an item was dropped.
type ItemDrop struct {
	Weapon common.Equipment
	Player *common.Player
}

// DataTablesParsed signals that the datatables were parsed.
// You can use the Parser.SendTableParser() after this event to register update notification on entities & properties.
type DataTablesParsed struct{}

// StringTableCreated signals that a string table was created via net message.
// Can be useful for figuring out when player-info is available via Parser.GameState().[Playing]Participants().
// E.g. after the table 'userinfo' has been created the player-data should be available after the next TickDone.
// The reason it's not immediately available is because we need to do some post-processing to prep that data after a tick has finished.
type StringTableCreated struct {
	TableName string
}

// ParserWarn signals that a non-fatal problem occurred during parsing.
type ParserWarn struct {
	Message string
}

// GenericGameEvent signals a otherwise unhandled event.
type GenericGameEvent struct {
	Name string
	Data map[string]*msg.CSVCMsg_GameEventKeyT
}

// InfernoStart signals that the fire of a incendiary or Molotov is starting.
// This is different from the FireGrenadeStart because it's sent out when the inferno entity is created instead of on the game-event.
type InfernoStart struct {
	Inferno *common.Inferno
}

// InfernoExpired signals that all fire from a incendiary or Molotov has extinguished.
// This is different from the FireGrenadeExpire event because it's sent out when the inferno entity is destroyed instead of on the game-event.
//
// Mainly useful for getting the final area of an inferno.
type InfernoExpired struct {
	Inferno *common.Inferno
}
