package sendtablescs2

import (
	"math"
)

type fieldDecoder func(*reader) interface{}
type fieldFactory func(*field) fieldDecoder

var fieldTypeFactories = map[string]fieldFactory{
	/*
		DemoSimpleEncoders_t { m_Name = "float32"								m_VarType = "NET_DATA_TYPE_FLOAT32" },
		DemoSimpleEncoders_t { m_Name = "CNetworkedQuantizedFloat"				m_VarType = "NET_DATA_TYPE_FLOAT32" },
	*/
	"float32":                  floatFactory,
	"CNetworkedQuantizedFloat": quantizedFactory,

	"uint64": unsigned64Factory,

	/*
		// some things with > 1 component
		DemoSimpleEncoders_t { m_Name = "Vector"								m_VarType = "NET_DATA_TYPE_FLOAT32"		m_nComponents="3" },
		DemoSimpleEncoders_t { m_Name = "QAngle"								m_VarType = "NET_DATA_TYPE_FLOAT32"		m_nComponents="3" },
		DemoSimpleEncoders_t { m_Name = "Vector2D"								m_VarType = "NET_DATA_TYPE_FLOAT32"		m_nComponents="2" },
		DemoSimpleEncoders_t { m_Name = "Vector4D"								m_VarType = "NET_DATA_TYPE_FLOAT32"		m_nComponents="4" },
		DemoSimpleEncoders_t { m_Name = "Quaternion"							m_VarType = "NET_DATA_TYPE_FLOAT32"		m_nComponents="4" },
		DemoSimpleEncoders_t { m_Name = "CTransform"							m_VarType = "NET_DATA_TYPE_FLOAT32"		m_nComponents="6" },
	*/
	"Vector":     vectorFactory(3),
	"VectorWS":   vectorFactory(3),
	"Vector2D":   vectorFactory(2),
	"Vector4D":   vectorFactory(4),
	"Quaternion": vectorFactory(4),
	"CTransform": vectorFactory(6),

	"CStrongHandle": unsigned64Factory,
	"QAngle":        qangleFactory,
}

var fieldNameDecoders = map[string]fieldDecoder{
	"m_iClip1": ammoDecoder,
}

