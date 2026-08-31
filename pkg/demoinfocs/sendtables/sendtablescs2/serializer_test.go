package sendtablescs2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func simpleField(varName, sendNode string) *field {
	return &field{
		varName:  varName,
		name:     varName,
		sendNode: sendNode,
		model:    fieldModelSimple,
	}
}

// https://github.com/markus-wa/demoinfocs-golang/issues/673
func TestResolveFieldNameCollisions(t *testing.T) {
	t.Parallel()

	velocityX := simpleField("m_vecX", "m_vecVelocity")
	viewOffsetX := simpleField("m_vecX", "m_vecViewOffset")
	health := simpleField("m_iHealth", "")

	base := newSerializer("CBasePlayerPawn", 0)
	base.addField(velocityX)
	base.addField(viewOffsetX)
	base.addField(health)

	// field rows are shared: a derived class re-lists its parent's fields
	derived := newSerializer("CCSPlayerPawn", 0)
	derived.addField(velocityX)
	derived.addField(viewOffsetX)
	derived.addField(health)

	// a serializer containing only one field of a colliding pair must be
	// re-keyed as well, since the rename applies to the shared field object
	partial := newSerializer("CPartial", 0)
	partial.addField(velocityX)

	resolveFieldNameCollisions([]*serializer{base, derived, partial})

	assert.Equal(t, "m_vecVelocity.m_vecX", velocityX.name)
	assert.Equal(t, "m_vecViewOffset.m_vecX", viewOffsetX.name)
	assert.Equal(t, "m_iHealth", health.name)

	for _, s := range []*serializer{base, derived} {
		fp := newFieldPath()
		assert.True(t, s.getFieldPathForName(fp, "m_vecVelocity.m_vecX", nil))
		assert.Equal(t, 0, fp.path[0])

		fp = newFieldPath()
		assert.True(t, s.getFieldPathForName(fp, "m_vecViewOffset.m_vecX", nil))
		assert.Equal(t, 1, fp.path[0])

		// the bare name stays as a deprecated alias resolving to the
		// declaration that won before the rename (the last one registered)
		fp = newFieldPath()
		assert.True(t, s.getFieldPathForName(fp, "m_vecX", nil))
		assert.Equal(t, 1, fp.path[0])

		fp = newFieldPath()
		assert.True(t, s.getFieldPathForName(fp, "m_iHealth", nil))
		assert.Equal(t, 2, fp.path[0])
	}

	// in the partial serializer the alias resolves to its only declaration
	fp := newFieldPath()
	assert.True(t, partial.getFieldPathForName(fp, "m_vecVelocity.m_vecX", nil))
	fp = newFieldPath()
	assert.True(t, partial.getFieldPathForName(fp, "m_vecX", nil))
	assert.Equal(t, 0, fp.path[0])

	// name generation must produce the canonical, disambiguated names
	fp = newFieldPath()
	fp.path[0] = 0
	assert.Equal(t, []string{"m_vecVelocity.m_vecX"}, base.getNameForFieldPath(fp, 0, nil))
	fp.path[0] = 1
	assert.Equal(t, []string{"m_vecViewOffset.m_vecX"}, base.getNameForFieldPath(fp, 0, nil))
}

func TestResolveFieldNameCollisions_NoCollision(t *testing.T) {
	t.Parallel()

	health := simpleField("m_iHealth", "")
	armor := simpleField("m_ArmorValue", "")

	s := newSerializer("CCSPlayerPawn", 0)
	s.addField(health)
	s.addField(armor)

	resolveFieldNameCollisions([]*serializer{s})

	assert.Equal(t, "m_iHealth", health.name)
	assert.Equal(t, "m_ArmorValue", armor.name)

	fp := newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_iHealth", nil))
}

// tableField builds a variable-table field (e.g. a CUtlVector of a serializer
// type such as m_Attributes or m_weaponPurchases).
func tableField(varName, sendNode string, sub *serializer) *field {
	return &field{
		varName:    varName,
		name:       varName,
		sendNode:   sendNode,
		model:      fieldModelVariableTable,
		serializer: sub,
	}
}

