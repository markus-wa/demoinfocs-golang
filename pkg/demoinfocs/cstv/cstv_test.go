package cstv_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/cstv"
)

// newBroadcastServer serves a broadcast with signup fragment 1, full fragment 5
// and the delta fragments 5 & 6, behind the token redirect "tok".
func newBroadcastServer(t *testing.T) *httptest.Server {
	t.Helper()

	fragments := map[string]string{
		"/bc/tok/1/start": "start-",
		"/bc/tok/5/full":  "full-",
		"/bc/tok/5/delta": "delta5-",
		"/bc/tok/6/delta": "delta6",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bc/sync" {
			_, _ = io.WriteString(w, `{"tick": 1000, "fragment": 5, "signup_fragment": 1, "token_redirect": "tok"}`)

			return
		}

		body, ok := fragments[r.URL.Path]
		if !ok {
			http.NotFound(w, r)

			return
		}

		_, _ = io.WriteString(w, body)
	}))

	t.Cleanup(srv.Close)

	return srv
}

func TestReader(t *testing.T) {
	t.Parallel()

	srv := newBroadcastServer(t)

	r, err := cstv.NewReader(t.Context(), srv.URL+"/bc", cstv.WithTimeout(time.Second))
	require.NoError(t, err)

	start := time.Now()
	b, err := io.ReadAll(r)

	// the end of the broadcast is detected by the retries for fragment 7 failing until the timeout
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, "start-full-delta5-delta6", string(b))
	assert.Less(t, time.Since(start), 5*time.Second)
}

type recordingTransport struct {
	mu    sync.Mutex
	paths []string
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	rt.paths = append(rt.paths, req.URL.Path)
	rt.mu.Unlock()

	return http.DefaultTransport.RoundTrip(req)
}

func TestReader_WithHTTPClient(t *testing.T) {
	t.Parallel()

	srv := newBroadcastServer(t)
	rt := new(recordingTransport)

	r, err := cstv.NewReader(t.Context(), srv.URL+"/bc",
		cstv.WithTimeout(time.Second),
		cstv.WithHTTPClient(&http.Client{Transport: rt}),
	)
	require.NoError(t, err)

	_, err = io.ReadAll(r)
	assert.ErrorIs(t, err, io.EOF)

	rt.mu.Lock()
	defer rt.mu.Unlock()

	assert.Equal(t, []string{
		"/bc/sync",
		"/bc/tok/1/start",
		"/bc/tok/5/full",
		"/bc/tok/5/delta",
		"/bc/tok/6/delta",
		"/bc/tok/7/delta", // fails, retried until the timeout
	}, rt.paths)
}

func TestReader_CancelDuringBackoff(t *testing.T) {
	t.Parallel()

	srv := newBroadcastServer(t)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// default timeout (10s), the first backoff alone is 1s
	r, err := cstv.NewReader(ctx, srv.URL+"/bc")
	require.NoError(t, err)

	time.AfterFunc(200*time.Millisecond, cancel)

	start := time.Now()
	_, err = io.ReadAll(r)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(start), 900*time.Millisecond, "Read didn't return promptly after cancellation")
}

func TestNewReader_ContextDone(t *testing.T) {
	t.Parallel()

	srv := newBroadcastServer(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := cstv.NewReader(ctx, srv.URL+"/bc")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestNewReader_InvalidSync(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "not json")
	}))
	defer srv.Close()

	_, err := cstv.NewReader(t.Context(), srv.URL)

	var syntaxErr *json.SyntaxError
	assert.ErrorAs(t, err, &syntaxErr)
}
