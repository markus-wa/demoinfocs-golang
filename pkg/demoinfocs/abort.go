package demoinfocs

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
)

// abortState records why parsing stops. It's shared by the parsing goroutine, the message-handling goroutine
// and the context.AfterFunc() of watchContext().
//
// What ParseToEnd() / ParseNextFrame() return (see result()), in order of precedence:
//  1. a handler panic (setPanic()).
//  2. an error of the message-handling goroutine (setError()), unless ctx was cancelled before it.
//     It comes from a frame that was read before the error of the parsing goroutine (the parsing goroutine is
//     ahead of the message-handling goroutine), so it wins over that error even though it's recorded later.
//  3. whichever of the error of the parsing goroutine (setReaderError()) and the cancellation of ctx
//     (setCancelled()) happened first.
//
// If ctx is already done when an error is recorded, the cancellation counts as having happened first
// (context.AfterFunc() records it asynchronously).
type abortState struct {
	mu        sync.Mutex
	err       error       // the first error of the message-handling goroutine or cancellation cause
	cancelled bool        // err is the cancellation cause
	readerErr error       // the error of the parsing goroutine, see setReaderError()
	readerWon bool        // readerErr was recorded before the cancellation of ctx
	panicErr  *PanicError // the first handler panic, see setPanic()

	stopped atomic.Bool     // the handlers were stopped, see stop()
	ctx     context.Context // of the running ParseToEnd() call, nil if it can't be cancelled; see watchContext()
	onStop  func()          // stops all handlers, called by the first stop()
}

// recordLocked records err (cancel: as cancellation cause) unless an error or cancellation was recorded already.
// If ctx is done, its cause is recorded instead, the cancellation happened before err.
// a.mu must be held.
func (a *abortState) recordLocked(err error, cancel bool) {
	if a.err != nil {
		return
	}

	if a.ctx != nil && a.ctx.Err() != nil {
		a.err, a.cancelled = context.Cause(a.ctx), true

		return
	}

	a.err, a.cancelled = err, cancel
}

// setError records a fatal error of the message-handling goroutine and stops all handlers.
func (a *abortState) setError(err error) {
	if err == nil {
		return
	}

	a.mu.Lock()
	a.recordLocked(err, false)
	a.mu.Unlock()

	a.stop()
}

// setCancelled records the cause of the cancellation of ctx and stops all handlers.
func (a *abortState) setCancelled(cause error) {
	a.mu.Lock()
	a.recordLocked(cause, true)
	a.mu.Unlock()

	a.stop()
}

// setReaderError records the error of the parsing goroutine (only the first one is kept).
// It doesn't stop the handlers, so the messages that were read before the error can still be handled
// (and their errors take precedence, see abortState).
// It remembers whether ctx was cancelled before, so a cancellation while the queue is drained doesn't replace it.
func (a *abortState) setReaderError(err error) {
	if err == nil {
		return
	}

	a.mu.Lock()

	if a.readerErr == nil {
		if a.ctx != nil && a.ctx.Err() != nil {
			a.recordLocked(nil, true) // records the cause, the cancellation happened before err
		}

		a.readerErr = err
		a.readerWon = !a.cancelled
	}

	a.mu.Unlock()
}

// setPanic records the panic of a handler, which is re-panicked by ParseToEnd() / ParseNextFrame().
// Only the first panic is kept, later ones (e.g. of other handlers while the queue is drained) are dropped.
// It stops all handlers, so parsing stops after the current frame.
//
// It doesn't format or otherwise use c.Value, so it can't panic itself.
func (a *abortState) setPanic(c *panics.Captured) {
	a.mu.Lock()

	if a.panicErr == nil {
		a.panicErr = &PanicError{
			Value: c.Value,
			Stack: c.Stack,
		}
	}

	a.mu.Unlock()

	a.stop()
}

// stop stops all handlers (once) and makes aborted() true.
func (a *abortState) stop() {
	if a.stopped.CompareAndSwap(false, true) && a.onStop != nil {
		a.onStop()
	}
}

// aborted reports whether the handlers were stopped or ctx is done.
// Until setCancelled() has run (context.AfterFunc() runs it on a separate goroutine),
// ctx is checked directly so cancelling from within a handler takes effect immediately.
func (a *abortState) aborted() bool {
	return a.stopped.Load() || (a.ctx != nil && a.ctx.Err() != nil)
}

// result returns the handler panic and the error that takes precedence, see abortState.
func (a *abortState) result() (*PanicError, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch {
	case a.err != nil && !a.cancelled:
		return a.panicErr, a.err

	case a.readerErr != nil && a.readerWon:
		return a.panicErr, a.readerErr

	default:
		return a.panicErr, a.err
	}
}
