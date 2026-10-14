package demoinfocs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
)

// scriptedReader returns the given results, one per Read() call.
type scriptedReader struct {
	results []scriptedRead
}

type scriptedRead struct {
	n   int
	err error
}

func (r *scriptedReader) Read(b []byte) (int, error) {
	if len(r.results) == 0 {
		return 0, io.EOF
	}

	res := r.results[0]
	r.results = r.results[1:]

	return min(res.n, len(b)), res.err
}

func TestDemoReader_EOFWithData(t *testing.T) {
	r := newDemoReader(&scriptedReader{results: []scriptedRead{
		{n: 3, err: fmt.Errorf("%w: end of stream", io.EOF)},
		{n: 0, err: errors.New("must not be read")},
	}})

	b := make([]byte, 8)

	n, err := r.Read(b)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)

	n, err = r.Read(b)
	assert.Zero(t, n)
	assert.Same(t, io.EOF, err) // must be the bare io.EOF
}

func TestDemoReader_WrappedEOF(t *testing.T) {
	r := newDemoReader(&scriptedReader{results: []scriptedRead{{n: 0, err: fmt.Errorf("%w: end of stream", io.EOF)}}})

	n, err := r.Read(make([]byte, 8))
	assert.Zero(t, n)
	assert.Same(t, io.EOF, err) // must be the bare io.EOF
}

func TestDemoReader_WrappedUnexpectedEOF(t *testing.T) {
	r := newDemoReader(&scriptedReader{results: []scriptedRead{{n: 2, err: fmt.Errorf("gzip: %w", io.ErrUnexpectedEOF)}}})

	n, err := r.Read(make([]byte, 8))
	assert.Equal(t, 2, n)
	assert.Same(t, io.ErrUnexpectedEOF, err) // must be the bare io.ErrUnexpectedEOF
}

func TestDemoReader_ReadError(t *testing.T) {
	r := newDemoReader(iotest.ErrReader(io.ErrClosedPipe))

	_, err := r.Read(make([]byte, 8))

	var readErr *panics.ReadError
	require.ErrorAs(t, err, &readErr)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
}

type failingSeeker struct {
	io.Reader
}

func (failingSeeker) Seek(int64, int) (int64, error) {
	return 0, io.ErrClosedPipe
}

func TestDemoReader_Seeker(t *testing.T) {
	_, ok := newDemoReader(iotest.ErrReader(io.ErrClosedPipe)).(io.Seeker)
	assert.False(t, ok, "must only implement io.Seeker if the stream does")

	r := newDemoReader(bytes.NewReader([]byte{1, 2, 3, 4}))
	s, ok := r.(io.Seeker)
	require.True(t, ok)

	pos, err := s.Seek(2, io.SeekCurrent)
	require.NoError(t, err)
	assert.Equal(t, int64(2), pos)

	b := make([]byte, 4)
	n, err := r.Read(b)
	require.NoError(t, err)
	assert.Equal(t, []byte{3, 4}, b[:n])

	_, err = newDemoReader(failingSeeker{}).(io.Seeker).Seek(1, io.SeekCurrent)

	var readErr *panics.ReadError
	require.ErrorAs(t, err, &readErr)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
}

type closeCountingReader struct {
	io.Reader
	closed int
}

func (r *closeCountingReader) Close() error {
	r.closed++

	return nil
}

func TestDemoReader_NotACloser(t *testing.T) {
	_, ok := newDemoReader(&closeCountingReader{}).(io.Closer)
	assert.False(t, ok, "the bit reader must not close the demo stream")

	_, ok = newDemoReader(struct {
		io.ReadSeekCloser
	}{}).(io.Closer)
	assert.False(t, ok, "the bit reader must not close the demo stream")
}

type panickingReader struct{}

//go:noinline
func (panickingReader) Read([]byte) (int, error) {
	panic("boom")
}

func TestDemoReader_StreamPanicIsTagged(t *testing.T) {
	r := func() (r any) {
		defer func() {
			r = recover()
		}()

		_, _ = newDemoReader(panickingReader{}).Read(make([]byte, 8))

		return nil
	}()

	c, ok := r.(*panics.Captured)
	require.True(t, ok, "expected *panics.Captured, got %T", r)
	assert.Equal(t, "boom", c.Value)
	assert.Contains(t, string(c.Stack), "panickingReader")
}

func TestNewParser_EmptyStream(t *testing.T) {
	var p *Parser

	require.NotPanics(t, func() {
		p = NewParser(bytes.NewReader(nil))
	})

	defer p.Close()

	err := p.ParseToEnd(t.Context())
	assert.ErrorIs(t, err, ErrUnexpectedEndOfDemo)
}

func TestNewParser_EmptyStream_ParseNextFrame(t *testing.T) {
	p := NewParser(bytes.NewReader(nil))
	defer p.Close()

	more, err := p.ParseNextFrame()
	assert.False(t, more)
	assert.ErrorIs(t, err, ErrUnexpectedEndOfDemo)
}

func TestNewParser_DoesntReadTheStream(t *testing.T) {
	p := NewParser(panickingReader{})
	assert.NoError(t, p.Close())
}

// Not treated differently yet (see demoReader): the bit reader's panic is re-panicked.
func TestParseToEnd_ReadErrorPanics(t *testing.T) {
	p := NewParser(iotest.ErrReader(io.ErrClosedPipe))
	defer p.Close()

	r := func() (r any) {
		defer func() {
			r = recover()
		}()

		_ = p.ParseToEnd(t.Context())

		return nil
	}()

	err, ok := r.(error)
	require.True(t, ok, "expected an error, got %T", r)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestParser_Close_ClosesDemoStream(t *testing.T) {
	t.Run("before parsing", func(t *testing.T) {
		stream := &closeCountingReader{Reader: bytes.NewReader(nil)}

		p := NewParser(stream)
		require.NoError(t, p.Close())

		assert.Equal(t, 1, stream.closed)
	})

	t.Run("after parsing", func(t *testing.T) {
		stream := &closeCountingReader{Reader: bytes.NewReader(make([]byte, 1024))}

		p := NewParser(stream)

		err := p.ParseToEnd(t.Context())
		require.ErrorIs(t, err, ErrInvalidFileType)
		assert.Zero(t, stream.closed, "parsing must not close the stream")

		require.NoError(t, p.Close())
		assert.Equal(t, 1, stream.closed, "the stream must be closed exactly once")
	})
}
