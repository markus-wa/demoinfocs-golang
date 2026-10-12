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

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/config"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
)

func TestNewParser_AppliesOptions(t *testing.T) {
	p := NewParser(bytes.NewReader(make([]byte, 1024)),
		option.MsgQueueBufferSize(0),
		option.EagerVelocityTracking(true),
		option.NetMessageDecryptionKey([]byte("key")),
	).(*parser)
	defer p.Close()

	assert.NotNil(t, p.msgQueue, "sequential parsing should init the queue right away")
	assert.True(t, p.eagerVelocityTracking)
	assert.Equal(t, []byte("key"), p.decryptionKey)
	assert.Equal(t, config.FormatFile, p.config.Format)
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

	p, err := NewCSTVBroadcastParser(t.Context(), srv.URL, option.HTTPClient(client), option.CSTVTimeout(time.Minute))
	require.NoError(t, err)

	defer p.Close()

	assert.EqualValues(t, 3, requests.Load(), "sync, start & full should be requested via the custom client")
	assert.Equal(t, config.FormatCSTVBroadcast, p.(*parser).config.Format)
	assert.Equal(t, time.Minute, p.(*parser).config.CSTVTimeout)
}
