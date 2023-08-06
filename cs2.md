# Counter-Strike 2 (Source 2)

Here are a few notes on what data we have available for Counter-Strike 2 (Source 2).
This is a work in progress and will be updated as we learn more.

### Legacy Events

Event name + count of a Wingman (2v2) match

```
    201 hltv_chase [distance inertia ineye phi target1 target2 theta]
    127 player_death [assistedflash assister assister_pawn attacker attacker_pawn attackerblind distance dmg_armor dmg_health dominated headshot hitgroup noreplay noscope penetrated revenge thrusmoke userid userid_pawn weapon weapon_fauxitemid weapon_itemid weapon_originalowner_xuid wipe]
      9 player_disconnect [PlayerID name networkid reason userid xuid]
     10 player_team [disconnect isbot oldteam silent team userid userid_pawn]
     14 round_end [legacy message nomusic player_count reason winner]
     14 round_start [fraglimit objective timelimit]
```

### Net Messages

Message name + count of a Wingman (2v2) match

```
      1 CS_UM_EndOfMatchAllPlayersData
    227 CS_UM_RadioText
      1 CS_UM_SendPlayerItemDrops
      1 CS_UM_WarmupHasEnded
    378 CS_UM_WeaponSound
     10 CS_UM_XpUpdate
    100 EM_RemoveAllDecals
   2379 GE_FireBulletsId
  20801 GE_SosStartSoundEvent
     50 GE_SosStopSoundEvent
    375 GE_Source1LegacyGameEvent
      1 GE_Source1LegacyGameEventList
      2 net_SetConVar
      3 net_SignonState
  28704 net_Tick
      1 svc_ClassInfo
      1 svc_ClearAllStringTables
     11 svc_CreateStringTable
    155 svc_HLTVStatus
  28703 svc_PacketEntities
      1 svc_ServerInfo
     53 svc_UpdateStringTable
      1 svc_VoiceInit
      2 TE_DecalId
   1482 TE_EffectDispatchId
     61 TE_PhysicsPropId
```

### Server Classes

