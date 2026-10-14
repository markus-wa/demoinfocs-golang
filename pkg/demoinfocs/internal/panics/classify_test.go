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

// Only the end of the demo stream is an unexpected EOF, see Classify().
func TestNewFailure_EOFIsNotUnwrapped(t *testing.T) {
	for _, v := range []any{io.EOF, io.ErrUnexpectedEOF, fmt.Errorf("x: %w", io.EOF)} {
		f := NewFailure(v)

		assert.NotErrorIs(t, f, io.EOF)
		assert.NotErrorIs(t, f, io.ErrUnexpectedEOF)
		assert.Equal(t, "unexpected end of data: "+fmt.Sprint(v), f.Error())
	}
}

// The stack of a tagged panic is kept, also if it's a decoding failure (see Classify()).
func TestNewFailure_TaggedHasStack(t *testing.T) {
	decodeFailure := new(testDecodeFailure)

	c := Capture(decodeFailure)
	f := NewFailure(c)

	assert.Same(t, error(decodeFailure), f.Cause)
	assert.Equal(t, c.Stack, f.Stack)

	f = NewFailure(consumerCodePanic(decodeFailure))

	assert.Same(t, error(decodeFailure), f.Cause)
	assert.NotEmpty(t, f.Stack)
}

func recoverDecode(f func()) (err error) {
	defer RecoverDecode(&err)

	f()

	return nil
}

func TestRecoverDecode(t *testing.T) {
	assert.NoError(t, recoverDecode(func() {}))

	t.Run("untagged", func(t *testing.T) {
		for _, v := range []any{"boom", errors.New("boom"), io.EOF, new(testDecodeFailure)} {
			err := recoverDecode(func() { panic(v) })

			var f *Failure
			require.ErrorAs(t, err, &f)
			assert.Contains(t, f.Cause.Error(), fmt.Sprint(v))
		}
	})

	t.Run("runtime error has stack", func(t *testing.T) {
		var f *Failure
		require.ErrorAs(t, recoverDecode(indexOutOfRange), &f)
		assert.Contains(t, string(f.Stack), "indexOutOfRange")
	})

	// Tagged panics propagate, also if they are decoding failures: only the top level classifies them.
	t.Run("tagged", func(t *testing.T) {
		decodeFailure := new(testDecodeFailure)

		for _, v := range []any{Capture("boom"), consumerCodePanic("boom"), Capture(decodeFailure), consumerCodePanic(decodeFailure)} {
			r := recoverFrom(func() { _ = recoverDecode(func() { panic(v) }) })

			c, ok := r.(*Captured)
			require.True(t, ok, "expected *Captured, got %T", r)
			assert.Equal(t, Innermost(v), c.Value)
		}
	})
}
