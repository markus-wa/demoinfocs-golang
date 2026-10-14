package demoinfocs

import (
	"errors"
	"fmt"
	"io"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// Parsing errors
var (
	// ErrUnexpectedEndOfDemo signals that the demo is incomplete / corrupt -
	// these demos may still be useful, check how far the parser got.
	//
	// Errors returned by ParseToEnd() / ParseNextFrame() for truncated demos also match io.ErrUnexpectedEOF (errors.Is()).
	ErrUnexpectedEndOfDemo = errors.New("demo stream ended unexpectedly (ErrUnexpectedEndOfDemo)")

	// ErrInvalidFileType signals that the input isn't a valid CS2 demo.
	ErrInvalidFileType = errors.New("invalid File-Type; expecting PBDEMS2 in the first 8 bytes (ErrInvalidFileType)")

	// ErrDecodeFailed signals that some data of the demo couldn't be decoded, see DecodeError.
	ErrDecodeFailed = errors.New("failed to decode demo data (ErrDecodeFailed)")
)

// DecodeError is returned by ParseToEnd() / ParseNextFrame() if some data of the demo couldn't be decoded.
// Either the data is corrupt, or the demo uses a schema / format that this version of the library doesn't support
// (e.g. after a game update), see also sendtables.PropertyNotFoundError.
//
// The events dispatched before the error are valid.
//
// errors.Is(err, ErrDecodeFailed) reports whether err is a DecodeError.
type DecodeError struct {
	Frame int    // the demo frame being decoded, see Parser.CurrentFrame()
	Cause error  // what went wrong
	Stack []byte // stack trace of the failure, only set if Cause is a runtime.Error (e.g. index out of range)
}

// Error implements error.
func (e *DecodeError) Error() string {
	msg := fmt.Sprintf("%s at frame %d: %v", ErrDecodeFailed.Error(), e.Frame, e.Cause)

	if e.Stack != nil {
		msg += "\nstacktrace:\n" + string(e.Stack)
	}

	return msg
}

// Is reports whether target is ErrDecodeFailed.
func (e *DecodeError) Is(target error) bool {
	return target == ErrDecodeFailed //nolint:errorlint // sentinel comparison, see errors.Is()
}

// Unwrap returns the cause.
func (e *DecodeError) Unwrap() error {
	return e.Cause
}

// newDecodeError converts a library failure into a *DecodeError.
func newDecodeError(f *panics.Failure, frame int) *DecodeError {
	return &DecodeError{
		Frame: frame,
		Cause: f.Cause,
		Stack: f.Stack,
	}
}

// PanicError is the value ParseToEnd() and ParseNextFrame() panic with when a handler panicked
// (or another callback, e.g. of an entity or the demo stream).
// The panic is re-raised on the goroutine that called ParseToEnd() / ParseNextFrame(), even if the handler
// ran on the parser's message-handling goroutine.
//
// To stop parsing from a handler, cancel the context passed to ParseToEnd() instead of panicking
// (e.g. with context.WithCancelCause()).
type PanicError struct {
	Value any    // the value the handler panicked with
	Stack []byte // stack trace of the goroutine at the time of the panic, contains the handler's frames
}

// Error implements error, it contains the panic value and the stack trace.
func (e *PanicError) Error() string {
	return fmt.Sprintf("handler panicked: %v\nstacktrace:\n%s", e.Value, e.Stack)
}

// Unwrap returns the panic value if it is an error, nil otherwise.
func (e *PanicError) Unwrap() error {
	err, _ := e.Value.(error)

	return err
}

// classifyPanic classifies a recovered panic value, see panics.Classify().
// A *sendtables.PropertyNotFoundError is a decoding failure wherever it comes from (schema drift).
func classifyPanic(r any) panics.Kind {
	return panics.Classify[*st.PropertyNotFoundError](r)
}

// errUnexpectedEndOfDemo is returned when the demo stream ended before the end of the demo.
var errUnexpectedEndOfDemo = fmt.Errorf("%w: %w", io.ErrUnexpectedEOF, ErrUnexpectedEndOfDemo)
