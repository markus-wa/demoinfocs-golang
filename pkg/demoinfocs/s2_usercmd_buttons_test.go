package demoinfocs

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestMergeUserCmdButtonsFullAndDelta(t *testing.T) {
	full := bytesField(1, bytesField(3, varintField(1, 512)))
	var buttons uint64
	require.NoError(t, mergeUserCmdButtons(full, 0, &buttons))
	require.EqualValues(t, 512, buttons)

	unchanged := bytesField(1, varintField(2, 101))
	require.NoError(t, mergeUserCmdButtons(unchanged, 0, &buttons))
	require.EqualValues(t, 512, buttons)

	reset := bytesField(1, bytesField(3, resetMarker(1)))
	require.NoError(t, mergeUserCmdButtons(reset, 0, &buttons))
	require.Zero(t, buttons)
}

func TestMergeUserCmdButtonsSkipsOtherWireTypes(t *testing.T) {
	data := append(varintField(2, 123), bytesField(4, []byte{1, 2, 3})...)
	var buttons uint64
	require.NoError(t, mergeUserCmdButtons(data, 0, &buttons))
	require.Zero(t, buttons)
}

func TestMergeUserCmdButtonsRejectsWrongTargetWireType(t *testing.T) {
	data := bytesField(1, bytesField(3, bytesField(1, []byte{1})))
	var buttons uint64
	require.Error(t, mergeUserCmdButtons(data, 0, &buttons))
}

func TestUserCmdButtonRingRequiresExactCommandNumber(t *testing.T) {
	var ring userCmdButtonRing
	ring.insert(150, 512)

	buttons, ok := ring.get(150)
	require.True(t, ok)
	require.EqualValues(t, 512, buttons)
	_, ok = ring.get(0)
	require.False(t, ok)
}

func TestUserCmdParsingModeDefaultsToButtonsOnly(t *testing.T) {
	var config ParserConfig
	require.Equal(t, UserCmdParsingButtonsOnly, config.UserCmdParsing)
	require.Equal(t, UserCmdParsingButtonsOnly, DefaultParserConfig.UserCmdParsing)
	require.Equal(t, protowire.Type(7), wireTypeReset)
}
