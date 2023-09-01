package sendtables2

type fieldPatch struct {
	minBuild uint32
	maxBuild uint32
	patch    func(f *field)
}

var fieldPatches = []fieldPatch{
	/*
		m_FieldEncoderOverrides =
		[
			DemoSimpleEncoders_t { m_Name =  "m_flSimulationTime"					m_VarType = "NET_DATA_TYPE_UINT64" },
			DemoSimpleEncoders_t { m_Name =  "m_flAnimTime"							m_VarType = "NET_DATA_TYPE_UINT64" },
		]
	*/
	{0, 0, func(f *field) {
		switch f.varName {
		case "m_flSimulationTime", "m_flAnimTime":
			f.encoder = "simtime"
		}
	}},
}

func (p *fieldPatch) shouldApply(build uint32) bool {
	if p.minBuild == 0 && p.maxBuild == 0 {
		return true
	}

	return build >= p.minBuild && build <= p.maxBuild
}