// Renamed non-leaf fields have dots inside their own name; their generated
// child names (e.g. m_weaponPurchasesThisMatch.m_weaponPurchases.0000.m_nCount)
// must still resolve, both through the canonical name and the bare alias.
func TestResolveFieldNameCollisions_NonLeafRoundTrip(t *testing.T) {
	t.Parallel()

	count := simpleField("m_nCount", "")
	sub := newSerializer("WeaponPurchaseCount_t", 0)
	sub.addField(count)

	thisMatch := tableField("m_weaponPurchases", "m_weaponPurchasesThisMatch", sub)
	thisRound := tableField("m_weaponPurchases", "m_weaponPurchasesThisRound", sub)

	s := newSerializer("CCSPlayer_ActionTrackingServices", 0)
	s.addField(thisMatch)
	s.addField(thisRound)

	resolveFieldNameCollisions([]*serializer{s, sub})

	assert.Equal(t, "m_weaponPurchasesThisMatch.m_weaponPurchases", thisMatch.name)
	assert.Equal(t, "m_weaponPurchasesThisRound.m_weaponPurchases", thisRound.name)

	// generated name for path [0, 7, 0] must round-trip
	fp := newFieldPath()
	fp.path[0], fp.path[1], fp.path[2] = 0, 7, 0
	fp.last = 2
	name := "m_weaponPurchasesThisMatch.m_weaponPurchases.0007.m_nCount"
	assert.Equal(t,
		[]string{"m_weaponPurchasesThisMatch.m_weaponPurchases", "0007", "m_nCount"},
		s.getNameForFieldPath(fp, 0, nil),
	)

	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, name, nil))
	assert.Equal(t, []int{0, 7, 0}, fp.path[:3])
	assert.Equal(t, 2, fp.last)

	// same through the other declaration
	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_weaponPurchasesThisRound.m_weaponPurchases.0003.m_nCount", nil))
	assert.Equal(t, []int{1, 3, 0}, fp.path[:3])

	// the bare alias resolves child names to the pre-rename winner (last one)
	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_weaponPurchases.0003.m_nCount", nil))
	assert.Equal(t, []int{1, 3, 0}, fp.path[:3])

	// non-existent member below a valid table must fail, not panic
	fp = newFieldPath()
	assert.False(t, s.getFieldPathForName(fp, "m_weaponPurchasesThisMatch.m_weaponPurchases.0007.m_nope", nil))

	// malformed index segments must fail, not panic
	fp = newFieldPath()
	assert.False(t, s.getFieldPathForName(fp, "m_weaponPurchasesThisMatch.m_weaponPurchases.xx.m_nCount", nil))
}

// A shorter dotted prefix can hit a real field whose subtree does not contain
// the remainder; the walk must backtrack and try longer prefixes.
func TestGetFieldPathForName_Backtracking(t *testing.T) {
	t.Parallel()

	subX := newSerializer("SubX", 0)
	subX.addField(simpleField("m_y", ""))

	subQualified := newSerializer("SubQualified", 0)
	member := simpleField("m_y2", "")
	subQualified.addField(member)

	table := tableField("m_x", "", subX)
	qualified := tableField("m_sub", "m_x", subQualified)

	s := newSerializer("CTest", 0)
	s.addField(table)
	s.addField(qualified)

	// as if renamed by collision resolution
	qualified.name = "m_x.m_sub"
	s.rebuildFieldIndexes()

	// "m_x" hits first and its recursion fails ("m_sub…" is no index segment);
	// the walk must restore fp.last and succeed via the "m_x.m_sub" prefix.
	fp := newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_x.m_sub.0001.m_y2", nil))
	assert.Equal(t, []int{1, 1, 0}, fp.path[:3])
	assert.Equal(t, 2, fp.last)

	// plain table lookups keep working
	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_x.0002.m_y", nil))
	assert.Equal(t, []int{0, 2, 0}, fp.path[:3])

	fp = newFieldPath()
	assert.False(t, s.getFieldPathForName(fp, "m_x.m_sub.0001.m_nope", nil))
}

// Collision groups can contain a member without a send_node (e.g.
// CShatterGlassShardPhysics.m_LightGroup: "" vs m_ShardDesc). Only the
// qualified member is renamed; the bare name then belongs to the
// send_node-less field, even if the renamed one was declared later.
func TestResolveFieldNameCollisions_EmptySendNodeMember(t *testing.T) {
	t.Parallel()

	plain := simpleField("m_LightGroup", "")
	sharded := simpleField("m_LightGroup", "m_ShardDesc")

	s := newSerializer("CShatterGlassShardPhysics", 0)
	s.addField(plain)
	s.addField(sharded)

	resolveFieldNameCollisions([]*serializer{s})

	assert.Equal(t, "m_LightGroup", plain.name)
	assert.Equal(t, "m_ShardDesc.m_LightGroup", sharded.name)

	fp := newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_LightGroup", nil))
	assert.Equal(t, 0, fp.path[0])

	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_ShardDesc.m_LightGroup", nil))
	assert.Equal(t, 1, fp.path[0])
}