var fieldTypeDecoders = map[string]fieldDecoder{
	/*
		FIXME: dotabuff/manta doesn't have these?
				DemoSimpleEncoders_t { m_Name = "float32"								m_VarType = "NET_DATA_TYPE_FLOAT32" },
				DemoSimpleEncoders_t { m_Name = "float64"								m_VarType = "NET_DATA_TYPE_FLOAT64" },
	*/
	// "float32": noscaleDecoder,

	/*
		DemoSimpleEncoders_t { m_Name = "bool"									m_VarType = "NET_DATA_TYPE_BOOL" },

		DemoSimpleEncoders_t { m_Name = "char"									m_VarType = "NET_DATA_TYPE_INT64" },
		DemoSimpleEncoders_t { m_Name = "int8"									m_VarType = "NET_DATA_TYPE_INT64" },
		DemoSimpleEncoders_t { m_Name = "int16"									m_VarType = "NET_DATA_TYPE_INT64" },
		DemoSimpleEncoders_t { m_Name = "int32"									m_VarType = "NET_DATA_TYPE_INT64" },
		DemoSimpleEncoders_t { m_Name = "int64"									m_VarType = "NET_DATA_TYPE_INT64" },

		DemoSimpleEncoders_t { m_Name = "uint8"									m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "uint16"								m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "uint32"								m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "uint64"								m_VarType = "NET_DATA_TYPE_UINT64" },

		DemoSimpleEncoders_t { m_Name = "CUtlString"							m_VarType = "NET_DATA_TYPE_STRING" },
		DemoSimpleEncoders_t { m_Name = "CUtlSymbolLarge"						m_VarType = "NET_DATA_TYPE_STRING" },
	*/
	"bool": booleanDecoder,

	"int8":  signedDecoder,
	"int16": signedDecoder,
	"int32": signedDecoder,

	"uint8":  unsignedDecoder,
	"uint16": unsignedDecoder,
	"uint32": unsignedDecoder,

	"char":            stringDecoder,
	"CUtlString":      stringDecoder,
	"CUtlSymbolLarge": stringDecoder,

	// some dotabuff/manta stuff
	"GameTime_t": noscaleDecoder,
	"CHandle":    unsignedDecoder,

	/*
		// some commmon stufff
		DemoSimpleEncoders_t { m_Name = "Color"									m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "CUtlStringToken"						m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "EHandle"								m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "CEntityHandle"							m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "CGameSceneNodeHandle"					m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "CStrongHandle"							m_VarType = "NET_DATA_TYPE_UINT64" },
	*/
	"Color":                unsignedDecoder,
	"CUtlStringToken":      unsignedDecoder,
	"EHandle":              unsignedDecoder,
	"CEntityHandle":        unsignedDecoder,
	"CGameSceneNodeHandle": unsignedDecoder,
	"CStrongHandle":        unsignedDecoder,

	/*
		/// some commmon stufff
		DemoSimpleEncoders_t { m_Name = "HSequence"								m_VarType = "NET_DATA_TYPE_INT64" },
		DemoSimpleEncoders_t { m_Name = "AttachmentHandle_t"					m_VarType = "NET_DATA_TYPE_UINT64" }, // uint8
		DemoSimpleEncoders_t { m_Name = "CEntityIndex"							m_VarType = "NET_DATA_TYPE_INT64" },
	*/
	"HSequence":          signedDecoder,
	"AttachmentHandle_t": unsignedDecoder,
	"CEntityIndex":       signedDecoder,

	/*
		// bunch of enum types, too
		DemoSimpleEncoders_t { m_Name = "MoveCollide_t"							m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8
		DemoSimpleEncoders_t { m_Name = "MoveType_t"							m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8
		DemoSimpleEncoders_t { m_Name = "RenderMode_t"							m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8
		DemoSimpleEncoders_t { m_Name = "RenderFx_t"							m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8
		DemoSimpleEncoders_t { m_Name = "SolidType_t"							m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8
		DemoSimpleEncoders_t { m_Name = "SurroundingBoundsType_t"				m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8
		DemoSimpleEncoders_t { m_Name = "ModelConfigHandle_t"					m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8
		DemoSimpleEncoders_t { m_Name = "NPC_STATE"								m_VarType = "NET_DATA_TYPE_INT64" },	// int32
		DemoSimpleEncoders_t { m_Name = "StanceType_t"							m_VarType = "NET_DATA_TYPE_INT64" },	// int32  ?
		DemoSimpleEncoders_t { m_Name = "AbilityPathType_t"						m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32 ? no neg values
		DemoSimpleEncoders_t { m_Name = "WeaponState_t"							m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32 ? no neg values
		DemoSimpleEncoders_t { m_Name = "DoorState_t"							m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32 ? no neg values
		DemoSimpleEncoders_t { m_Name = "RagdollBlendDirection"					m_VarType = "NET_DATA_TYPE_INT64" },	// int32  ?
		DemoSimpleEncoders_t { m_Name = "BeamType_t"							m_VarType = "NET_DATA_TYPE_INT64" },	// int32  ?
		DemoSimpleEncoders_t { m_Name = "BeamClipStyle_t"						m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
		DemoSimpleEncoders_t { m_Name = "EntityDisolveType_t"					m_VarType = "NET_DATA_TYPE_INT64" },	// int32  ?
	*/
	"MoveCollide_t":           unsignedDecoder,
	"MoveType_t":              unsignedDecoder,
	"RenderMode_t":            unsignedDecoder,
	"RenderFx_t":              unsignedDecoder,
	"SolidType_t":             unsignedDecoder,
	"SurroundingBoundsType_t": unsignedDecoder,
	"ModelConfigHandle_t":     unsignedDecoder,
	"NPC_STATE":               signedDecoder,
	"StanceType_t":            signedDecoder,
	"WeaponState_t":           unsignedDecoder,
	"DoorState_t":             unsignedDecoder,
	"RagdollBlendDirection":   signedDecoder,
	"BeamType_t":              signedDecoder,
	"BeamClipStyle_t":         unsignedDecoder,
	"EntityDisolveType_t":     signedDecoder,

	/*
		DemoSimpleEncoders_t { m_Name = "ValueRemapperInputType_t"				m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
		DemoSimpleEncoders_t { m_Name = "ValueRemapperOutputType_t"				m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
		DemoSimpleEncoders_t { m_Name = "ValueRemapperHapticsType_t"			m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
		DemoSimpleEncoders_t { m_Name = "ValueRemapperMomentumType_t"			m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
		DemoSimpleEncoders_t { m_Name = "ValueRemapperRatchetType_t"			m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?

		DemoSimpleEncoders_t { m_Name = "PointWorldTextJustifyHorizontal_t"		m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
		DemoSimpleEncoders_t { m_Name = "PointWorldTextJustifyVertical_t"		m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
		DemoSimpleEncoders_t { m_Name = "PointWorldTextReorientMode_t"			m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?

		DemoSimpleEncoders_t { m_Name = "PoseController_FModType_t"				m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
		DemoSimpleEncoders_t { m_Name = "PrecipitationType_t"					m_VarType = "NET_DATA_TYPE_INT64" },	// int32  ?
		DemoSimpleEncoders_t { m_Name = "ShardSolid_t"							m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
		DemoSimpleEncoders_t { m_Name = "ShatterPanelMode"						m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32  ?
	*/
	"ValueRemapperInputType_t":          unsignedDecoder,
	"ValueRemapperOutputType_t":         unsignedDecoder,
	"ValueRemapperHapticsType_t":        unsignedDecoder,
	"ValueRemapperMomentumType_t":       unsignedDecoder,
	"ValueRemapperRatchetType_t":        unsignedDecoder,
	"PointWorldTextJustifyHorizontal_t": unsignedDecoder,
	"PointWorldTextJustifyVertical_t":   unsignedDecoder,
	"PointWorldTextReorientMode_t":      unsignedDecoder,
	"PoseController_FModType_t":         unsignedDecoder,
	"PrecipitationType_t":               signedDecoder,
	"ShardSolid_t":                      unsignedDecoder,
	"ShatterPanelMode":                  unsignedDecoder,

	/*
		DemoSimpleEncoders_t{ m_Name = "gender_t"								m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8, deprecated enum type in S2 ?

		DemoSimpleEncoders_t { m_Name = "item_definition_index_t"				m_VarType = "NET_DATA_TYPE_UINT64" },	// uint16/32 depending on game
		DemoSimpleEncoders_t { m_Name = "itemid_t"								m_VarType = "NET_DATA_TYPE_UINT64" },	// uint64
		DemoSimpleEncoders_t { m_Name = "style_index_t"							m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8
		DemoSimpleEncoders_t { m_Name = "attributeprovidertypes_t"				m_VarType = "NET_DATA_TYPE_UINT64" },	// uint32 ?
		DemoSimpleEncoders_t { m_Name = "DamageOptions_t"						m_VarType = "NET_DATA_TYPE_UINT64" },	// uint8
		DemoSimpleEncoders_t { m_Name = "ScreenEffectType_t"					m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "MaterialModifyMode_t"					m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "AmmoIndex_t"							m_VarType = "NET_DATA_TYPE_INT64" },	// int8
		DemoSimpleEncoders_t { m_Name = "TakeDamageFlags_t"						m_VarType = "NET_DATA_TYPE_INT64" },	// uint16
	*/
	"gender_t":                 unsignedDecoder,
	"item_definition_index_t":  unsignedDecoder,
	"itemid_t":                 unsignedDecoder,
	"style_index_t":            unsignedDecoder,
	"attributeprovidertypes_t": unsignedDecoder,
	"DamageOptions_t":          unsignedDecoder,
	"ScreenEffectType_t":       unsignedDecoder,
	"MaterialModifyMode_t":     unsignedDecoder,
	"AmmoIndex_t":              signedDecoder,
	"TakeDamageFlags_t":        signedDecoder,

	/*
		// csgo
		DemoSimpleEncoders_t { m_Name = "CSWeaponMode"							m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "ESurvivalSpawnTileState"				m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "SpawnStage_t"							m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "ESurvivalGameRuleDecision_t"			m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "RelativeDamagedDirection_t"			m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "CSPlayerState"							m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "MedalRank_t"							m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "CSPlayerBlockingUseAction_t"			m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "MoveMountingAmount_t"					m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "QuestProgress::Reason"					m_VarType = "NET_DATA_TYPE_UINT64" },
		DemoSimpleEncoders_t { m_Name = "tablet_skin_state_t"					m_VarType = "NET_DATA_TYPE_UINT64" },
	*/
	"CSWeaponMode":                unsignedDecoder,
	"ESurvivalSpawnTileState":     unsignedDecoder,
	"SpawnStage_t":                unsignedDecoder,
	"ESurvivalGameRuleDecision_t": unsignedDecoder,
	"RelativeDamagedDirection_t":  unsignedDecoder,
	"CSPlayerState":               unsignedDecoder,
	"MedalRank_t":                 unsignedDecoder,
	"CSPlayerBlockingUseAction_t": unsignedDecoder,
	"MoveMountingAmount_t":        unsignedDecoder,
	"QuestProgress::Reason":       unsignedDecoder,
	"tablet_skin_state_t":         unsignedDecoder,

	"CBodyComponent":    componentDecoder,
	"CPhysicsComponent": componentDecoder,
	"CLightComponent":   componentDecoder,
	"CRenderComponent":  componentDecoder,
}

