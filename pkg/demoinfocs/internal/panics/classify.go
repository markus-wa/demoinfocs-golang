package panics

import (
	"errors"
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
	Stack []byte // stack trace of the panic, see NewFailure
}

// NewFailure converts the recovered panic value r into a *Failure.
//
// The stack trace is kept if r is tagged as user code (see IsTagged, e.g. a decoding failure of the user's own
// lookup, see Classify) or a runtime.Error (e.g. index out of range), the stack of a *Captured value is preferred.
//
// If the innermost value is (or wraps) io.EOF / io.ErrUnexpectedEOF, e.g. because a nested reader ran out of data,
// the cause doesn't match them (errors.Is()) anymore: only the end of the demo stream is an unexpected EOF.
//
// It must be called from the deferred function that recovered r, otherwise the stack trace doesn't
// contain the panicking frames.
func NewFailure(r any) *Failure {
	v := Innermost(r)

	cause, ok := v.(error)
	if !ok {
		cause = fmt.Errorf("%v", v) //nolint:err113 // the panic value isn't an error
	} else if errors.Is(cause, io.EOF) || errors.Is(cause, io.ErrUnexpectedEOF) {
		cause = &endOfDataError{msg: cause.Error()}
	}

	f := &Failure{Cause: cause}

	if IsTagged(r) {
		f.Stack = Capture(r).Stack
	} else if _, ok := v.(runtime.Error); ok {
		f.Stack = debug.Stack()
	}

	return f
}

// endOfDataError is the cause of a Failure for io.EOF / io.ErrUnexpectedEOF, see NewFailure.
// It deliberately doesn't unwrap to them.
type endOfDataError struct {
	msg string
}

// Error implements error.
func (e *endOfDataError) Error() string {
	return "unexpected end of data: " + e.msg
}

// Error implements error.
func (f *Failure) Error() string {
	return f.Cause.Error()
}

// Unwrap returns the cause.
func (f *Failure) Unwrap() error {
	return f.Cause
}

// RecoverDecode is the recover boundary of a decoding step that returns its failures as an error
// instead of panicking. It must be deferred directly: defer panics.RecoverDecode(&err),
// recover() doesn't work from a nested function call.
//
//   - Tagged panics (user code, see IsTagged) keep propagating, re-panicked with Capture(r).
//     This includes tagged panics whose innermost value is a decoding failure (e.g. a missing property, see Classify):
//     only the parser's top-level boundary decides what they are, a decoding step must never skip them.
//   - Any other panic is stored in *errp as a *Failure.
func RecoverDecode(errp *error) {
	r := recover()
	if r == nil {
		return
	}

	if IsTagged(r) {
		panic(Capture(r))
	}

	*errp = NewFailure(r)
}
