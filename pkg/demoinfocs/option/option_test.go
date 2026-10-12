package option

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/config"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

func TestOptions(t *testing.T) {
	creators := map[int]NetMessageCreator{
		4: func() proto.Message { return new(msg.CNETMsg_Tick) },
	}

	cfg := config.New([]Option{
		MsgQueueBufferSize(0),
		AdditionalNetMessageCreators(creators),
		IgnoreErrBombsiteIndexNotFound(true),
		NetMessageDecryptionKey([]byte("key")),
		DisableMimicSource1Events(true),
		Source2FallbackGameEventListBin([]byte("bin")),
		IgnorePacketEntitiesPanic(true),
		UserCmdParsing(UserCmdParsingFull),
		EagerVelocityTracking(true),
	})

	assert.Equal(t, 0, cfg.MsgQueueBufferSize)
	assert.Len(t, cfg.AdditionalNetMessageCreators, 1)
	assert.True(t, cfg.IgnoreErrBombsiteIndexNotFound)
	assert.Equal(t, []byte("key"), cfg.NetMessageDecryptionKey)
	assert.True(t, cfg.DisableMimicSource1Events)
	assert.Equal(t, []byte("bin"), cfg.Source2FallbackGameEventListBin)
	assert.True(t, cfg.IgnorePacketEntitiesPanic)
	assert.Equal(t, config.FormatFile, cfg.Format)
	assert.Equal(t, UserCmdParsingFull, cfg.UserCmdParsing)
	assert.True(t, cfg.EagerVelocityTracking)
}

func TestOptions_LastOneWins(t *testing.T) {
	cfg := config.New([]Option{
		EagerVelocityTracking(true),
		EagerVelocityTracking(false),
		MsgQueueBufferSize(100),
	})

	assert.False(t, cfg.EagerVelocityTracking)
	assert.Equal(t, 100, cfg.MsgQueueBufferSize)
	assert.Equal(t, config.Default().UserCmdParsing, cfg.UserCmdParsing)
}
