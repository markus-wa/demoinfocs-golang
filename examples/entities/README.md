# Using unhandled entity-data

This example shows how to use unhandled data of entities by registering entity-creation and property-update handlers on server-classes & entities respectively.

## Finding interesting server-classes & entity-properties

You can use the build tag `debugdemoinfocs` and the set `debugServerClasses=YES` with ldflags to find interesting server-classes and their properties.

Example: `go run myprogram.go -tags debugdemoinfocs -ldflags '-X github.com/markus-wa/demoinfocs-golang.debugServerClasses=YES' | grep ServerClass`

This gives you a list of all server-classes from any demo that was parsed in `myprogram.go`.

<details>
<summary>Sample output</summary>

```
ServerClass: id=0 name=CAI_BaseNPC
ServerClass: id=1 name=CAK47
ServerClass: id=2 name=CBaseAnimating
ServerClass: id=3 name=CBaseAnimatingOverlay
ServerClass: id=4 name=CBaseAttributableItem
ServerClass: id=5 name=CBaseButton
ServerClass: id=6 name=CBaseCombatCharacter
ServerClass: id=7 name=CBaseCombatWeapon
ServerClass: id=8 name=CBaseCSGrenade
ServerClass: id=9 name=CBaseCSGrenadeProjectile
ServerClass: id=10 name=CBaseDoor
ServerClass: id=11 name=CBaseEntity
ServerClass: id=12 name=CBaseFlex
ServerClass: id=13 name=CBaseGrenade
ServerClass: id=14 name=CBaseParticleEntity
ServerClass: id=15 name=CBasePlayer
ServerClass: id=16 name=CBasePropDoor
ServerClass: id=17 name=CBaseTeamObjectiveResource
ServerClass: id=18 name=CBaseTempEntity
ServerClass: id=19 name=CBaseToggle
ServerClass: id=20 name=CBaseTrigger
ServerClass: id=21 name=CBaseViewModel
ServerClass: id=22 name=CBaseVPhysicsTrigger
ServerClass: id=23 name=CBaseWeaponWorldModel
ServerClass: id=24 name=CBeam
ServerClass: id=25 name=CBeamSpotlight
ServerClass: id=26 name=CBoneFollower
ServerClass: id=27 name=CBreakableProp
ServerClass: id=28 name=CBreakableSurface
ServerClass: id=29 name=CC4
ServerClass: id=30 name=CCascadeLight
ServerClass: id=31 name=CChicken
ServerClass: id=32 name=CColorCorrection
ServerClass: id=33 name=CColorCorrectionVolume
ServerClass: id=34 name=CCSGameRulesProxy
ServerClass: id=35 name=CCSPlayer
ServerClass: id=36 name=CCSPlayerResource
ServerClass: id=37 name=CCSRagdoll
ServerClass: id=38 name=CCSTeam
ServerClass: id=39 name=CDEagle
ServerClass: id=40 name=CDecoyGrenade
ServerClass: id=41 name=CDecoyProjectile
ServerClass: id=42 name=CDynamicLight
ServerClass: id=43 name=CDynamicProp
ServerClass: id=44 name=CEconEntity
ServerClass: id=45 name=CEmbers
ServerClass: id=46 name=CEntityDissolve
ServerClass: id=47 name=CEntityFlame
ServerClass: id=48 name=CEntityFreezing
ServerClass: id=49 name=CEntityParticleTrail
ServerClass: id=50 name=CEnvAmbientLight
ServerClass: id=51 name=CEnvDetailController
ServerClass: id=52 name=CEnvDOFController
ServerClass: id=53 name=CEnvParticleScript
ServerClass: id=54 name=CEnvProjectedTexture
ServerClass: id=55 name=CEnvQuadraticBeam
ServerClass: id=56 name=CEnvScreenEffect
ServerClass: id=57 name=CEnvScreenOverlay
ServerClass: id=58 name=CEnvTonemapController
ServerClass: id=59 name=CEnvWind
ServerClass: id=60 name=CFEPlayerDecal
ServerClass: id=61 name=CFireCrackerBlast
ServerClass: id=62 name=CFireSmoke
ServerClass: id=63 name=CFireTrail
ServerClass: id=64 name=CFish
ServerClass: id=65 name=CFlashbang
ServerClass: id=66 name=CFogController
ServerClass: id=67 name=CFootstepControl
ServerClass: id=68 name=CFunc_Dust
ServerClass: id=69 name=CFunc_LOD
ServerClass: id=70 name=CFuncAreaPortalWindow
ServerClass: id=71 name=CFuncBrush
ServerClass: id=72 name=CFuncConveyor
ServerClass: id=73 name=CFuncLadder
ServerClass: id=74 name=CFuncMonitor
ServerClass: id=75 name=CFuncMoveLinear
ServerClass: id=76 name=CFuncOccluder
ServerClass: id=77 name=CFuncReflectiveGlass
ServerClass: id=78 name=CFuncRotating
ServerClass: id=79 name=CFuncSmokeVolume
ServerClass: id=80 name=CFuncTrackTrain
ServerClass: id=81 name=CGameRulesProxy
ServerClass: id=82 name=CHandleTest
ServerClass: id=83 name=CHEGrenade
ServerClass: id=84 name=CHostage
ServerClass: id=85 name=CHostageCarriableProp
ServerClass: id=86 name=CIncendiaryGrenade
ServerClass: id=87 name=CInferno
ServerClass: id=88 name=CInfoLadderDismount
ServerClass: id=89 name=CInfoOverlayAccessor
ServerClass: id=90 name=CItem_Healthshot
ServerClass: id=91 name=CKnife
ServerClass: id=92 name=CKnifeGG
ServerClass: id=93 name=CLightGlow
ServerClass: id=94 name=CMaterialModifyControl
ServerClass: id=95 name=CMolotovGrenade
ServerClass: id=96 name=CMolotovProjectile
ServerClass: id=97 name=CMovieDisplay
ServerClass: id=98 name=CParticleFire
ServerClass: id=99 name=CParticlePerformanceMonitor
ServerClass: id=100 name=CParticleSystem
ServerClass: id=101 name=CPhysBox
ServerClass: id=102 name=CPhysBoxMultiplayer
ServerClass: id=103 name=CPhysicsProp
ServerClass: id=104 name=CPhysicsPropMultiplayer
ServerClass: id=105 name=CPhysMagnet
ServerClass: id=106 name=CPlantedC4
ServerClass: id=107 name=CPlasma
ServerClass: id=108 name=CPlayerResource
ServerClass: id=109 name=CPointCamera
ServerClass: id=110 name=CPointCommentaryNode
ServerClass: id=111 name=CPoseController
ServerClass: id=112 name=CPostProcessController
ServerClass: id=113 name=CPrecipitation
ServerClass: id=114 name=CPrecipitationBlocker
ServerClass: id=115 name=CPredictedViewModel
ServerClass: id=116 name=CProp_Hallucination
ServerClass: id=117 name=CPropDoorRotating
ServerClass: id=118 name=CPropJeep
ServerClass: id=119 name=CPropVehicleDriveable
ServerClass: id=120 name=CRagdollManager
ServerClass: id=121 name=CRagdollProp
ServerClass: id=122 name=CRagdollPropAttached
ServerClass: id=123 name=CRopeKeyframe
ServerClass: id=124 name=CSCAR17
ServerClass: id=125 name=CSceneEntity
ServerClass: id=126 name=CSensorGrenade
ServerClass: id=127 name=CSensorGrenadeProjectile
ServerClass: id=128 name=CShadowControl
ServerClass: id=129 name=CSlideshowDisplay
ServerClass: id=130 name=CSmokeGrenade
ServerClass: id=131 name=CSmokeGrenadeProjectile
ServerClass: id=132 name=CSmokeStack
ServerClass: id=133 name=CSpatialEntity
ServerClass: id=134 name=CSpotlightEnd
ServerClass: id=135 name=CSprite
ServerClass: id=136 name=CSpriteOriented
ServerClass: id=137 name=CSpriteTrail
ServerClass: id=138 name=CStatueProp
ServerClass: id=139 name=CSteamJet
ServerClass: id=140 name=CSun
ServerClass: id=141 name=CSunlightShadowControl
ServerClass: id=142 name=CTeam
ServerClass: id=143 name=CTeamplayRoundBasedRulesProxy
ServerClass: id=144 name=CTEArmorRicochet
ServerClass: id=145 name=CTEBaseBeam
ServerClass: id=146 name=CTEBeamEntPoint
ServerClass: id=147 name=CTEBeamEnts
ServerClass: id=148 name=CTEBeamFollow
ServerClass: id=149 name=CTEBeamLaser
ServerClass: id=150 name=CTEBeamPoints
ServerClass: id=151 name=CTEBeamRing
ServerClass: id=152 name=CTEBeamRingPoint
ServerClass: id=153 name=CTEBeamSpline
ServerClass: id=154 name=CTEBloodSprite
ServerClass: id=155 name=CTEBloodStream
ServerClass: id=156 name=CTEBreakModel
ServerClass: id=157 name=CTEBSPDecal
ServerClass: id=158 name=CTEBubbles
ServerClass: id=159 name=CTEBubbleTrail
ServerClass: id=160 name=CTEClientProjectile
ServerClass: id=161 name=CTEDecal
ServerClass: id=162 name=CTEDust
ServerClass: id=163 name=CTEDynamicLight
ServerClass: id=164 name=CTEEffectDispatch
ServerClass: id=165 name=CTEEnergySplash
ServerClass: id=166 name=CTEExplosion
ServerClass: id=167 name=CTEFireBullets
ServerClass: id=168 name=CTEFizz
ServerClass: id=169 name=CTEFootprintDecal
ServerClass: id=170 name=CTEFoundryHelpers
ServerClass: id=171 name=CTEGaussExplosion
ServerClass: id=172 name=CTEGlowSprite
ServerClass: id=173 name=CTEImpact
ServerClass: id=174 name=CTEKillPlayerAttachments
ServerClass: id=175 name=CTELargeFunnel
ServerClass: id=176 name=CTEMetalSparks
ServerClass: id=177 name=CTEMuzzleFlash
ServerClass: id=178 name=CTEParticleSystem
ServerClass: id=179 name=CTEPhysicsProp
ServerClass: id=180 name=CTEPlantBomb
ServerClass: id=181 name=CTEPlayerAnimEvent
ServerClass: id=182 name=CTEPlayerDecal
ServerClass: id=183 name=CTEProjectedDecal
ServerClass: id=184 name=CTERadioIcon
ServerClass: id=185 name=CTEShatterSurface
ServerClass: id=186 name=CTEShowLine
ServerClass: id=187 name=CTesla
ServerClass: id=188 name=CTESmoke
ServerClass: id=189 name=CTESparks
ServerClass: id=190 name=CTESprite
ServerClass: id=191 name=CTESpriteSpray
ServerClass: id=192 name=CTest_ProxyToggle_Networkable
ServerClass: id=193 name=CTestTraceline
ServerClass: id=194 name=CTEWorldDecal
ServerClass: id=195 name=CTriggerPlayerMovement
ServerClass: id=196 name=CTriggerSoundOperator
ServerClass: id=197 name=CVGuiScreen
ServerClass: id=198 name=CVoteController
ServerClass: id=199 name=CWaterBullet
ServerClass: id=200 name=CWaterLODControl
ServerClass: id=201 name=CWeaponAug
ServerClass: id=202 name=CWeaponAWP
ServerClass: id=203 name=CWeaponBaseItem
ServerClass: id=204 name=CWeaponBizon
ServerClass: id=205 name=CWeaponCSBase
ServerClass: id=206 name=CWeaponCSBaseGun
ServerClass: id=207 name=CWeaponCycler
ServerClass: id=208 name=CWeaponElite
ServerClass: id=209 name=CWeaponFamas
ServerClass: id=210 name=CWeaponFiveSeven
ServerClass: id=211 name=CWeaponG3SG1
ServerClass: id=212 name=CWeaponGalil
ServerClass: id=213 name=CWeaponGalilAR
ServerClass: id=214 name=CWeaponGlock
ServerClass: id=215 name=CWeaponHKP2000
ServerClass: id=216 name=CWeaponM249
ServerClass: id=217 name=CWeaponM3
ServerClass: id=218 name=CWeaponM4A1
ServerClass: id=219 name=CWeaponMAC10
ServerClass: id=220 name=CWeaponMag7
ServerClass: id=221 name=CWeaponMP5Navy
ServerClass: id=222 name=CWeaponMP7
ServerClass: id=223 name=CWeaponMP9
ServerClass: id=224 name=CWeaponNegev
ServerClass: id=225 name=CWeaponNOVA
ServerClass: id=226 name=CWeaponP228
ServerClass: id=227 name=CWeaponP250
ServerClass: id=228 name=CWeaponP90
ServerClass: id=229 name=CWeaponSawedoff
ServerClass: id=230 name=CWeaponSCAR20
ServerClass: id=231 name=CWeaponScout
ServerClass: id=232 name=CWeaponSG550
ServerClass: id=233 name=CWeaponSG552
ServerClass: id=234 name=CWeaponSG556
ServerClass: id=235 name=CWeaponSSG08
ServerClass: id=236 name=CWeaponTaser
ServerClass: id=237 name=CWeaponTec9
ServerClass: id=238 name=CWeaponTMP
ServerClass: id=239 name=CWeaponUMP45
ServerClass: id=240 name=CWeaponUSP
ServerClass: id=241 name=CWeaponXM1014
ServerClass: id=242 name=CWorld
ServerClass: id=243 name=DustTrail
ServerClass: id=244 name=MovieExplosion
ServerClass: id=245 name=ParticleSmokeGrenade
ServerClass: id=246 name=RocketTrail
ServerClass: id=247 name=SmokeTrail
ServerClass: id=248 name=SporeExplosion
ServerClass: id=249 name=SporeTrail
```
</details>

