package demoinfocs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/fuzzcheck"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
)

const s2DemPathInternal = "../../test/cs-demos/s2/s2.dem"

// corruptedInput is a corrupted version of a demo.
type corruptedInput struct {
	name    string
	newRead func() io.Reader
	wantEOF bool // whether only ErrUnexpectedEndOfDemo is expected (otherwise ErrDecodeFailed is too)
}

const (
	corpusFlipInputs  = 100
	corpusEarlyFlips  = 80      // inputs with the window in the first corpusEarlyBytes, so parsing them is fast
	corpusEarlyBytes  = 8 << 20 // (the corpus runs in CI with coverage, which is a lot slower)
	corpusFlipsPerIn  = 8
	corpusFlipWindow  = 512 << 10
	corpusCutAfterWin = 1 << 20
)

// s2Corpus returns truncated versions of data (biased towards the first few MB), and versions with flipped bytes:
// each has a few random bytes flipped in a 512 KB window and is cut 1 MB after it, so parsing doesn't take long.
// Most windows are in the first 8 MB, the others anywhere in the demo.
// It's deterministic.
func s2Corpus(data []byte) []corruptedInput {
	inputs := make([]corruptedInput, 0, 16+corpusFlipInputs)

	// 16 truncations: 1 KB, 2.5 KB, 6.25 KB, ... (x2.5) up to ~24 MB, then a few later ones
	for i, n := 0, 1<<10; n < len(data) && i < 12; i, n = i+1, n*4/2+n/2 {
		inputs = append(inputs, truncated(data, n))
	}

	for _, n := range []int{len(data) / 3, len(data) / 2, len(data) * 2 / 3, len(data) * 9 / 10} {
		inputs = append(inputs, truncated(data, n))
	}

	rng := rand.New(rand.NewPCG(4, 15)) // deterministic corpus

	for i := range corpusFlipInputs {
		end := len(data)
		if i < corpusEarlyFlips {
			end = min(end, corpusEarlyBytes)
		}

		start := 16 + rng.IntN(end-16-corpusFlipWindow)
		end = start + corpusFlipWindow
		cut := min(end+corpusCutAfterWin, len(data))

		window := bytes.Clone(data[start:end])

		for range corpusFlipsPerIn {
			window[rng.IntN(len(window))] ^= byte(1 + rng.IntN(255))
		}

		inputs = append(inputs, corruptedInput{
			name: fmt.Sprintf("flip %d at %d", i, start),
			newRead: func() io.Reader {
				return io.MultiReader(bytes.NewReader(data[:start]), bytes.NewReader(window), bytes.NewReader(data[end:cut]))
			},
		})
	}

	return inputs
}

func truncated(data []byte, n int) corruptedInput {
	return corruptedInput{
		name:    fmt.Sprintf("truncated at %d", n),
		newRead: func() io.Reader { return bytes.NewReader(data[:n]) },
		wantEOF: true,
	}
}

// Truncated or corrupted versions of a real demo fail with ErrUnexpectedEndOfDemo or ErrDecodeFailed,
// without panics or hangs. FuzzParse checks the memory, its handmade seeds don't get as far as a real demo.
// It runs in parallel with the other tests that parse real demos, and runs a few inputs in parallel itself.
func TestCorruptS2Corpus(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	data, err := os.ReadFile(s2DemPathInternal)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("test demo missing")
	}

	if err != nil {
		t.Fatal(err)
	}

	inputs := s2Corpus(data)

	var (
		wg  sync.WaitGroup
		sem = make(chan struct{}, max(2, runtime.NumCPU()/2))
	)

	for _, in := range inputs {
		wg.Add(1)

		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if err := parseCorrupted(in); err != nil {
				t.Errorf("%s: %v", in.name, err)
			}
		}()
	}

	wg.Wait()
}

// parseCorrupted parses in, it returns an error if the result isn't the expected one.
func parseCorrupted(in corruptedInput) (failure error) {
	// generous, coverage and race builds are slow
	stop := fuzzcheck.StartWatchdog(in.name, 2*time.Minute)
	defer stop()

	p := NewParser(in.newRead(), option.MsgQueueBufferSize(1<<10))
	defer p.Close()

	registerFuzzHandlers(p)

	defer func() {
		if r := recover(); r != nil {
			failure = fmt.Errorf("unexpected panic: %v", r)
		}
	}()

	err := p.ParseToEnd(context.Background())

	switch {
	case errors.Is(err, ErrUnexpectedEndOfDemo):
		return nil
	case !in.wantEOF && errors.Is(err, ErrDecodeFailed):
		return nil
	}

	return fmt.Errorf("unexpected error: %v", err) //nolint:err113 // test failure
}
