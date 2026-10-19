// Package panics contains the building blocks to tell panics in user code (handlers, readers)
// apart from panics in the library's own code.
//
// User code is tagged where the library calls into it:
//   - net-message and game-event handlers are wrapped by the dispatcher in a dispatch.ConsumerCodePanic
//     (the parser's own net-message handlers are registered with dispatch.HandlerOptions.UnwrappedPanics,
//     so their panics stay untagged);
//   - everything else (e.g. send-table callbacks) defers GuardUser(), which re-panics with a *Captured.
//
// Everything that isn't tagged comes from the library itself.
//
// Intended for internal use only.
package panics

import (
	"fmt"
	"runtime/debug"

	dispatch "github.com/markus-wa/godispatch"
)

// Captured is a panic value recovered from user code, together with the stack trace of the panic.
type Captured struct {
	Value any    // the original panic value
	Stack []byte // stack trace of the goroutine at the time of the panic, see debug.Stack()
}

// String formats the original panic value, so a *Captured prints like the value it captured.
func (c *Captured) String() string {
	return fmt.Sprint(c.Value)
}

// Capture tags the recovered value r as a user-code panic.
//
// Values wrapped in (possibly nested) dispatch.ConsumerCodePanic are unwrapped first.
// An existing *Captured (also inside a ConsumerCodePanic) is returned as is, so its original stack is kept.
// Otherwise the stack of the current goroutine is recorded, which still contains the panicking frames
// when Capture is called from a deferred function while panicking.
func Capture(r any) *Captured {
	r = unwrapConsumerCodePanic(r)

	if c, ok := r.(*Captured); ok {
		return c
	}

	return &Captured{
		Value: r,
		Stack: debug.Stack(),
	}
}

func unwrapConsumerCodePanic(r any) any {
	for {
		ccp, ok := r.(dispatch.ConsumerCodePanic)
		if !ok {
			return r
		}

		r = ccp.Value()
	}
}

// IsTagged reports whether r is a panic from user code: a *Captured or a dispatch.ConsumerCodePanic.
//
// A ConsumerCodePanic counts as user code because the parser registers all its own net-message handlers
// with dispatch.HandlerOptions.UnwrappedPanics and the game-event dispatcher only holds user handlers.
func IsTagged(r any) bool {
	switch r.(type) {
	case *Captured, dispatch.ConsumerCodePanic:
		return true
	default:
		return false
	}
}

// GuardUser tags a panic of the calling function as a user-code panic: it recovers and re-panics with Capture(r).
//
// It must be deferred directly (defer panics.GuardUser()), recover() doesn't work from a nested function call.
func GuardUser() {
	if r := recover(); r != nil {
		panic(Capture(r))
	}
}

// ReadError is an error returned by the demo stream (io.Reader / io.Seeker) passed to the parser,
// other than io.EOF and io.ErrUnexpectedEOF.
type ReadError struct {
	Err error
}

// Error implements error.
func (e *ReadError) Error() string {
	return "failed to read demo stream: " + e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *ReadError) Unwrap() error {
	return e.Err
}
