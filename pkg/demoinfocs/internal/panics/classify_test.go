package panics

import (
	"errors"
	"fmt"
	"io"
	"testing"

	dispatch "github.com/markus-wa/godispatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testDecodeFailure struct{}

func (*testDecodeFailure) Error() string { return "decode failure" }

// consumerCodePanic dispatches a handler that panics with v, so v ends up in a dispatch.ConsumerCodePanic.
func consumerCodePanic(v any) any {
	d := new(dispatch.Dispatcher)
	d.RegisterHandler(func(int) { panic(v) })

	return recoverFrom(func() { d.Dispatch(1) })
}

func TestClassify(t *testing.T) {
	decodeFailure := new(testDecodeFailure)
	userErr := errors.New("user")

	tests := []struct {
		name string
		r    any
		want Kind
	}{
		{"decode failure", decodeFailure, KindDecode},
		{"decode failure in user code (captured)", Capture(decodeFailure), KindDecode},
		{"decode failure in user code (consumer code panic)", consumerCodePanic(decodeFailure), KindDecode},
		{"decode failure in user code (nested)", consumerCodePanic(Capture(decodeFailure)), KindDecode},
		{"captured", Capture("boom"), KindUser},
		{"captured error", Capture(userErr), KindUser},
		{"captured EOF", Capture(io.EOF), KindUser},
		{"captured read error", Capture(&ReadError{Err: userErr}), KindUser},
		{"consumer code panic", consumerCodePanic("boom"), KindUser},
		{"nested consumer code panic", consumerCodePanic(consumerCodePanic("boom")), KindUser},
		{"io.EOF", io.EOF, KindEndOfDemo},
		{"io.ErrUnexpectedEOF", io.ErrUnexpectedEOF, KindEndOfDemo},
		{"wrapped io.EOF", fmt.Errorf("x: %w", io.EOF), KindDecode},
		{"read error", &ReadError{Err: userErr}, KindRead},
		{"string", "boom", KindDecode},
		{"error", userErr, KindDecode},
		{"runtime error", runtimeError(), KindDecode},
		{"int", 42, KindDecode},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Classify[*testDecodeFailure](tc.r))
		})
	}
}

func runtimeError() (r any) {
	defer func() {
		r = recover()
	}()

	indexOutOfRange()

	return nil
}

func TestInnermost(t *testing.T) {
	assert.Equal(t, "boom", Innermost("boom"))
	assert.Equal(t, "boom", Innermost(Capture("boom")))
	assert.Equal(t, "boom", Innermost(consumerCodePanic(Capture("boom"))))
	assert.Equal(t, "boom", Innermost(consumerCodePanic(consumerCodePanic("boom"))))
	assert.Nil(t, Innermost(nil))
}

func TestNewFailure(t *testing.T) {
	err := errors.New("boom")

	f := NewFailure(err)
	assert.Same(t, err, f.Cause)
	assert.Nil(t, f.Stack)
	assert.ErrorIs(t, f, err)
	assert.Equal(t, "boom", f.Error())

	f = NewFailure("boom")
	assert.EqualError(t, f.Cause, "boom")
	assert.Nil(t, f.Stack)

	decodeFailure := new(testDecodeFailure)
	f = NewFailure(Capture(decodeFailure))
	assert.Same(t, error(decodeFailure), f.Cause)
}

//go:noinline
func indexOutOfRange() {
	s := []int{}
	i := len(s) + 1

	_ = s[i]
}

func TestNewFailure_RuntimeErrorHasStack(t *testing.T) {
	var f *Failure

	func() {
		defer func() {
			f = NewFailure(recover())
		}()

		indexOutOfRange()
	}()

	require.NotNil(t, f)
	assert.Contains(t, string(f.Stack), "indexOutOfRange")
}
