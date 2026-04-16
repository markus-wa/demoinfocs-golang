package sendtablescs2

type fieldPatch struct {
	patch func(f *field)
}

var fieldPatches = []fieldPatch{
	/*
		m_FieldEncoderOverrides =
		[
			DemoSimpleEncoders_t { m_Name =  "m_flSimulationTime"					m_VarType = "NET_DATA_TYPE_UINT64" },
			DemoSimpleEncoders_t { m_Name =  "m_flAnimTime"							m_VarType = "NET_DATA_TYPE_UINT64" },
		]
	*/
	{func(f *field) {
		switch f.varName {
		case "m_flSimulationTime", "m_flAnimTime":
			f.encoder = "simtime"
		}
	}},
}