func unsigned64Factory(f *field) fieldDecoder {
	switch f.encoder {
	case "fixed64":
		return fixed64Decoder
	}
	return unsigned64Decoder
}

func floatFactory(f *field) fieldDecoder {
	switch f.encoder {
	case "coord":
		return floatCoordDecoder
	case "simtime":
		return simulationTimeDecoder
	case "runetime":
		return runeTimeDecoder
	}

	if f.bitCount == nil || (*f.bitCount <= 0 || *f.bitCount >= 32) {
		return noscaleDecoder
	}

	return quantizedFactory(f)
}

func quantizedFactory(f *field) fieldDecoder {
	if f.bitCount == nil || (*f.bitCount <= 0 || *f.bitCount >= 32) {
		return noscaleDecoder
	}

	qfd := newQuantizedFloatDecoder(f.bitCount, f.encodeFlags, f.lowValue, f.highValue)

	return func(r *reader) interface{} {
		return qfd.decode(r)
	}
}

func vectorFactory(n int) fieldFactory {
	return func(f *field) fieldDecoder {
		if n == 3 && f.encoder == "normal" {
			return vectorNormalDecoder
		}

		d := floatFactory(f)
		return func(r *reader) interface{} {
			x := make([]float32, n)

			for i := 0; i < n; i++ {
				x[i] = d(r).(float32)
			}

			return x
		}
	}
}

