package demoinfocs

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

func TestDefaultConfig(t *testing.T) {
	// same values as the former DefaultParserConfig
	assert.Equal(t, config{
		MsgQueueBufferSize: -1,
		CSTVTimeout:        10 * time.Second,
		UserCmdParsing:     UserCmdParsingButtonsOnly,
	}, defaultConfig())

	assert.Equal(t, defaultConfig(), newConfig(nil))
}

func TestOptions(t *testing.T) {
	creators := map[int]NetMessageCreator{
		4: func() proto.Message { return new(msg.CNETMsg_Tick) },
	}
	client := &http.Client{}

	cfg := newConfig([]Option{
		WithMsgQueueBufferSize(0),
		WithAdditionalNetMessageCreators(creators),
		WithIgnoreErrBombsiteIndexNotFound(true),
		WithNetMessageDecryptionKey([]byte("key")),
		WithDisableMimicSource1Events(true),
		WithSource2FallbackGameEventListBin([]byte("bin")),
		WithIgnorePacketEntitiesPanic(true),
		WithUserCmdParsing(UserCmdParsingFull),
		WithEagerVelocityTracking(true),
		WithCSTVTimeout(time.Minute),
		WithHTTPClient(client),
	})

	assert.Equal(t, 0, cfg.MsgQueueBufferSize)
	assert.Len(t, cfg.AdditionalNetMessageCreators, 1)
	assert.True(t, cfg.IgnoreErrBombsiteIndexNotFound)
	assert.Equal(t, []byte("key"), cfg.NetMessageDecryptionKey)
	assert.True(t, cfg.DisableMimicSource1Events)
	assert.Equal(t, []byte("bin"), cfg.Source2FallbackGameEventListBin)
	assert.True(t, cfg.IgnorePacketEntitiesPanic)
	assert.Equal(t, demoFormatFile, cfg.Format)
	assert.Equal(t, time.Minute, cfg.CSTVTimeout)
	assert.Same(t, client, cfg.HTTPClient)
	assert.Equal(t, UserCmdParsingFull, cfg.UserCmdParsing)
	assert.True(t, cfg.EagerVelocityTracking)
}

func TestOptions_LastOneWins(t *testing.T) {
	cfg := newConfig([]Option{
		WithEagerVelocityTracking(true),
		WithEagerVelocityTracking(false),
		WithMsgQueueBufferSize(100),
	})

	assert.False(t, cfg.EagerVelocityTracking)
	assert.Equal(t, 100, cfg.MsgQueueBufferSize)
	assert.Equal(t, defaultConfig().CSTVTimeout, cfg.CSTVTimeout)
}

func TestNewParser_AppliesOptions(t *testing.T) {
	p := NewParser(bytes.NewReader(make([]byte, 1024)),
		WithMsgQueueBufferSize(0),
		WithEagerVelocityTracking(true),
		WithNetMessageDecryptionKey([]byte("key")),
	).(*parser)
	defer p.Close()

	assert.NotNil(t, p.msgQueue, "sequential parsing should init the queue right away")
	assert.True(t, p.eagerVelocityTracking)
	assert.Equal(t, []byte("key"), p.decryptionKey)
	assert.Equal(t, demoFormatFile, p.config.Format)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewCSTVBroadcastParser_ForwardsOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync":
			_, _ = io.WriteString(w, `{"fragment": 2, "signup_fragment": 1}`)
		case "/1/start":
			// enough data for the parser's initial read so it doesn't wait for delta fragments
			_, _ = w.Write(make([]byte, 4096))
		}
	}))
	defer srv.Close()

	var requests atomic.Int32

	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)

		return http.DefaultTransport.RoundTrip(req)
	})}

	p, err := NewCSTVBroadcastParser(t.Context(), srv.URL, WithHTTPClient(client), WithCSTVTimeout(time.Minute))
	require.NoError(t, err)

	defer p.Close()

	assert.EqualValues(t, 3, requests.Load(), "sync, start & full should be requested via the custom client")
	assert.Equal(t, demoFormatCSTVBroadcast, p.(*parser).config.Format)
	assert.Equal(t, time.Minute, p.(*parser).config.CSTVTimeout)
}
