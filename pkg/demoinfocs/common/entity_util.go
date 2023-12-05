package common

import st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"

func getInt(entity st.Entity, propName string) int {
	if entity == nil {
		return 0
	}

	return entity.PropertyValueMust(propName).Int()
}

func getUInt64(entity st.Entity, propName string) uint64 {
	if entity == nil {
		return 0
	}

	if entity.Property(propName) == nil {
		return 0
	}

	return entity.PropertyValueMust(propName).S2UInt64()
}

func getFloat(entity st.Entity, propName string) float32 {
	if entity == nil {
		return 0
	}

	return entity.PropertyValueMust(propName).Float()
}

func getString(entity st.Entity, propName string) string {
	if entity == nil {
		return ""
	}

	return entity.PropertyValueMust(propName).String()
}

func getBool(entity st.Entity, propName string) bool {
	if entity == nil {
		return false
	}

	return entity.PropertyValueMust(propName).BoolVal()
}