If you remove the `grep ServerClass` it will also print all properties that this server-class has.

<details>
<summary>Sample ServerClass with properties</summary>

```
ServerClass: id=202 name=CWeaponAWP
	dataTableId=202
	dataTableName=DT_WeaponAWP
	baseClasses:
		CBaseEntity
		CBaseAnimating
		CBaseAnimatingOverlay
		CBaseFlex
		CEconEntity
		CBaseCombatWeapon
		CWeaponCSBase
		CWeaponCSBaseGun
	props:
		m_flSimulationTime
		m_cellbits
		m_cellX
		m_cellY
		m_cellZ
		serveranimdata.m_flCycle
		m_vecOrigin
		m_angRotation
		m_fAccuracyPenalty
		m_bSpottedByMask.001
		m_flPoseParameter.000
		m_flPoseParameter.001
		m_flPoseParameter.002
		m_flPoseParameter.003
		m_flPoseParameter.004
		m_flPoseParameter.005
		m_flPoseParameter.006
		m_flPoseParameter.007
		m_flPoseParameter.008
		m_flPoseParameter.009
		m_flPoseParameter.010
		m_flPoseParameter.011
		m_flPoseParameter.012
		m_flPoseParameter.013
		m_flPoseParameter.014
		m_flPoseParameter.015
		m_flPoseParameter.016
		m_flPoseParameter.017
		m_flPoseParameter.018
		m_flPoseParameter.019
		m_flPoseParameter.020
		m_flPoseParameter.021
		m_flPoseParameter.022
		m_flPoseParameter.023
		m_flEncodedController.000
		m_flEncodedController.001
		m_flEncodedController.002
		m_flEncodedController.003
		m_Collision.m_triggerBloat
		m_AnimOverlay.lengthproxy.lengthprop15
		m_AnimOverlay.000.m_nSequence
		m_AnimOverlay.000.m_flCycle
		m_AnimOverlay.000.m_flPlaybackRate
		m_AnimOverlay.000.m_flPrevCycle
		m_AnimOverlay.000.m_flWeight
		m_AnimOverlay.000.m_flWeightDeltaRate
		m_AnimOverlay.000.m_nOrder
		m_AnimOverlay.001.m_nSequence
		m_AnimOverlay.001.m_flCycle
		m_AnimOverlay.001.m_flPlaybackRate
		m_AnimOverlay.001.m_flPrevCycle
		m_AnimOverlay.001.m_flWeight
		m_AnimOverlay.001.m_flWeightDeltaRate
		m_AnimOverlay.001.m_nOrder
		m_AnimOverlay.002.m_nSequence
		m_AnimOverlay.002.m_flCycle
		m_AnimOverlay.002.m_flPlaybackRate
		m_AnimOverlay.002.m_flPrevCycle
		m_AnimOverlay.002.m_flWeight
		m_AnimOverlay.002.m_flWeightDeltaRate
		m_AnimOverlay.002.m_nOrder
		m_AnimOverlay.003.m_nSequence
		m_AnimOverlay.003.m_flCycle
		m_AnimOverlay.003.m_flPlaybackRate
		m_AnimOverlay.003.m_flPrevCycle
		m_AnimOverlay.003.m_flWeight
		m_AnimOverlay.003.m_flWeightDeltaRate
		m_AnimOverlay.003.m_nOrder
		m_AnimOverlay.004.m_nSequence
		m_AnimOverlay.004.m_flCycle
		m_AnimOverlay.004.m_flPlaybackRate
		m_AnimOverlay.004.m_flPrevCycle
		m_AnimOverlay.004.m_flWeight
		m_AnimOverlay.004.m_flWeightDeltaRate
		m_AnimOverlay.004.m_nOrder
		m_AnimOverlay.005.m_nSequence
		m_AnimOverlay.005.m_flCycle
		m_AnimOverlay.005.m_flPlaybackRate
		m_AnimOverlay.005.m_flPrevCycle
		m_AnimOverlay.005.m_flWeight
		m_AnimOverlay.005.m_flWeightDeltaRate
		m_AnimOverlay.005.m_nOrder
		m_AnimOverlay.006.m_nSequence
		m_AnimOverlay.006.m_flCycle
		m_AnimOverlay.006.m_flPlaybackRate
		m_AnimOverlay.006.m_flPrevCycle
		m_AnimOverlay.006.m_flWeight
		m_AnimOverlay.006.m_flWeightDeltaRate
		m_AnimOverlay.006.m_nOrder
		m_AnimOverlay.007.m_nSequence
		m_AnimOverlay.007.m_flCycle
		m_AnimOverlay.007.m_flPlaybackRate
		m_AnimOverlay.007.m_flPrevCycle
		m_AnimOverlay.007.m_flWeight
		m_AnimOverlay.007.m_flWeightDeltaRate
		m_AnimOverlay.007.m_nOrder
		m_AnimOverlay.008.m_nSequence
		m_AnimOverlay.008.m_flCycle
		m_AnimOverlay.008.m_flPlaybackRate
		m_AnimOverlay.008.m_flPrevCycle
		m_AnimOverlay.008.m_flWeight
		m_AnimOverlay.008.m_flWeightDeltaRate
		m_AnimOverlay.008.m_nOrder
		m_AnimOverlay.009.m_nSequence
		m_AnimOverlay.009.m_flCycle
		m_AnimOverlay.009.m_flPlaybackRate
		m_AnimOverlay.009.m_flPrevCycle
		m_AnimOverlay.009.m_flWeight
		m_AnimOverlay.009.m_flWeightDeltaRate
		m_AnimOverlay.009.m_nOrder
		m_AnimOverlay.010.m_nSequence
		m_AnimOverlay.010.m_flCycle
		m_AnimOverlay.010.m_flPlaybackRate
		m_AnimOverlay.010.m_flPrevCycle
		m_AnimOverlay.010.m_flWeight
		m_AnimOverlay.010.m_flWeightDeltaRate
		m_AnimOverlay.010.m_nOrder
		m_AnimOverlay.011.m_nSequence
		m_AnimOverlay.011.m_flCycle
		m_AnimOverlay.011.m_flPlaybackRate
		m_AnimOverlay.011.m_flPrevCycle
		m_AnimOverlay.011.m_flWeight
		m_AnimOverlay.011.m_flWeightDeltaRate
		m_AnimOverlay.011.m_nOrder
		m_AnimOverlay.012.m_nSequence
		m_AnimOverlay.012.m_flCycle
		m_AnimOverlay.012.m_flPlaybackRate
		m_AnimOverlay.012.m_flPrevCycle
		m_AnimOverlay.012.m_flWeight
		m_AnimOverlay.012.m_flWeightDeltaRate
		m_AnimOverlay.012.m_nOrder
		m_AnimOverlay.013.m_nSequence
		m_AnimOverlay.013.m_flCycle
		m_AnimOverlay.013.m_flPlaybackRate
		m_AnimOverlay.013.m_flPrevCycle
		m_AnimOverlay.013.m_flWeight
		m_AnimOverlay.013.m_flWeightDeltaRate
		m_AnimOverlay.013.m_nOrder
		m_AnimOverlay.014.m_nSequence
		m_AnimOverlay.014.m_flCycle
		m_AnimOverlay.014.m_flPlaybackRate
		m_AnimOverlay.014.m_flPrevCycle
		m_AnimOverlay.014.m_flWeight
		m_AnimOverlay.014.m_flWeightDeltaRate
		m_AnimOverlay.014.m_nOrder
		m_flexWeight.000
		m_flexWeight.001
		m_flexWeight.002
		m_flexWeight.003
		m_flexWeight.004
		m_flexWeight.005
		m_flexWeight.006
		m_flexWeight.007
		m_flexWeight.008
		m_flexWeight.009
		m_flexWeight.010
		m_flexWeight.011
		m_flexWeight.012
		m_flexWeight.013
		m_flexWeight.014
		m_flexWeight.015
		m_flexWeight.016
		m_flexWeight.017
		m_flexWeight.018
		m_flexWeight.019
		m_flexWeight.020
		m_flexWeight.021
		m_flexWeight.022
		m_flexWeight.023
		m_flexWeight.024
		m_flexWeight.025
		m_flexWeight.026
		m_flexWeight.027
		m_flexWeight.028
		m_flexWeight.029
		m_flexWeight.030
		m_flexWeight.031
		m_flexWeight.032
		m_flexWeight.033
		m_flexWeight.034
		m_flexWeight.035
		m_flexWeight.036
		m_flexWeight.037
		m_flexWeight.038
		m_flexWeight.039
		m_flexWeight.040
		m_flexWeight.041
		m_flexWeight.042
		m_flexWeight.043
		m_flexWeight.044
		m_flexWeight.045
		m_flexWeight.046
		m_flexWeight.047
		m_flexWeight.048
		m_flexWeight.049
		m_flexWeight.050
		m_flexWeight.051
		m_flexWeight.052
		m_flexWeight.053
		m_flexWeight.054
		m_flexWeight.055
		m_flexWeight.056
		m_flexWeight.057
		m_flexWeight.058
		m_flexWeight.059
		m_flexWeight.060
		m_flexWeight.061
		m_flexWeight.062
		m_flexWeight.063
		m_flexWeight.064
		m_flexWeight.065
		m_flexWeight.066
		m_flexWeight.067
		m_flexWeight.068
		m_flexWeight.069
		m_flexWeight.070
		m_flexWeight.071
		m_flexWeight.072
		m_flexWeight.073
		m_flexWeight.074
		m_flexWeight.075
		m_flexWeight.076
		m_flexWeight.077
		m_flexWeight.078
		m_flexWeight.079
		m_flexWeight.080
		m_flexWeight.081
		m_flexWeight.082
		m_flexWeight.083
		m_flexWeight.084
		m_flexWeight.085
		m_flexWeight.086
		m_flexWeight.087
		m_flexWeight.088
		m_flexWeight.089
		m_flexWeight.090
		m_flexWeight.091
		m_flexWeight.092
		m_flexWeight.093
		m_flexWeight.094
		m_flexWeight.095
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.lengthproxy.lengthprop32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.000.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.000.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.000.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.000.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.000.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.001.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.001.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.001.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.001.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.001.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.002.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.002.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.002.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.002.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.002.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.003.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.003.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.003.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.003.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.003.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.004.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.004.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.004.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.004.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.004.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.005.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.005.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.005.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.005.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.005.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.006.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.006.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.006.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.006.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.006.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.007.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.007.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.007.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.007.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.007.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.008.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.008.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.008.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.008.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.008.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.009.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.009.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.009.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.009.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.009.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.010.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.010.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.010.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.010.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.010.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.011.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.011.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.011.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.011.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.011.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.012.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.012.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.012.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.012.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.012.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.013.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.013.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.013.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.013.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.013.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.014.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.014.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.014.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.014.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.014.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.015.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.015.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.015.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.015.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.015.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.016.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.016.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.016.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.016.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.016.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.017.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.017.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.017.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.017.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.017.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.018.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.018.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.018.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.018.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.018.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.019.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.019.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.019.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.019.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.019.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.020.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.020.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.020.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.020.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.020.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.021.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.021.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.021.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.021.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.021.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.022.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.022.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.022.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.022.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.022.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.023.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.023.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.023.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.023.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.023.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.024.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.024.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.024.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.024.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.024.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.025.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.025.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.025.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.025.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.025.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.026.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.026.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.026.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.026.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.026.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.027.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.027.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.027.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.027.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.027.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.028.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.028.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.028.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.028.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.028.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.029.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.029.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.029.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.029.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.029.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.030.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.030.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.030.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.030.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.030.m_bSetBonus
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.031.m_iAttributeDefinitionIndex
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.031.m_iRawValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.031.m_iRawInitialValue32
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.031.m_nRefundableCurrency
		m_AttributeManager.m_Item.m_NetworkedDynamicAttributesForDemos.m_Attributes.031.m_bSetBonus
		m_AttributeManager.m_Item.m_iItemDefinitionIndex
		m_AttributeManager.m_Item.m_iEntityLevel
		m_AttributeManager.m_Item.m_iItemIDHigh
		m_AttributeManager.m_Item.m_iItemIDLow
		m_AttributeManager.m_Item.m_iAccountID
		m_AttributeManager.m_Item.m_iEntityQuality
		m_AttributeManager.m_Item.m_bInitialized
		m_AttributeManager.m_Item.m_szCustomName
		m_AttributeManager.m_hOuter
		m_AttributeManager.m_ProviderType
		m_AttributeManager.m_iReapplyProvisionParity
		LocalWeaponData.m_iPrimaryAmmoType
		LocalWeaponData.m_iSecondaryAmmoType
		LocalWeaponData.m_nViewModelIndex
		LocalWeaponData.m_bFlipViewModel
		LocalWeaponData.m_iWeaponOrigin
		LocalWeaponData.m_iWeaponModule
		LocalActiveWeaponData.m_flNextPrimaryAttack
		LocalActiveWeaponData.m_flNextSecondaryAttack
		LocalActiveWeaponData.m_nNextThinkTick
		LocalActiveWeaponData.m_flTimeWeaponIdle
		m_Collision.m_vecMins
		m_Collision.m_vecMaxs
		m_Collision.m_nSolidType
		m_Collision.m_usSolidFlags
		m_Collision.m_nSurroundType
		m_Collision.m_vecSpecifiedSurroundingMins
		m_nModelIndex
		m_nRenderFX
		m_nRenderMode
		m_fEffects
		m_clrRender
		m_iTeamNum
		m_iPendingTeamNum
		m_CollisionGroup
		m_flElasticity
		m_flShadowCastDistance
		m_hOwnerEntity
		m_hEffectEntity
		moveparent
		m_iParentAttachment
		m_iName
		movetype
		movecollide
		m_Collision.m_vecSpecifiedSurroundingMaxs
		m_iTextureFrameIndex
		m_bSimulatedEveryTick
		m_bAnimatedEveryTick
		m_bAlternateSorting
		m_bSpotted
		m_bIsAutoaimTarget
		m_fadeMinDist
		m_fadeMaxDist
		m_flFadeScale
		m_nMinCPULevel
		m_nMaxCPULevel
		m_nMinGPULevel
		m_nMaxGPULevel
		m_flUseLookAtAngle
		m_flLastMadeNoiseTime
		m_nForceBone
		m_vecForce
		m_nSkin
		m_nBody
		m_nHitboxSet
		m_flModelScale
		m_flPlaybackRate
		m_bClientSideAnimation
		m_bClientSideFrameReset
		m_bClientSideRagdoll
		m_nNewSequenceParity
		m_nResetEventsParity
		m_nMuzzleFlashParity
		m_hLightingOrigin
		m_flFrozen
		m_ScaleType
		m_bSuppressAnimSounds
		m_blinktoggle
		m_viewtarget
		m_OriginalOwnerXuidLow
		m_OriginalOwnerXuidHigh
		m_nFallbackPaintKit
		m_nFallbackSeed
		m_flFallbackWear
		m_nFallbackStatTrak
		m_iViewModelIndex
		m_iWorldModelIndex
		m_iWorldDroppedModelIndex
		m_iState
		m_hOwner
		m_iClip1
		m_iClip2
		m_iPrimaryReserveAmmoCount
		m_iSecondaryReserveAmmoCount
		m_hWeaponWorldModel
		m_iNumEmptyAttacks
		m_weaponMode
		m_bSpottedByMask.000
		m_fLastShotTime
		m_flRecoilIndex
		m_hPrevOwner
		m_bBurstMode
		m_flPostponeFireReadyTime
		m_bReloadVisuallyComplete
		m_bSilencerOn
		m_flDoneSwitchingSilencer
		m_iOriginalTeamNumber
		m_iIronSightMode
		m_zoomLevel
		m_iBurstShotsRemaining
```
</details>

