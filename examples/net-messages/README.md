# Parsing & handling custom net-messages

## Finding interesting messages

You can use the build tag `debugdemoinfocs` to find interesting net-messages.

Example: `go run myprogram.go -tags debugdemoinfocs | grep "UnhandledMessage" | sort | uniq -c`

<details>
<summary>Sample output</summary>

```
      1 UnhandledMessage: id=10 name=svc_ClassInfo
      1 UnhandledMessage: id=14 name=svc_VoiceInit
   9651 UnhandledMessage: id=17 name=svc_Sounds
      1 UnhandledMessage: id=18 name=svc_SetView
    227 UnhandledMessage: id=21 name=svc_BSPDecal
  12705 UnhandledMessage: id=27 name=svc_TempEntities
    514 UnhandledMessage: id=28 name=svc_Prefetch
  85308 UnhandledMessage: id=4 name=net_Tick
      2 UnhandledMessage: id=5 name=net_StringCmd
      3 UnhandledMessage: id=6 name=net_SetConVar
      3 UnhandledMessage: id=7 name=net_SignonState
      1 UnhandledMessage: id=8 name=svc_ServerInfo
```
</details>

## Configuring a `NetMessageCreator`

NetMessageCreators are needed for creating instances of net-messages that aren't parsed by default.

You need to add them to the `ParserConfig.AdditionalNetMessageCreators` map where the key is the message-ID as seen in the debug output.

Example: `ConVar` messages

```go
import (
	dem "github.com/markus-wa/demoinfocs-golang"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

cfg := dem.DefaultParserConfig
cfg.AdditionalNetMessageCreators = map[int]dem.NetMessageCreator{
	6: func() proto.Message {
		return new(msg.CNETMsg_SetConVar)
	},
}
```

## Registering net-message handlers

To register a handler for net-messages `Parser.RegisterNetMessageHandler()` can be used.

When using `Parser.ParseToEnd()` net-messages and events are dispatched asynchronously. To get around this you can use `Parser.ParseNextFrame()` instead.

Example:

```go
p.RegisterNetMessageHandler(func(m *msg.CNETMsg_SetConVar) {
	for _, cvar := range m.Convars.Cvars {
		fmt.Println(fmt.Sprintf("cvar %s=%s", cvar.Name, cvar.Value))
	}
})
```

<details>
<summary>Sample output</summary>

```
cvar mp_spec_swapplayersides=1
cvar cash_team_rescued_hostage=750
cvar bot_autodifficulty_threshold_high=0
cvar cash_team_win_by_defusing_bomb=3500
cvar game_mode=1
cvar sv_damage_print_enable=0
cvar mp_force_pick_time=160
cvar mp_ggtr_bomb_pts_for_upgrade=2
cvar bot_quota=0
cvar ff_damage_reduction_bullets=0.33
cvar sv_gameinstructor_disable=1
cvar =0
cvar bot_quota_mode=fill
cvar mp_join_grace_time=30
cvar mp_maxrounds=30
cvar ammo_grenade_limit_total=4
cvar mp_spectators_max=10
cvar mp_round_restart_delay=5
cvar mp_win_panel_display_time=15
cvar mp_respawn_immunitytime=0
cvar mp_roundtime_defuse=1.92
cvar mp_ggprogressive_round_restart_delay=15
cvar mp_match_end_restart=1
cvar mp_timelimit=0
cvar mp_warmuptime=5
cvar mp_randomspawn_los=0
cvar sv_competitive_official_5v5=1
cvar sv_mincmdrate=30
cvar mp_halftime_duration=15
cvar mp_weapons_allow_map_placed=1
cvar mp_autokick=0
cvar sv_grenade_trajectory_time_spectator=1
cvar sv_minrate=20000
cvar sv_holiday_mode=0
cvar sv_kick_players_with_cooldown=0
cvar mp_ggtr_bomb_defuse_bonus=1
cvar spec_freeze_time=5
cvar mp_buytime=15
cvar bot_difficulty=2
cvar mp_playerid_delay=0.5
cvar mp_roundtime_hostage=1.92
cvar mp_freezetime=15
cvar sv_maxcmdrate=128
cvar bot_defer_to_human_goals=1
cvar sv_skyname=sky_cs15_daylight01_hdr
cvar mp_friendlyfire=1
cvar cash_team_hostage_interaction=150
cvar spec_freeze_panel_extended_time=0
cvar ff_damage_reduction_other=0.4
cvar sv_allow_wait_command=0
cvar mp_molotovusedelay=0
cvar mp_playerid_hold=0.25
cvar mp_limitteams=0
cvar cash_team_elimination_hostage_map_t=3000
cvar sv_friction=4.8
cvar mp_ggtr_bomb_detonation_bonus=1
cvar ammo_grenade_limit_flashbang=2
cvar sv_coaching_enabled=1
cvar steamworks_sessionid_server=1169494819006
cvar mp_overtime_enable=1
cvar tv_snapshotrate=24
cvar mp_roundtime=1.92
cvar sv_kick_ban_duration=0
cvar mp_halftime=1
cvar sv_spawn_afk_bomb_drop_time=30
cvar mp_ggtr_bomb_respawn_delay=0
cvar mp_overtime_startmoney=16000
cvar think_limit=0
cvar sv_accelerate=5.6
cvar ff_damage_reduction_grenade=0.85
cvar cash_team_elimination_hostage_map_ct=3000
cvar sv_maxupdaterate=128
cvar cash_team_hostage_alive=150
cvar tv_transmitall=1
cvar steamworks_sessionid_server=0
cvar steamworks_sessionid_server=1169497558498
```
</details>
