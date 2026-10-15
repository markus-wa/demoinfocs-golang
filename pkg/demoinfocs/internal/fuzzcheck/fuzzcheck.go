// Package fuzzcheck provides the checks of the fuzz targets that the fuzzer doesn't do itself:
// it neither detects inputs that take too long, nor ones that allocate too much memory.
//
// Only used by tests.
package fuzzcheck

import (
	"fmt"
	"runtime"
	"runtime/metrics"
	"testing"
	"time"
)

// Timeout is how long a fuzz target may take for an input, see Run().
const Timeout = 5 * time.Second

// Budget returns the number of bytes a fuzz target may allocate for an input of the given size:
// the parser's per-message limits, plus a bounded amount per input byte.
func Budget(inputLen int) uint64 {
	return 64<<20 + 1024*uint64(inputLen)
}

// AllocatedBytes returns the number of bytes f allocated on the heap (of all goroutines, so callers mustn't run
// in parallel with other tests). testing.AllocsPerRun() counts allocations, not bytes.
// It's not exact: the runtime accounts some small allocations in batches.
func AllocatedBytes(f func()) uint64 {
	sample := []metrics.Sample{{Name: "/gc/heap/allocs:bytes"}}

	metrics.Read(sample)
	before := sample[0].Value.Uint64()

	f()

	metrics.Read(sample)

	return sample[0].Value.Uint64() - before
}

// StartWatchdog panics (crashing the fuzz worker, so the fuzzer records the input) with the stacks
// of all goroutines if stop isn't called within timeout.
func StartWatchdog(name string, timeout time.Duration) (stop func()) {
	done := make(chan struct{})

	go func() {
		select {
		case <-done:
		case <-time.After(timeout):
			buf := make([]byte, 1<<22)
			buf = buf[:runtime.Stack(buf, true)]

			panic(fmt.Sprintf("%s didn't finish within %v (hang?), goroutines:\n%s", name, timeout, buf))
		}
	}()

	return func() { close(done) }
}

// Run runs f for an input of the given size. It fails tb if f allocates more than Budget(inputLen) bytes,
// and crashes the process (see StartWatchdog()) if it takes longer than Timeout.
// It must not run in parallel with other tests, see AllocatedBytes().
func Run(tb testing.TB, inputLen int, f func()) {
	tb.Helper()

	stop := StartWatchdog(tb.Name(), Timeout)
	defer stop()

	allocated := AllocatedBytes(f)

	if budget := Budget(inputLen); allocated > budget {
		tb.Fatalf("allocated %d bytes for %d input bytes, more than the budget of %d bytes", allocated, inputLen, budget)
	}
}
