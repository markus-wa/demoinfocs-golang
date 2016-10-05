package dt

import ()

type SendTable struct {
	properties []*SendTableProperty
	Name       string
	IsEnd      bool
}

func (st *SendTable) Properties() []*SendTableProperty {
	return st.properties
}