```
CAK47
CBarnLight
CBaseAnimGraph
CBaseButton
CBaseClientUIEntity
CBaseCombatCharacter
CBaseCSGrenade
CBaseCSGrenadeProjectile
CBaseDoor
CBaseEntity
CBaseFlex
CBaseGrenade
CBaseModelEntity
CBasePlayerController
CBasePlayerPawn
CBasePlayerWeapon
CBasePropDoor
CBaseToggle
CBaseTrigger
CBaseViewModel
CBeam
CBrBaseItem
CBRC4Target
CBreachCharge
CBreachChargeProjectile
CBreakable
CBreakableProp
CBumpMine
CBumpMineProjectile
CC4
CChicken
CColorCorrection
CColorCorrectionVolume
CCSEnvGunfire
CCSGameRulesProxy
CCSGO_TeamIntroCharacterPosition
CCSGO_TeamIntroCounterTerroristPosition
CCSGO_TeamIntroTerroristPosition
CCSGO_TeamPreviewCharacterPosition
CCSGO_TeamSelectCharacterPosition
CCSGO_TeamSelectCounterTerroristPosition
CCSGO_TeamSelectTerroristPosition
CCSGOViewModel
CCSMinimapBoundary
CCSObserverPawn
CCSPlayerController
CCSPlayerPawn
CCSPlayerPawnBase
CCSPlayerResource
CCSPropExplodingBarrel
CCSPropExplodingBarrelTop
CCSTeam
CDangerZone
CDangerZoneController
CDEagle
CDecoyGrenade
CDecoyProjectile
CDrone
CDronegun
CDynamicLight
CDynamicProp
CEconEntity
CEconWearable
CEntityDissolve
CEntityFlame
CEnvCombinedLightProbeVolume
CEnvCubemap
CEnvCubemapBox
CEnvCubemapFog
CEnvDecal
CEnvDetailController
CEnvGasCanister
CEnvLightProbeVolume
CEnvParticleGlow
CEnvProjectedTexture
CEnvScreenOverlay
CEnvSky
CEnvVolumetricFogController
CEnvVolumetricFogVolume
CEnvWind
CFireCrackerBlast
CFireSmoke
CFish
CFists
CFlashbang
CFogController
CFootstepControl
CFuncBrush
CFuncConveyor
CFuncElectrifiedVolume
CFuncLadder
CFuncMonitor
CFuncMoveLinear
CFuncRotating
CFuncTrackTrain
CFuncWater
CGameRulesProxy
CGradientFog
CGrassBurn
CHandleTest
CHEGrenade
CHostage
CHostageCarriableProp
CIncendiaryGrenade
CInferno
CInfoInstructorHintHostageRescueZone
CInfoLadderDismount
CInfoMapRegion
CInfoOffscreenPanoramaTexture
CInfoVisibilityBox
CInfoWorldLayer
CItemCash
CItemDogtags
CItem_Healthshot
CKnife
CKnifeGG
CLightDirectionalEntity
CLightEntity
CLightEnvironmentEntity
CLightGlow
CLightOrthoEntity
CLightSpotEntity
CMapVetoPickController
CMelee
CModelPointEntity
CMolotovGrenade
CMolotovProjectile
COmniLight
CParadropChopper
CParticleSystem
CPathParticleRope
CPhysBox
CPhysicsProp
CPhysicsPropMultiplayer
CPhysMagnet
CPhysPropAmmoBox
CPhysPropLootCrate
CPhysPropRadarJammer
CPhysPropWeaponUpgrade
CPlantedC4
CPlantedC4Survival
CPlayerPing
CPlayerSprayDecal
CPlayerVisibility
CPointCamera
CPointClientUIDialog
CPointClientUIWorldPanel
CPointClientUIWorldTextPanel
CPointCommentaryNode
CPointEntity
CPointValueRemapper
CPointWorldText
CPostProcessingVolume
CPrecipitation
CPrecipitationBlocker
CPredictedViewModel
CPropCounter
CRagdollManager
CRagdollProp
CRagdollPropAttached
CRectLight
CRopeKeyframe
CSceneEntity
CSensorGrenade
CSensorGrenadeProjectile
CShatterGlassShardPhysics
CSkyCamera
CSmokeGrenade
CSmokeGrenadeProjectile
CSoundAreaEntityBase
CSoundAreaEntityOrientedBox
CSoundAreaEntitySphere
CSoundOpvarSetAABBEntity
CSoundOpvarSetOBBEntity
CSoundOpvarSetOBBWindEntity
CSoundOpvarSetPathCornerEntity
CSoundOpvarSetPointBase
CSoundOpvarSetPointEntity
CSpotlightEnd
CSprite
CSpriteOriented
CSun
CSurvivalSpawnChopper
CTablet
CTeam
CTextureBasedAnimatable
CTonemapController2
CTriggerBuoyancy
CTriggerVolume
CTripWireFire
CTripWireFireProjectile
CVoteController
CWaterBullet
CWeaponAug
CWeaponAWP
CWeaponBaseItem
CWeaponBizon
CWeaponCSBase
CWeaponCSBaseGun
CWeaponElite
CWeaponFamas
CWeaponFiveSeven
CWeaponG3SG1
CWeaponGalilAR
CWeaponGlock
CWeaponHKP2000
CWeaponM249
CWeaponM4A1
CWeaponMAC10
CWeaponMag7
CWeaponMP7
CWeaponMP9
CWeaponNegev
CWeaponNOVA
CWeaponP250
CWeaponP90
CWeaponSawedoff
CWeaponSCAR20
CWeaponSG556
CWeaponShield
CWeaponSSG08
CWeaponTaser
CWeaponTec9
CWeaponUMP45
CWeaponXM1014
CWeaponZoneRepulsor
CWorld
```