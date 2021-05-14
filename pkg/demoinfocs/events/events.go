// Package events contains all events that can be sent out from demoinfocs.Parser.
//
// Events are generally named in the tense that fits the best for each event.
// E.g. SmokeExpired is in the past tense because it's sent out when the smoke has completely faded away
// while SmokeStart is in the present tense because it's sent out when the smoke starts to bloom.
package events

import (
	"time"

	"github.com/golang/geo/r3"

	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	msg "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg"
)

// FrameDone signals that a demo-frame has been processed.
// A frame can contain multiple ticks (usually 2 or 4) if the tv_snapshotrate differs from the tick-rate the game was played at.
type FrameDone struct{}

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
	Message     string
	Reason      RoundEndReason
	Winner      common.Team
	WinnerState *common.TeamState // TeamState of the winner. May be nil if it's a draw (Winner == TeamSpectators)
	LoserState  *common.TeamState // TeamState of the loser. May be nil if it's a draw (Winner == TeamSpectators)
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
	Player *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
}

// PlayerTeamChange occurs when a player swaps teams.
type PlayerTeamChange struct {
	Player *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).

	// TeamState of the old team.
	// May be nil if player changed from spectators/unassigned (OldTeam == TeamSpectators || OldTeam == TeamUnassigned).
	NewTeamState *common.TeamState

	// TeamState of the old team.
	// May be nil if player changed from spectators/unassigned (OldTeam == TeamSpectators || OldTeam == TeamUnassigned).
	OldTeamState *common.TeamState

	NewTeam common.Team
	OldTeam common.Team

	Silent bool
	IsBot  bool
}

// PlayerJump signals that a player has jumped.
type PlayerJump struct {
	Player *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
}

// Kill signals that a player has been killed.
type Kill struct {
	Weapon            *common.Equipment
	Victim            *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
	Killer            *common.Player // May be nil for world damage (EqWorld) or if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
	Assister          *common.Player
	PenetratedObjects int
	IsHeadshot        bool
	AssistedFlash     bool
	AttackerBlind     bool
	NoScope           bool
	ThroughSmoke      bool
	Distance          float32
}

// IsWallBang returns true if PenetratedObjects is larger than 0.
func (k Kill) IsWallBang() bool {
	return k.PenetratedObjects > 0
}

// BotTakenOver signals that a player took over a bot.
type BotTakenOver struct {
	Taker *common.Player
}

// WeaponFire signals that a weapon has been fired.
type WeaponFire struct {
	Shooter *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
	Weapon  *common.Equipment
}

// WeaponReload signals that a player started to reload his weapon.
type WeaponReload struct {
	Player *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
}

// GrenadeEventIf is the interface for all GrenadeEvents (except GrenadeProjectile* events).
// Used to catch the different events with the same handler.
type GrenadeEventIf interface {
	Base() GrenadeEvent
}

