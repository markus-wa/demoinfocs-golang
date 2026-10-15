package sendtables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldStateSet_MaxFieldIndex(t *testing.T) {
	t.Parallel()

	for _, path := range [][]int{{maxFieldIndex}, {1, maxFieldIndex}} {
		fp := &fieldPath{path: path, last: len(path) - 1}

		assert.NotPanics(t, func() { newFieldState().set(fp, "x") }, "set(%v)", path)
	}

	for _, path := range [][]int{{maxFieldIndex + 1}, {1, maxFieldIndex + 1}} {
		fp := &fieldPath{path: path, last: len(path) - 1}

		assert.Panics(t, func() { newFieldState().set(fp, "x") }, "set(%v)", path)
	}
}

func TestFieldStateSet_Allocated(t *testing.T) {
	t.Parallel()

	s := &fieldState{}

	assert.Equal(t, 101, s.set(&fieldPath{path: []int{99}}, 1)) // z+2
	assert.Zero(t, s.set(&fieldPath{path: []int{99}}, 1))
	assert.Equal(t, 200, s.set(&fieldPath{path: []int{100}}, 1)) // 2x
	assert.Zero(t, s.set(&fieldPath{path: []int{150}}, 1))
	assert.Equal(t, newFieldStateSlots, s.set(&fieldPath{path: []int{1, 2}, last: 1}, 1))
	assert.Equal(t, 1002+newFieldStateSlots, s.set(&fieldPath{path: []int{2, 1000}, last: 1}, 1))
}

func TestEntityBudget(t *testing.T) {
	t.Parallel()

	fp := &fieldPath{path: []int{1}}

	b := newEntityBudget()
	b.spendState(entityBudgetBaseStateSlots, fp)
	b.spendBaseline(entityBudgetBaseBaselineBytes, 1)

	assert.PanicsWithValue(t, "entity state exceeds the budget of 32 slots per byte of entity data (path [1]): corrupt or desynced demo bitstream", func() {
		b.spendState(1, fp)
	})
	assert.PanicsWithValue(t, "decoded instance baselines exceed the budget of 64 bytes per byte of entity data (class 1): corrupt demo", func() {
		b.spendBaseline(1, 1)
	})

	b = newEntityBudget()
	b.credit(10)
	b.spendState(entityBudgetBaseStateSlots+10*entityBudgetStateSlotsPerByte, fp)
	b.spendBaseline(entityBudgetBaseBaselineBytes+10*entityBudgetBaselineBytesPerByte, 1)

	assert.Panics(t, func() { b.spendState(1, fp) })

	// the remaining budget is bounded
	b = newEntityBudget()

	for range 1000 {
		b.credit(1 << 30)
	}

	assert.Equal(t, entityBudgetMax, b.stateSlots)
	assert.Equal(t, entityBudgetMax, b.baselineBytes)
}
