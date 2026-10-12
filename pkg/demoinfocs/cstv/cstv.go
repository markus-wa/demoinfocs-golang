// Package cstv reads live CSTV broadcasts, see NewReader().
package cstv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type sync struct {
	Tick             int     `json:"tick"`
	EndTick          int     `json:"endtick"`
	MaxTick          int     `json:"maxtick"`
	RtDelay          float64 `json:"rtdelay"`
	RcvAge           float64 `json:"rcvage"`
	Fragment         int     `json:"fragment"`
	SignupFragment   int     `json:"signup_fragment"`
	Tps              int     `json:"tps"`
	KeyframeInterval int     `json:"keyframe_interval"`
	Map              string  `json:"map"`
	Protocol         int     `json:"protocol"`
	TokenRedirect    string  `json:"token_redirect"`
}

// Reader reads a live CSTV broadcast as a continuous demo stream.
//
// It stores the context.Context passed to NewReader() because io.Reader.Read() can't take one:
// the context is used for all HTTP requests and backoff waits made while reading.
type Reader struct {
	ctx     context.Context
	client  *http.Client
	baseURL string
	sync    sync
	frag    int
	buf     bytes.Buffer
	timeout time.Duration
}

// Option configures a Reader.
type Option func(*Reader)

// WithTimeout sets the maximum time to retry for a response from the CSTV server,
// using an exponential backoff mechanism, starting at 1s (10s by default).
func WithTimeout(timeout time.Duration) Option {
	return func(r *Reader) {
		r.timeout = timeout
	}
}

// WithHTTPClient sets the HTTP client used for all requests to the CSTV server (http.DefaultClient by default).
func WithHTTPClient(client *http.Client) Option {
	return func(r *Reader) {
		if client != nil {
			r.client = client
		}
	}
}

func (c *Reader) get(url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %q: %w", url, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get %q: %w", url, err)
	}

	return resp, nil
}

// fetchInto appends the response body of url to c.buf.
func (c *Reader) fetchInto(url string) error {
	resp, err := c.get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	_, err = io.Copy(&c.buf, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response from %q: %w", url, err)
	}

	return nil
}

// fetchDelta appends the next delta fragment to c.buf.
// Returns false if the fragment isn't available (yet).
func (c *Reader) fetchDelta() (bool, error) {
	deltaURL := c.baseURL + fmt.Sprintf("/%d/delta", c.frag)

	deltaResp, err := c.get(deltaURL)
	if err != nil {
		return false, err
	}

	defer deltaResp.Body.Close()

	if deltaResp.StatusCode != http.StatusOK {
		return false, nil
	}

	_, err = io.Copy(&c.buf, deltaResp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response from %q: %w", deltaURL, err)
	}

	return true, nil
}

// wait waits for d or until c.ctx is done, whichever happens first.
func (c *Reader) wait(d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-c.ctx.Done():
		return context.Cause(c.ctx)
	}
}

// Read reads the broadcast, fetching new fragments from the CSTV server as needed.
// Returns an error wrapping io.EOF once no new fragment was available within the timeout
// (the end of the broadcast) and context.Cause(ctx) once the Reader's context is done.
func (c *Reader) Read(p []byte) (n int, err error) {
	n, err = c.buf.Read(p)

	nFails := 0
	backoff := time.Second

	for n < len(p) && errors.Is(err, io.EOF) {
		ok, fetchErr := c.fetchDelta()
		if fetchErr != nil {
			return n, fetchErr
		}

		if !ok {
			waitErr := c.wait(backoff)
			if waitErr != nil {
				return n, waitErr
			}

			backoff = time.Duration(float64(backoff) * 1.5)
			nFails++

			if nFails == 5 || backoff > c.timeout {
				return n, fmt.Errorf("%w: end of CSTV stream", io.EOF)
			}

			continue
		}

		c.frag++
		backoff = time.Second // reset backoff on success

		// bytes.Buffer.Read can only fail with io.EOF, which err already is at this point.
		n2, _ := c.buf.Read(p[n:])
		n += n2
	}

	if errors.Is(err, io.EOF) {
		err = nil
	}

	return n, err
}

// NewReader creates a new CSTV reader for the broadcast at baseURL, e.g. "http://localhost:8080/s85568392932860274t1733091777".
// The ctx is used for all requests to the CSTV server, including the ones made by Read(),
// so it should live as long as the Reader is used.
// See WithTimeout() and WithHTTPClient() for the options.
// If the timeout is exceeded while waiting for new fragments, Read() returns an io.EOF error.
//
// Pass the Reader directly to demoinfocs.NewParser(), which detects it and parses the CSTV broadcast format;
// a Reader wrapped in another io.Reader (e.g. bufio.Reader) isn't detected and is parsed as a .dem file:
//
//	r, err := cstv.NewReader(ctx, baseURL, cstv.WithTimeout(30*time.Second))
//	if err != nil {
//		return err
//	}
//
//	p := demoinfocs.NewParser(r)
//	defer p.Close()
//
//	err = p.ParseToEnd(ctx)
func NewReader(ctx context.Context, baseURL string, opts ...Option) (*Reader, error) {
	r := &Reader{
		ctx:     ctx,
		client:  http.DefaultClient,
		timeout: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(r)
	}

	syncURL := baseURL + "/sync"

	syncResp, err := r.get(syncURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync: %w", err)
	}

	defer syncResp.Body.Close()

	b, err := io.ReadAll(syncResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %q: %w", syncURL, err)
	}

	var s sync

	err = json.Unmarshal(b, &s)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response from %q: %w", syncURL, err)
	}

	baseURL, err = url.JoinPath(baseURL, s.TokenRedirect)
	if err != nil {
		return nil, fmt.Errorf("failed to join base url and token redirect: %w", err)
	}

	err = r.fetchInto(fmt.Sprintf(baseURL+"/%d/start", s.SignupFragment))
	if err != nil {
		return nil, err
	}

	err = r.fetchInto(fmt.Sprintf(baseURL+"/%d/full", s.Fragment))
	if err != nil {
		return nil, err
	}

	r.baseURL = baseURL
	r.sync = s
	r.frag = s.Fragment

	return r, nil
}
