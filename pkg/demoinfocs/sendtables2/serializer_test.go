package sendtables2

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

func TestNewField_SendNodeQualifiedName(t *testing.T) {
	t.Parallel()

	i32 := func(i int32) *int32 { return &i }

	ser := &msgs2.CSVCMsg_FlattenedSerializer{
		Symbols: []string{"m_vecX", "CNetworkedQuantizedFloat", "m_vecVelocity", "(root)", "m_iHealth", "int32"},
	}

	qualified := newField(nil, ser, &msgs2.ProtoFlattenedSerializerFieldT{
		VarNameSym:  i32(0),
		VarTypeSym:  i32(1),
		SendNodeSym: i32(2),
	})
	assert.Equal(t, "m_vecVelocity.m_vecX", qualified.name)

	root := newField(nil, ser, &msgs2.ProtoFlattenedSerializerFieldT{
		VarNameSym:  i32(4),
		VarTypeSym:  i32(5),
		SendNodeSym: i32(3), // "(root)" means no send node
	})
	assert.Equal(t, "m_iHealth", root.name)

	none := newField(nil, ser, &msgs2.ProtoFlattenedSerializerFieldT{
		VarNameSym: i32(4),
		VarTypeSym: i32(5),
	})
	assert.Equal(t, "m_iHealth", none.name)
}

func simpleField(varName, sendNode string) *field {
	name := varName
	if sendNode != "" {
		name = sendNode + "." + varName
	}

	return &field{
		varName:  varName,
		name:     name,
		sendNode: sendNode,
		model:    fieldModelSimple,
	}
}

// tableField builds a variable-table field (e.g. a CUtlVector of a serializer
// type such as m_Attributes or m_weaponPurchases).
func tableField(varName, sendNode string, sub *serializer) *field {
	f := simpleField(varName, sendNode)
	f.model = fieldModelVariableTable
	f.serializer = sub

	return f
}

func TestGetFieldPathForName_SendNodeQualifiedNames(t *testing.T) {
	t.Parallel()

	s := newSerializer("CCSPlayerPawn", 0)
	s.addField(simpleField("m_vecX", "m_vecVelocity"))
	s.addField(simpleField("m_vecX", "m_vecViewOffset"))
	s.addField(simpleField("m_iHealth", ""))

	fp := newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_vecVelocity.m_vecX"))
	assert.Equal(t, 0, fp.path[0])

	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_vecViewOffset.m_vecX"))
	assert.Equal(t, 1, fp.path[0])

	fp = newFieldPath()
	assert.False(t, s.getFieldPathForName(fp, "m_vecX"), "the bare name of a send-node field must not resolve")

	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_iHealth"))
	assert.Equal(t, 2, fp.path[0])

	// name generation must produce the qualified names
	fp = newFieldPath()
	fp.path[0] = 0
	assert.Equal(t, []string{"m_vecVelocity.m_vecX"}, s.getNameForFieldPath(fp, 0))
	fp.path[0] = 1
	assert.Equal(t, []string{"m_vecViewOffset.m_vecX"}, s.getNameForFieldPath(fp, 0))
}

// Qualified non-leaf fields have dots inside their own name; their generated
// child names (e.g. m_weaponPurchasesThisMatch.m_weaponPurchases.0000.m_nCount)
// must still round-trip.
func TestGetFieldPathForName_NonLeafRoundTrip(t *testing.T) {
	t.Parallel()

	sub := newSerializer("WeaponPurchaseCount_t", 0)
	sub.addField(simpleField("m_nCount", ""))

	s := newSerializer("CCSPlayer_ActionTrackingServices", 0)
	s.addField(tableField("m_weaponPurchases", "m_weaponPurchasesThisMatch", sub))
	s.addField(tableField("m_weaponPurchases", "m_weaponPurchasesThisRound", sub))

	// generated name for path [0, 7, 0] must round-trip
	fp := newFieldPath()
	fp.path[0], fp.path[1], fp.path[2] = 0, 7, 0
	fp.last = 2
	assert.Equal(t,
		[]string{"m_weaponPurchasesThisMatch.m_weaponPurchases", "0007", "m_nCount"},
		s.getNameForFieldPath(fp, 0),
	)

	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_weaponPurchasesThisMatch.m_weaponPurchases.0007.m_nCount"))
	assert.Equal(t, []int{0, 7, 0}, fp.path[:3])
	assert.Equal(t, 2, fp.last)

	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_weaponPurchasesThisRound.m_weaponPurchases.0003.m_nCount"))
	assert.Equal(t, []int{1, 3, 0}, fp.path[:3])

	// non-existent member below a valid table must fail, not panic
	fp = newFieldPath()
	assert.False(t, s.getFieldPathForName(fp, "m_weaponPurchasesThisMatch.m_weaponPurchases.0007.m_nope"))

	// malformed index segments must fail, not panic
	fp = newFieldPath()
	assert.False(t, s.getFieldPathForName(fp, "m_weaponPurchasesThisMatch.m_weaponPurchases.xx.m_nCount"))
}

// A shorter dotted prefix can hit a real field whose subtree does not contain
// the remainder; the walk must backtrack and try longer prefixes.
func TestGetFieldPathForName_Backtracking(t *testing.T) {
	t.Parallel()

	subX := newSerializer("SubX", 0)
	subX.addField(simpleField("m_y", ""))

	subQualified := newSerializer("SubQualified", 0)
	subQualified.addField(simpleField("m_y2", ""))

	s := newSerializer("CTest", 0)
	s.addField(tableField("m_x", "", subX))
	s.addField(tableField("m_sub", "m_x", subQualified))

	// "m_x" hits first and its recursion fails ("m_sub…" is no index segment);
	// the walk must restore fp.last and succeed via the "m_x.m_sub" prefix.
	fp := newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_x.m_sub.0001.m_y2"))
	assert.Equal(t, []int{1, 1, 0}, fp.path[:3])
	assert.Equal(t, 2, fp.last)

	// plain table lookups keep working
	fp = newFieldPath()
	assert.True(t, s.getFieldPathForName(fp, "m_x.0002.m_y"))
	assert.Equal(t, []int{0, 2, 0}, fp.path[:3])

	fp = newFieldPath()
	assert.False(t, s.getFieldPathForName(fp, "m_x.m_sub.0001.m_nope"))
}
