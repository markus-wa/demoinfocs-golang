package demoinfocs

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/cstv"
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

// newCSTVReader returns a *cstv.Reader for a broadcast whose start fragment only contains zero bytes.
// In the CSTV frame format that's a DEM_Stop frame (command 0, tick 0) followed by padding,
// in the .dem format it's an invalid filestamp.
func newCSTVReader(t *testing.T) *cstv.Reader {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sync":
			_, _ = io.WriteString(w, `{"fragment": 2, "signup_fragment": 1}`)
		case "/1/start":
			// enough data for the parser's reads so it doesn't wait for delta fragments
			_, _ = w.Write(make([]byte, 4096))
		case "/2/full":
			// empty: everything the parser needs is in the start fragment
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	r, err := cstv.NewReader(t.Context(), srv.URL)
	require.NoError(t, err)

	return r
}

func TestNewParser_DetectsCSTVReader(t *testing.T) {
	p := NewParser(newCSTVReader(t))
	defer p.Close()

	assert.Equal(t, config.FormatCSTVBroadcast, p.(*parser).config.Format)

	// the CSTV path doesn't read a PBDEMS2 filestamp and parses the DEM_Stop frame
	err := p.ParseToEnd(t.Context())
	require.NoError(t, err)
	assert.Equal(t, filestampS2, p.(*parser).header.Filestamp)
}

func TestNewParser_WrappedCSTVReaderIsNotDetected(t *testing.T) {
	// the same bytes through a wrapper are read as a .dem file
	p := NewParser(bufio.NewReader(newCSTVReader(t)))
	defer p.Close()

	assert.Equal(t, config.FormatFile, p.(*parser).config.Format)

	err := p.ParseToEnd(t.Context())
	assert.ErrorIs(t, err, ErrInvalidFileType)
}