## Registering entity-creation & property-update handlers

Registering the entity-creation handlers needs to be done after the DataTablesParsedEvent has been dispatched.
Before that the server-classes won't be available in `Parser.ServerClasses()`.

Property-update handlers that are registered in a entity-creation handler will be triggered with the initial value.

This example prints the life-cycle of all AWPs during the game - i.e. who picked up whose AWP:

```go
p.RegisterEventHandler(func(events.DataTablesParsed) {
	// DataTablesParsed has been sent out, register entity-creation handler
	p.ServerClasses().FindByName("CWeaponAWP").OnEntityCreated(func(entity *st.Entity) {
		// Register update-hander on the owning entity (player who's holding the AWP)
		entity.FindPropertyI("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
			owner := p.GameState().Participants().FindByHandle(val.IntVal)
			if owner != nil {
				var prev string
				prevHandle := entity.FindPropertyI("m_hPrevOwner").Value().IntVal
				prevPlayer := p.GameState().Participants().FindByHandle(prevHandle)
				if prevPlayer != nil {
					if prevHandle != val.IntVal {
						prev = prevPlayer.Name + "'s"
					} else {
						prev = "his dropped"
					}
				} else {
					prev = "a brand new"
				}
				fmt.Printf("%s picked up %s AWP (#%d)\n", owner.Name, prev, entity.ID())
			}
		})
	})
})
```

