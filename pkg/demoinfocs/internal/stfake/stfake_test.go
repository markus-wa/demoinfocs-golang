package stfake_test

import (
	"testing"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stfake"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

func TestNewEntity_Properties(t *testing.T) {
	e := stfake.NewEntity(7, stfake.P("m_iHealth", int32(100)), stfake.P("m_pWeaponServices.m_hActiveWeapon", uint64(3)), stfake.P("m_unset", nil))

	assert.Equal(t, 7, e.ID())
	assert.Equal(t, 0, e.SerialNum())

	assert.Equal(t, 100, e.PropertyValueMust("m_iHealth").Int())
	assert.Equal(t, uint64(3), e.Property("m_pWeaponServices.m_hActiveWeapon").Value().Handle())
	assert.Equal(t, int32(100), e.Get("m_iHealth"))

	v, ok := e.PropertyValue("m_unset")
	assert.True(t, ok)
	assert.Nil(t, v.Any)

	assert.Nil(t, e.Property("m_hActiveWeapon"), "only the full name is declared")
	assert.Nil(t, e.Property("m_missing"))

	_, ok = e.PropertyValue("m_missing")
	assert.False(t, ok)
	assert.Panics(t, func() { e.PropertyValueMust("m_missing") })

	props := e.Properties()
	names := make([]string, 0, len(props))

	for _, p := range props {
		names = append(names, p.Name())
	}

	assert.Equal(t, []string{
		"m_iHealth", "m_pWeaponServices.m_hActiveWeapon", "m_unset",
		"CBodyComponent.m_cellX", "CBodyComponent.m_cellY", "CBodyComponent.m_cellZ",
		"CBodyComponent.m_vecX", "CBodyComponent.m_vecY", "CBodyComponent.m_vecZ",
	}, names)
}

func TestNewEntity_PropertyValueIsUnwrapped(t *testing.T) {
	e := stfake.NewEntity(1, stfake.P("m_x", st.PropertyValue{Any: float32(1.5)}))

	assert.Equal(t, float32(1.5), e.PropertyValueMust("m_x").Float())
}

func TestNewEntity_DuplicatePropPanics(t *testing.T) {
	assert.Panics(t, func() { stfake.NewEntity(1, stfake.P("m_x", 1), stfake.P("m_x", 2)) })
}

func TestPosition(t *testing.T) {
	e := stfake.NewEntity(1)
	assert.Equal(t, r3.Vector{}, e.Position(), "position props are declared by default")

	for _, pos := range []r3.Vector{
		{X: 1, Y: 2, Z: 3},
		{X: -2006, Y: 1015.5, Z: -0.25},
		{X: 16383, Y: -16384, Z: 512},
	} {
		e = stfake.NewEntity(1, stfake.Pos(pos))
		assert.Equal(t, pos, e.Position())
	}
}

func TestOnPositionUpdate(t *testing.T) {
	e := stfake.NewEntity(1, stfake.Pos(r3.Vector{X: 1}))

	var got []r3.Vector

	e.OnPositionUpdate(func(pos r3.Vector) {
		got = append(got, pos)
	})

	stfake.Update(e, stfake.Pos(r3.Vector{X: 1, Y: 2}))

	// fires per changed component, in field order (cells first, then offsets)
	require.NotEmpty(t, got)
	assert.Equal(t, r3.Vector{X: 1, Y: 2}, got[len(got)-1])
	assert.Equal(t, r3.Vector{X: 1, Y: 2}, e.Position())
}

func TestSet_Silent(t *testing.T) {
	e := stfake.NewEntity(1, stfake.P("m_x", int32(1)))

	fired := 0
	e.Property("m_x").OnUpdate(func(st.PropertyValue) { fired++ })
	e.OnUpdateFinished(func() { fired++ })

	stfake.Set(e, stfake.P("m_x", int32(2)))

	assert.Zero(t, fired)
	assert.Equal(t, 2, e.PropertyValueMust("m_x").Int())
}

func TestSet_UndeclaredPanics(t *testing.T) {
	e := stfake.NewEntity(1)

	assert.Panics(t, func() { stfake.Set(e, stfake.P("m_x", 1)) })
	assert.Panics(t, func() { stfake.Update(e, stfake.P("m_x", 1)) })
}

func TestUpdate_FieldOrderThenFinished(t *testing.T) {
	e := stfake.NewEntity(1, stfake.P("m_a", int32(0)), stfake.P("m_b", int32(0)), stfake.P("m_c", int32(0)))

	var log []string

	for _, name := range []string{"m_c", "m_a", "m_b"} {
		e.Property(name).OnUpdate(func(v st.PropertyValue) {
			log = append(log, name+"="+v.String())
		})
	}

	// m_b's handler sees m_a's new value but not yet m_c's
	e.Property("m_b").OnUpdate(func(st.PropertyValue) {
		log = append(log, "m_b sees a="+e.PropertyValueMust("m_a").String()+" c="+e.PropertyValueMust("m_c").String())
	})

	e.OnUpdateFinished(func() {
		log = append(log, "finished")
	})

	stfake.Update(e, stfake.P("m_c", int32(3)), stfake.P("m_a", int32(1)), stfake.P("m_b", int32(2)))

	assert.Equal(t, []string{"m_a=1", "m_b=2", "m_b sees a=1 c=0", "m_c=3", "finished"}, log)

	log = nil

	stfake.Update(e, stfake.P("m_b", int32(4)))

	assert.Equal(t, []string{"m_b=4", "m_b sees a=1 c=3", "finished"}, log)
}

func TestUpdateUnfinished(t *testing.T) {
	e := stfake.NewEntity(1, stfake.P("m_a", int32(0)))

	var log []string

	e.Property("m_a").OnUpdate(func(v st.PropertyValue) { log = append(log, "m_a="+v.String()) })
	e.OnUpdateFinished(func() { log = append(log, "finished") })

	stfake.UpdateUnfinished(e, stfake.P("m_a", int32(1)))

	assert.Equal(t, []string{"m_a=1"}, log)
}

func TestBind(t *testing.T) {
	e := stfake.NewEntity(1,
		stfake.P("m_int", int32(0)),
		stfake.P("m_bool", false),
		stfake.P("m_float", float32(0)),
		stfake.P("m_string", ""),
		stfake.P("m_vec", [3]float32{}),
	)

	var (
		i   int
		b   bool
		f   float32
		f64 float64
		s   string
		vec r3.Vector
	)

	e.BindProperty("m_int", &i, st.ValTypeInt)
	e.Property("m_bool").Bind(&b, st.ValTypeBoolInt)
	e.BindProperty("m_float", &f, st.ValTypeFloat32)
	e.BindProperty("m_float", &f64, st.ValTypeFloat64)
	e.BindProperty("m_string", &s, st.ValTypeString)
	e.BindProperty("m_vec", &vec, st.ValTypeVector)

	stfake.Update(e,
		stfake.P("m_int", int32(5)),
		stfake.P("m_bool", true),
		stfake.P("m_float", float32(2.5)),
		stfake.P("m_string", "x"),
		stfake.P("m_vec", [3]float32{1, 2, 3}),
	)

	assert.Equal(t, 5, i)
	assert.True(t, b)
	assert.Equal(t, float32(2.5), f)
	assert.Equal(t, 2.5, f64)
	assert.Equal(t, "x", s)
	assert.Equal(t, r3.Vector{X: 1, Y: 2, Z: 3}, vec)
}

func TestCreate_Order(t *testing.T) {
	p := stfake.NewParser("CFoo", "CBar")

	sc := p.ServerClasses().FindByName("CBar")
	require.NotNil(t, sc)
	assert.Nil(t, p.ServerClasses().FindByName("CBaz"))

	var log []string

	p.OnEntity(func(e *st.Entity, op st.EntityOp) error {
		log = append(log, "entity handler "+op.String())

		return nil
	})

	sc.OnEntityCreated(func(e *st.Entity) {
		log = append(log, "created")

		assert.Equal(t, "CBar", e.ServerClass().Name())
		assert.Equal(t, sc.ID(), e.ServerClass().ID())
		assert.Equal(t, 2, e.PropertyValueMust("m_b").Int(), "values are set before the created-handlers fire")

		e.Property("m_b").OnUpdate(func(v st.PropertyValue) { log = append(log, "m_b="+v.String()) })
		e.Property("m_a").OnUpdate(func(v st.PropertyValue) { log = append(log, "m_a="+v.String()) })
		e.OnCreateFinished(func() { log = append(log, "create finished") })
		e.OnUpdateFinished(func() { log = append(log, "update finished") })
		e.OnDestroy(func() { log = append(log, "destroyed") })
	})

	e := stfake.New(stfake.Spec{
		Parser: p,
		Class:  "CBar",
		ID:     10,
		Serial: 3,
		Props:  []stfake.Prop{stfake.P("m_a", int32(1)), stfake.P("m_b", int32(2))},
	})

	assert.Equal(t, 10, e.ID())
	assert.Equal(t, 3, e.SerialNum())
	assert.Empty(t, log, "nothing fires before Create")

	stfake.Create(e)

	// initial update handlers fire in registration order
	assert.Equal(t, []string{"created", "create finished", "entity handler Created+Entered", "m_b=2", "m_a=1", "update finished"}, log)

	log = nil

	stfake.Update(e, stfake.P("m_b", int32(3)), stfake.P("m_a", int32(4)))

	assert.Equal(t, []string{"m_a=4", "m_b=3", "update finished", "entity handler Updated"}, log)

	log = nil

	stfake.Destroy(e)
	stfake.Destroy(e)

	assert.Equal(t, []string{"destroyed", "entity handler Deleted+Left"}, log)
	assert.False(t, e.Replaced())
}

func TestCreate_WithoutParser(t *testing.T) {
	e := stfake.NewEntity(1, stfake.P("m_a", int32(1)))

	var log []string

	e.Property("m_a").OnUpdate(func(v st.PropertyValue) { log = append(log, "m_a="+v.String()) })
	e.OnCreateFinished(func() { log = append(log, "create finished") })
	e.OnUpdateFinished(func() { log = append(log, "update finished") })

	stfake.Create(e)

	assert.Equal(t, []string{"create finished", "m_a=1", "update finished"}, log)
}

func TestNew_UnknownClassPanics(t *testing.T) {
	p := stfake.NewParser("CFoo")

	assert.Panics(t, func() { stfake.New(stfake.Spec{Parser: p, Class: "CBar"}) })
}

func TestServerClass_PropertyEntries(t *testing.T) {
	e := stfake.New(stfake.Spec{Class: "CFoo", Props: []stfake.Prop{stfake.P("m_a", nil)}})

	assert.Equal(t, "CFoo", e.ServerClass().Name())
	assert.Contains(t, e.ServerClass().PropertyEntries(), "m_a")
}