func vectorNormalDecoder(r *reader) interface{} {
	return r.read3BitNormal()
}

func fixed64Decoder(r *reader) interface{} {
	return r.readLeUint64()
}

func handleDecoder(r *reader) interface{} {
	return r.readVarUint32()
}

func booleanDecoder(r *reader) interface{} {
	return r.readBoolean()
}

func stringDecoder(r *reader) interface{} {
	return r.readString()
}

func defaultDecoder(r *reader) interface{} {
	return r.readVarUint32()
}

func signedDecoder(r *reader) interface{} {
	return r.readVarInt32()
}

func floatCoordDecoder(r *reader) interface{} {
	return r.readCoord()
}

func ammoDecoder(r *reader) interface{} {
	return r.readVarUint32() - 1
}

func noscaleDecoder(r *reader) interface{} {
	return math.Float32frombits(r.readLeUint32())
}

func runeTimeDecoder(r *reader) interface{} {
	return math.Float32frombits(r.readBits(4))
}

func simulationTimeDecoder(r *reader) interface{} {
	return float32(r.readVarUint32()) * (1.0 / 64)
}

func readBitCoordPres(r *reader) float32 {
	return r.readAngle(20) - 180.0
}

func qanglePreciseDecoder(r *reader) interface{} {
	v := make([]float32, 3)
	hasX := r.readBoolean()
	hasY := r.readBoolean()
	hasZ := r.readBoolean()

	if hasX {
		v[0] = readBitCoordPres(r)
	}

	if hasY {
		v[1] = readBitCoordPres(r)
	}

	if hasZ {
		v[2] = readBitCoordPres(r)
	}

	return v
}

func qangleFactory(f *field) fieldDecoder {
	if f.encoder == "qangle_precise" {
		return qanglePreciseDecoder
	}

	if f.bitCount != nil && *f.bitCount != 0 {
		n := uint32(*f.bitCount)
		return func(r *reader) interface{} {
			return []float32{
				r.readAngle(n),
				r.readAngle(n),
				r.readAngle(n),
			}
		}
	}

	return func(r *reader) interface{} {
		ret := make([]float32, 3)
		rX := r.readBoolean()
		rY := r.readBoolean()
		rZ := r.readBoolean()
		if rX {
			ret[0] = r.readCoord()
		}
		if rY {
			ret[1] = r.readCoord()
		}
		if rZ {
			ret[2] = r.readCoord()
		}
		return ret
	}
}

func unsignedDecoder(r *reader) interface{} {
	return uint64(r.readVarUint32())
}

func unsigned64Decoder(r *reader) interface{} {
	return r.readVarUint64()
}

func componentDecoder(r *reader) interface{} {
	return r.readBits(1)
}

func findDecoder(f *field) fieldDecoder {
	if v, ok := fieldTypeFactories[f.fieldType.baseType]; ok {
		return v(f)
	}

	if v, ok := fieldNameDecoders[f.varName]; ok {
		return v
	}

	if v, ok := fieldTypeDecoders[f.fieldType.baseType]; ok {
		return v
	}

	return defaultDecoder
}

func findDecoderByBaseType(f *field) fieldDecoder {
	if v, ok := fieldTypeFactories[f.fieldType.genericType.baseType]; ok {
		return v(f)
	}

	if v, ok := fieldTypeDecoders[f.fieldType.genericType.baseType]; ok {
		return v
	}

	return defaultDecoder
}