## Running the example

The Example prints the life-cycle of all AWPs during the game - i.e. who picked up whose AWP.

`go run server_classes.go -demo /path/to/demo`

Sample output:

```
unconnected picked up a brand new AWP (#226)
to1nou * Seagate picked up a brand new AWP (#783)
keev picked up a brand new AWP (#231)
ALEX * Intel picked up keev's AWP (#231)
keev picked up a brand new AWP (#81)
keev picked up his dropped AWP (#81)
to1nou * Seagate picked up ALEX * Intel's AWP (#231)
keev picked up his dropped AWP (#81)
xms*ASUS ♥ /F/ picked up keev's AWP (#81)
to1nou * Seagate picked up xms*ASUS ♥ /F/'s AWP (#81)
tiziaN picked up a brand new AWP (#81)     <-- Not sure how this happened
keev picked up tiziaN's AWP (#81)
crisby picked up a brand new AWP (#204)
syrsoNR picked up crisby's AWP (#204)
crisby picked up syrsoNR's AWP (#204)
keev picked up crisby's AWP (#204)
syrsoNR picked up keev's AWP (#204)
to1nou * Seagate picked up a brand new AWP (#124)
keev picked up syrsoNR's AWP (#204)
keev picked up his dropped AWP (#204)
keev picked up his dropped AWP (#204)
keev picked up his dropped AWP (#204)
keev picked up his dropped AWP (#204)
Ex6TenZ-BALLISTIX picked up keev's AWP (#204)
keev picked up a brand new AWP (#74)
mistou * Cooler Master picked up a brand new AWP (#267)
Ex6TenZ-BALLISTIX picked up mistou * Cooler Master's AWP (#267)
to1nou * Seagate picked up a brand new AWP (#125)
ALEX * Intel picked up a brand new AWP (#164)
mistou * Cooler Master picked up ALEX * Intel's AWP (#164)
tiziaN picked up to1nou * Seagate's AWP (#125)
keev picked up tiziaN's AWP (#125)
ALEX * Intel picked up mistou * Cooler Master's AWP (#164)
mistou * Cooler Master picked up ALEX * Intel's AWP (#164)
Ex6TenZ-BALLISTIX picked up keev's AWP (#125)
Ex6TenZ-BALLISTIX picked up a brand new AWP (#171)
Ex6TenZ-BALLISTIX picked up a brand new AWP (#188)
mistou * Cooler Master picked up a brand new AWP (#203)
to1nou * Seagate picked up Ex6TenZ-BALLISTIX's AWP (#188)
ALEX * Intel picked up mistou * Cooler Master's AWP (#203)
mistou * Cooler Master picked up ALEX * Intel's AWP (#203)
crisby picked up mistou * Cooler Master's AWP (#203)
keev picked up crisby's AWP (#203)
keev picked up a brand new AWP (#203)
xms*ASUS ♥ /F/ picked up keev's AWP (#203)
Ex6TenZ-BALLISTIX picked up xms*ASUS ♥ /F/'s AWP (#203)
mistou * Cooler Master picked up Ex6TenZ-BALLISTIX's AWP (#203)
keev picked up mistou * Cooler Master's AWP (#203)
kzy LJ∼ picked up to1nou * Seagate's AWP (#188)
keev picked up kzy LJ∼'s AWP (#188)
Ex6TenZ-BALLISTIX picked up keev's AWP (#188)
mistou * Cooler Master picked up Ex6TenZ-BALLISTIX's AWP (#188)
mistou * Cooler Master picked up his dropped AWP (#188)
Ex6TenZ-BALLISTIX picked up a brand new AWP (#186)
to1nou * Seagate picked up Ex6TenZ-BALLISTIX's AWP (#186)
```
