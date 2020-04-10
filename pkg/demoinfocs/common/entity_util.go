package common

import st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"

func getInt(entity st.IEntity, propName string) int {
	if entity == nil {
		return 0
	}

	return entity.PropertyValueMust(propName).IntVal
}

func getFloat(entity st.IEntity, propName string) float32 {
	if entity == nil {
		return 0
	}

	return entity.PropertyValueMust(propName).FloatVal
}

func getString(entity st.IEntity, propName string) string {
	if entity == nil {
		return ""
	}

	return entity.PropertyValueMust(propName).StringVal
}

func getBool(entity st.IEntity, propName string) bool {
	if entity == nil {
		return false
	}

	return entity.PropertyValueMust(propName).BoolVal()
}
