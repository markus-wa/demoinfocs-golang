package cstv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
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

type Reader struct {
	baseUrl string
	sync    sync
	frag    int
	buf     bytes.Buffer
	timeout time.Duration
}

func (c *Reader) Read(p []byte) (n int, err error) {
	n, err = c.buf.Read(p)

	nFails := 0
	backoff := time.Second

	for n < len(p) && errors.Is(err, io.EOF) {
		deltaUrl := c.baseUrl + fmt.Sprintf("/%d/delta", c.frag)

		deltaResp, err := http.Get(deltaUrl)
		if err != nil {
			return n, fmt.Errorf("failed to get %q: %w", deltaUrl, err)
		}

		defer deltaResp.Body.Close()

		if deltaResp.StatusCode != http.StatusOK || deltaResp.Header.Get("Content-Type") != "application/octet-stream" {
			time.Sleep(backoff)

			backoff = time.Duration(float64(backoff) * 1.5)
			nFails++

			if nFails == 5 || backoff > c.timeout {
				return n, fmt.Errorf("%w: end of CSTV stream", io.EOF)
			}

			continue
		}

		_, err = io.Copy(&c.buf, deltaResp.Body)
		if err != nil {
			return n, fmt.Errorf("failed to read response from %q: %w", deltaUrl, err)
		}

		c.frag++
		backoff = time.Second // reset backoff on success

		n2, err := c.buf.Read(p[n:])
		n += n2
	}

	if errors.Is(err, io.EOF) {
		err = nil
	}

	return n, err
}

// NewReader creates a new CSTV reader.
// The timeout is the maximum time to retry for a response from the CSTV server,
// using an exponential backoff mechanism, starting at 1s.
// If the timeout is exceeded, the reader will return an io.EOF error.
func NewReader(baseUrl string, timeout time.Duration) (*Reader, error) {
	syncUrl := baseUrl + "/sync"

	syncResp, err := http.Get(syncUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync from %q: %w", syncUrl, err)
	}

	defer syncResp.Body.Close()

	b, err := io.ReadAll(syncResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %q: %w", syncUrl, err)
	}

	var s sync

	err = json.Unmarshal(b, &s)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response from %q: %w", syncUrl, err)
	}

	baseUrl, err = url.JoinPath(baseUrl, s.TokenRedirect)
	if err != nil {
		return nil, fmt.Errorf("failed to join base url and token redirect: %w", err)
	}

	startUrl := fmt.Sprintf(baseUrl+"/%d/start", s.SignupFragment)

	startResp, err := http.Get(startUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get %q: %w", startUrl, err)
	}

	defer startResp.Body.Close()

	var buf bytes.Buffer

	_, err = io.Copy(&buf, startResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %q: %w", startUrl, err)
	}

	fullUrl := fmt.Sprintf(baseUrl+"/%d/full", s.Fragment)

	fullResp, err := http.Get(fullUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get %q: %w", fullUrl, err)
	}

	defer fullResp.Body.Close()

	_, err = io.Copy(&buf, fullResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %q: %w", fullUrl, err)
	}

	return &Reader{
		baseUrl: baseUrl,
		sync:    s,
		buf:     buf,
		frag:    s.Fragment + 1,
		timeout: timeout,
	}, nil
}
