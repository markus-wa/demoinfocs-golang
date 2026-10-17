package sendtables

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	bit "github.com/markus-wa/demoinfocs-golang/v4/internal/bitread"
)

// emptyEntityUpdate returns the data of an entity update without any properties:
// the 'new way' bit (0) followed by the field index end marker (0xfff, 7 + 7 bits set).
func emptyEntityUpdate() *bit.BitReader {
	return bit.NewSmallBitReader(bytes.NewReader([]byte{0xfe, 0x7f, 0, 0, 0, 0, 0, 0}))
}

func newBaselineTestEntity(sc *serverClass, values ...int) (*entity, []int) {
	e := &entity{serverClass: sc, id: 42, serialNum: 1, props: make([]property, len(values))}
	updates := make([]int, len(values))

	for i, v := range values {
		e.props[i].value = PropertyValue{IntVal: v}
		i2 := i
		e.props[i].updateHandlers = []PropertyUpdateHandler{func(PropertyValue) { updates[i2]++ }}
	}

	return e, updates
}

func intValues(e *entity) []int {
	res := make([]int, len(e.props))
	for i := range e.props {
		res[i] = e.props[i].value.IntVal
	}

	return res
}

// Properties that aren't sent when re-entering the PVS have their entity baseline value.
func TestEntity_ApplyEnterPVS_EntityBaseline(t *testing.T) {
	e, updates := newBaselineTestEntity(&serverClass{}, 10, 20, 30)

	e.applyEnterPVS(emptyEntityUpdate(), []PropertyValue{{IntVal: 10}, {IntVal: 0}, {IntVal: 5}})

	assert.Equal(t, []int{10, 0, 5}, intValues(e))
	assert.Equal(t, []int{0, 1, 1}, updates, "only the reset properties must be updated")
}

// Without an entity baseline the properties that aren't sent have their instance baseline value.
func TestEntity_ApplyEnterPVS_InstanceBaseline(t *testing.T) {
	sc := &serverClass{preprocessedBaseline: map[int]PropertyValue{1: {IntVal: 7}, 2: {IntVal: 30}}}
	e, updates := newBaselineTestEntity(sc, 10, 20, 30)

	e.applyEnterPVS(emptyEntityUpdate(), nil)

	assert.Equal(t, []int{0, 7, 30}, intValues(e))
	assert.Equal(t, []int{1, 1, 0}, updates)
}

func TestSendTableParser_EntityBaselines(t *testing.T) {
	sc := &serverClass{}
	p := NewSendTableParser()
	e, _ := newBaselineTestEntity(sc, 1, 2)

	// a message using baseline 0 and updating the baseline stores the entities entering the PVS in baseline 1
	p.BeginPacketEntities(true, 0, true)
	assert.Nil(t, p.enterPVSBaseline(sc, e.id), "nothing in baseline 0 yet")
	p.updateEntityBaseline(e)

	e.props[0].value = PropertyValue{IntVal: 100}

	// the next messages use baseline 1
	p.BeginPacketEntities(true, 1, false)
	assert.Equal(t, []PropertyValue{{IntVal: 1}, {IntVal: 2}}, p.enterPVSBaseline(sc, e.id), "baseline must not change with the entity")
	assert.Nil(t, p.enterPVSBaseline(&serverClass{}, e.id), "baseline of another server-class")
	assert.Nil(t, p.enterPVSBaseline(sc, e.id+1))

	p.updateEntityBaseline(e)
	assert.Equal(t, []PropertyValue{{IntVal: 1}, {IntVal: 2}}, p.enterPVSBaseline(sc, e.id), "only updated by messages updating the baseline")

	// updating baseline 0 from baseline 1 keeps the entities that don't enter the PVS
	p.BeginPacketEntities(true, 1, true)
	p.BeginPacketEntities(true, 0, false)
	assert.Equal(t, []PropertyValue{{IntVal: 1}, {IntVal: 2}}, p.enterPVSBaseline(sc, e.id))

	// full updates use the instance baselines and clear the entity baselines
	p.BeginPacketEntities(false, 0, false)
	assert.Nil(t, p.enterPVSBaseline(sc, e.id))

	p.BeginPacketEntities(true, 0, false)
	assert.Nil(t, p.enterPVSBaseline(sc, e.id))
	p.BeginPacketEntities(true, 1, false)
	assert.Nil(t, p.enterPVSBaseline(sc, e.id))
}
