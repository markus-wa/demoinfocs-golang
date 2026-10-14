package panics

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
)

// Kind is the class of a recovered panic value, see Classify.
type Kind int

const (
	// KindDecode is a failure of the library itself, e.g. while decoding corrupt data.
	KindDecode Kind = iota
	// KindUser is a panic of user code (a handler, a callback or the demo stream).
	KindUser
	// KindEndOfDemo is a bare io.EOF or io.ErrUnexpectedEOF, i.e. the demo stream ended unexpectedly.
	// It's only the end of the demo where the demo stream is read (the parsing goroutine's top level),
	// anywhere else it's a KindDecode (e.g. a nested reader running out of data).
	KindEndOfDemo
	// KindRead is a *ReadError, i.e. reading the demo stream failed.
	KindRead
)

// Classify tells what a recovered panic value r is. The rules are applied in order:
//  1. If the innermost value (see Innermost) is of type DecodeFailure, it's a KindDecode, even when it comes from user code.
//     This is used for *sendtables.PropertyNotFoundError: a missing property means the demo's schema differs from
//     what the code expects, also when the user's own code looked it up.
//  2. Tagged values (see IsTagged) are KindUser.
//  3. A bare io.EOF or io.ErrUnexpectedEOF is KindEndOfDemo.
//  4. A *ReadError is KindRead.
//  5. Anything else is KindDecode.
//
// It never panics and doesn't format r.
func Classify[DecodeFailure any](r any) Kind {
	if _, ok := Innermost(r).(DecodeFailure); ok {
		return KindDecode
	}

	if IsTagged(r) {
		return KindUser
	}

	//nolint:errorlint // only the bare values the bit reader panics with
	if r == io.EOF || r == io.ErrUnexpectedEOF {
		return KindEndOfDemo
	}

	if _, ok := r.(*ReadError); ok {
		return KindRead
	}

	return KindDecode
}

// Innermost returns the original panic value of r: (possibly nested) dispatch.ConsumerCodePanic and *Captured
// values are unwrapped. Any other value is returned as is.
func Innermost(r any) any {
	for {
		switch v := unwrapConsumerCodePanic(r).(type) {
		case *Captured:
			r = v.Value
		default:
			return v
		}
	}
}

// Failure is a recovered panic of the library itself, as an error (see NewFailure).
type Failure struct {
	Cause error  // the innermost panic value, converted to an error if it isn't one
	Stack []byte // stack trace of the panic, only set if Cause is a runtime.Error (e.g. index out of range)
}

// NewFailure converts the recovered panic value r into a *Failure.
//
// It must be called from the deferred function that recovered r, otherwise the stack trace doesn't
// contain the panicking frames.
func NewFailure(r any) *Failure {
	v := Innermost(r)

	cause, ok := v.(error)
	if !ok {
		cause = fmt.Errorf("%v", v) //nolint:err113 // the panic value isn't an error
	}

	f := &Failure{Cause: cause}

	if _, ok := v.(runtime.Error); ok {
		f.Stack = debug.Stack()
	}

	return f
}

// Error implements error.
func (f *Failure) Error() string {
	return f.Cause.Error()
}

// Unwrap returns the cause.
func (f *Failure) Unwrap() error {
	return f.Cause
}