// GrenadeEvent contains the common attributes of nade events. Dont register
// handlers on this tho, you want GrenadeEventIf for that
type GrenadeEvent struct {
	GrenadeType     common.EquipmentType
	Grenade         *common.Equipment // Maybe nil for InfernoStart & InfernoExpired since we don't know the thrower (at least in old demos)
	Position        r3.Vector
	Thrower         *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
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

// DecoyExpired signals the end of a decoy.
type DecoyExpired struct {
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
// GrenadeType will always be EqIncendiary as it's not networked whether it's an incendiary or molotov.
// Thrower will always be nil as this isn't networked.
type FireGrenadeStart struct {
	GrenadeEvent
}

// FireGrenadeExpired signals that all fires of a molly/incendiary have extinguished.
// GrenadeType will always be EqIncendiary as it's not networked whether it's an incendiary or molotov.
// Thrower will always be nil as this isn't networked.
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
	Player     *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
	Attacker   *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
	Projectile *common.GrenadeProjectile
}

// FlashDuration returns the duration of the blinding effect.
// This is just a shortcut for Player.FlashDurationTime().
func (e PlayerFlashed) FlashDuration() time.Duration {
	return e.Player.FlashDurationTime()
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

// BombPlantAborted signals the abortion of a plant.
type BombPlantAborted struct {
	Player *common.Player
}

func (BombPlantAborted) implementsBombEventIf() {}

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

func (BombDefuseStart) implementsBombEventIf() {}

// BombDefuseAborted signals that the defuser aborted the action.
type BombDefuseAborted struct {
	Player *common.Player
}

func (BombDefuseAborted) implementsBombEventIf() {}

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

// HostageRecued signals that a hostage has been rescued.
type HostageRecued struct {
	Player  *common.Player
	Hostage *common.Hostage
}

// HostageRescuedAll signals that all hostages have been rescued.
type HostageRescuedAll struct{}

// HostageHurt signals that a hostage has been hurt.
type HostageHurt struct {
	Player  *common.Player
	Hostage *common.Hostage
}

// HostageKilled signals that a hostage has been killed.
type HostageKilled struct {
	Killer  *common.Player
	Hostage *common.Hostage
}

// HostageStateChanged signals that the state of a hostage has changed.
// e.g. being untied, picked up, rescued etc.
// See HostageState for all possible values.
type HostageStateChanged struct {
	OldState common.HostageState
	NewState common.HostageState
	Hostage  *common.Hostage
}

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
	Player            *common.Player // May be nil if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
	Attacker          *common.Player // May be nil if the player is taking world damage (e.g. fall damage) or if the demo is partially corrupt (player is 'unconnected', see #156 and #172).
	Health            int
	Armor             int
	Weapon            *common.Equipment // May be EqUnknown for world-damage (falling / bomb).
	WeaponString      string            // Wrong for CZ, M4A1-S etc.
	HealthDamage      int
	ArmorDamage       int
	HealthDamageTaken int // HealthDamage excluding over-damage (e.g. if player has 5 health and is hit for 15 damage this would be 5 instead of 15)
	ArmorDamageTaken  int // ArmorDamage excluding over-damage (e.g. if player has 5 armor and is hit for 15 armor damage this would be 5 instead of 15)
	HitGroup          HitGroup
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
	SteamID32  int32 // 32-bit variant of the SteamID. See https://developer.valvesoftware.com/wiki/SteamID
	RankChange float32
	RankOld    int
	RankNew    int
	WinCount   int
	Player     *common.Player // may be nil if the player has already disconnected
}

// SteamID64 converts SteamID32 to the 64-bit SteamID variant and returns the result.
// See https://developer.valvesoftware.com/wiki/SteamID
func (ru RankUpdate) SteamID64() uint64 {
	return common.ConvertSteamID32To64(uint32(ru.SteamID32))
}

// ItemEquip signals an item was equipped.
// This event is not available in all demos.
type ItemEquip struct {
	Player *common.Player
	Weapon *common.Equipment
}

// ItemPickup signals an item was bought or picked up.
// This event is not available in all demos.
type ItemPickup struct {
	Player *common.Player
	Weapon *common.Equipment
}

// ItemDrop signals an item was dropped.
// This event is not available in all demos.
type ItemDrop struct {
	Player *common.Player
	Weapon *common.Equipment
}

// DataTablesParsed signals that the datatables were parsed.
// You can use the Parser.SendTableParser() after this event to register update notification on entities & properties.
type DataTablesParsed struct{}

// StringTableCreated signals that a string table was created via net message.
// Can be useful for figuring out when player-info is available via Parser.GameState().[Playing]Participants().
// E.g. after the table 'userinfo' has been created the player-data should be available after the next FrameDone.
// The reason it's not immediately available is because we need to do some post-processing to prep that data after a tick has finished.
type StringTableCreated struct {
	TableName string
}

// WarnType identifies a kind of warning for the ParserWarn event.
type WarnType int

const (
	WarnTypeUndefined                  = iota
	WarnTypeBombsiteUnknown            // may occur on de_grind for bombsite B as the bounding box of the bombsite is wrong
	WarnTypeTeamSwapPlayerNil          // TODO: figure out why this happens
	WarnTypeGameEventBeforeDescriptors // may occur in POV demos
)

// ParserWarn signals that a non-fatal problem occurred during parsing.
type ParserWarn struct {
	Message string
	Type    WarnType
}

// GenericGameEvent signals any game-event.
// It contains the raw data as received from the net-message.
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

// ScoreUpdated signals that the score of one of the teams has been updated.
// It has been observed that some demos do not always trigger the RoundEnd event as one would expect.
//
// ScoreUpdated can be used to circumvent missing RoundEnd events in some cases, as (we assume) it is required in order to show the correct score in-game.
type ScoreUpdated struct {
	OldScore  int
	NewScore  int
	TeamState *common.TeamState
}

// GamePhaseChanged signals that the game phase has changed. This event is for advanced usage, as more
// user friendly events are available for important game phase changes, such as TeamSideSwitch and GameHalfEnded.
type GamePhaseChanged struct {
	OldGamePhase common.GamePhase
	NewGamePhase common.GamePhase
}

// TeamSideSwitch signals that teams are switching sides, i.e. swapping between T and CT.
// TeamSideSwitch is usually dispatched in the beginning of a round, just after OnRoundStart.
type TeamSideSwitch struct {
}

// GameHalfEnded signals that the currently ongoing game half has ended.
// GameHalfEnded is usually dispatched in the end of a round, just before RoundEndOfficial.
type GameHalfEnded struct {
}

// MatchStartedChanged signals that the value of data table DT_GameRulesProxy.m_bHasMatchStarted has changed
// This can be useful for some demos where the MatchStart event is not sent.
type MatchStartedChanged struct {
	OldIsStarted bool
	NewIsStarted bool
}

// IsWarmupPeriodChanged signals that the value of data table DT_GameRulesProxy.m_bIsWarmupPeriod has changed.
// This can be useful to ignore things that happen during the warmup.
type IsWarmupPeriodChanged struct {
	OldIsWarmupPeriod bool
	NewIsWarmupPeriod bool
}

// PlayerSpottersChanged signals that a player's spotters (other players that can se him) changed.
type PlayerSpottersChanged struct {
	Spotted *common.Player
}

// ConVarsUpdated signals that ConVars/CVars have been updated.
// See GameState.ConVars().
type ConVarsUpdated struct {
	UpdatedConVars map[string]string
}

// RoundImpactScoreData contains impact assessments of events that happened during the last round.
type RoundImpactScoreData struct {
	RawMessage *msg.CCSUsrMsg_RoundImpactScoreData
}
