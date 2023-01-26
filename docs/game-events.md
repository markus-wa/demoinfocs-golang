## Game events

List of game events that may be trigerred during parsing, some events are available only in GOTV and/or POV demos.

You can add a listener on the parser's event `GenericGameEvent` to listen to all events, example:

```go
parser.RegisterEventHandler(func(event events.GenericGameEvent) {
    fmt.Println(event.Name, event.Data)
})
```

> **Warning**
> It has been noticed that some demos may not fire events when it should. A noticable one is the `round_end` event.
> If you encounter this problem it's probably not a parser bug but simply a demo with missing events.
> As a workaround you may subscribe to properties update.
> For example to detect rounds end you could subscribe to updates of the property `m_iRoundWinStatus` of the entity `CCSGameRulesProxy`.

✅ = Available, ❌ = Not available, ? = Not sure, need to be tested

| Event name                      | GOTV | POV |
| ------------------------------- | ---- | --- |
| ammo_pickup                     | ❌   | ✅  |
| announce_phase_end              | ✅   | ✅  |
| begin_new_match                 | ✅   | ✅  |
| bomb_beep                       | ✅   | ❌  |
| bomb_begindefuse                | ✅   | ❌  |
| bomb_beginplant                 | ✅   | ❌  |
| bomb_defused                    | ✅   | ✅  |
| bomb_dropped                    | ✅   | ✅  |
| bomb_exploded                   | ✅   | ✅  |
| bomb_pickup                     | ✅   | ✅  |
| bomb_planted                    | ✅   | ✅  |
| bot_takeover                    | ✅   | ✅  |
| buytime_ended                   | ✅   | ❌  |
| choppers_incoming_warning       | ?    | ✅  |
| cs_intermission                 | ❌   | ✅  |
| cs_match_end_restart            | ✅   | ✅  |
| cs_pre_restart                  | ✅   | ✅  |
| cs_round_final_beep             | ✅   | ✅  |
| cs_round_start_beep             | ✅   | ✅  |
| cs_win_panel_match              | ✅   | ✅  |
| cs_win_panel_round              | ✅   | ✅  |
| decoy_detonate                  | ✅   | ✅  |
| decoy_started                   | ✅   | ✅  |
| endmatch_cmm_start_reveal_items | ✅   | ❌  |
| enter_bombzone                  | ❌   | ✅  |
| enter_buyzone                   | ❌   | ✅  |
| entity_visible                  | ❌   | ✅  |
| exit_bombzone                   | ❌   | ✅  |
| exit_buyzone                    | ❌   | ✅  |
| firstbombs_incoming_warning     | ?    | ✅  |
| flashbang_detonate              | ✅   | ✅  |
| hegrenade_detonate              | ✅   | ✅  |
| hltv_chase                      | ✅   | ❌  |
| hltv_fixed                      | ✅   | ?   |
| hltv_message                    | ✅   | ?   |
| hltv_status                     | ❌   | ✅  |
| hostage_follows                 | ❌   | ✅  |
| hostage_hurt                    | ✅   | ✅  |
| hostage_killed                  | ✅   | ✅  |
| hostage_rescued                 | ✅   | ✅  |
| hostage_rescued_all             | ✅   | ✅  |
| hostname_changed                | ❌   | ✅  |
| inferno_expire                  | ✅   | ✅  |
| inferno_startburn               | ✅   | ✅  |
| inspect_weapon                  | ❌   | ✅  |
| item_equip                      | ✅   | ❌  |
| item_pickup                     | ✅   | ❌  |
| item_pickup_slerp               | ❌   | ✅  |
| item_remove                     | ✅   | ❌  |
| jointeam_failed                 | ❌   | ✅  |
| other_death                     | ✅   | ✅  |
| player_blind                    | ✅   | ❌  |
| player_connect                  | ✅   | ✅  |
| player_connect_full             | ✅   | ✅  |
| player_death                    | ✅   | ✅  |
| player_disconnect               | ✅   | ✅  |
| player_falldamage               | ✅   | ❌  |
| player_footstep                 | ✅   | ❌  |
| player_given_c4                 | ❌   | ✅  |
| player_hurt                     | ✅   | ✅  |
| player_jump                     | ✅   | ❌  |
| player_changename               | ✅   | ✅  |
| player_ping                     | ❌   | ✅  |
| player_ping_stop                | ❌   | ✅  |
| player_spawn                    | ✅   | ✅  |
| player_spawned                  | ❌   | ✅  |
| player_team                     | ✅   | ✅  |
| round_announce_final            | ✅   | ✅  |
| round_announce_last_round_half  | ✅   | ✅  |
| round_announce_match_point      | ✅   | ✅  |
| round_announce_match_start      | ✅   | ✅  |
| round_announce_warmup           | ✅   | ✅  |
| round_end                       | ✅   | ✅  |
| round_end_upload_stats          | ❌   | ✅  |
| round_freeze_end                | ✅   | ✅  |
| round_mvp                       | ✅   | ✅  |
| round_poststart                 | ✅   | ❌  |
| round_prestart                  | ✅   | ❌  |
| round_officially_ended          | ✅   | ✅  |
| round_start                     | ✅   | ✅  |
| round_time_warning              | ✅   | ✅  |
| server_cvar                     | ✅   | ✅  |
| show_survival_respawn_status    | ?    | ✅  |
| smokegrenade_detonate           | ✅   | ✅  |
| smokegrenade_expired            | ✅   | ✅  |
| survival_paradrop_spawn         | ?    | ✅  |
| switch_team                     | ❌   | ✅  |
| tournament_reward               | ✅   | ?   |
| vote_cast                       | ❌   | ✅  |
| weapon_fire                     | ✅   | ✅  |
| weapon_fire_on_empty            | ✅   | ❌  |
| weapon_reload                   | ✅   | ❌  |
| weapon_zoom                     | ✅   | ❌  |
| weapon_zoom_rifle               | ❌   | ✅  |
