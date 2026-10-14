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
// What ParseToEnd() / ParseNextFrame() return (see result()): a handler panic wins over everything,
// otherwise the error that was recorded first - an error of the message-handling goroutine (setError()),
// the cancellation of ctx (setCancelled()) or an error of the parsing goroutine (setReaderError()).
// If ctx is already done when an error is recorded, the cancellation counts as first
// (context.AfterFunc() records it asynchronously, see record()).
type abortState struct {
	mu       sync.Mutex
	err      error       // the first error or cancellation cause, see record()
	panicErr *PanicError // the first handler panic, see setPanic()

	stopped atomic.Bool     // the handlers were stopped, see stop()
	ctx     context.Context // of the running ParseToEnd() call, nil if it can't be cancelled; see watchContext()
	onStop  func()          // stops all handlers, called by the first stop()
}

// record records err (if non-nil) unless an error was recorded already.
// If ctx is done, its cause is recorded instead, the cancellation happened before err was recorded.
func (a *abortState) record(err error) {
	if err == nil {
		return
	}

	a.mu.Lock()

	if a.err == nil {
		if a.ctx != nil && a.ctx.Err() != nil {
			a.err = context.Cause(a.ctx)
		} else {
			a.err = err
		}
	}

	a.mu.Unlock()
}

// setError records a fatal error of the message-handling goroutine and stops all handlers.
func (a *abortState) setError(err error) {
	if err == nil {
		return
	}

	a.record(err)
	a.stop()
}

// setCancelled records the cause of the cancellation of ctx and stops all handlers.
func (a *abortState) setCancelled(cause error) {
	a.record(cause)
	a.stop()
}

// setReaderError records an error of the parsing goroutine.
// It doesn't stop the handlers, so the messages that were read before the error can still be handled.
// It's recorded before they are handled, so a cancellation of ctx while they are handled doesn't replace it.
func (a *abortState) setReaderError(err error) {
	a.record(err)
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

// result returns the handler panic and the first error, see abortState.
func (a *abortState) result() (*PanicError, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.panicErr, a.err
}

// error returns the first error, see abortState.
func (a *abortState) error() error {
	_, err := a.result()

	return err
}

// handlerPanic returns the first handler panic, see setPanic().
func (a *abortState) handlerPanic() *PanicError {
	pe, _ := a.result()

	return pe
}
