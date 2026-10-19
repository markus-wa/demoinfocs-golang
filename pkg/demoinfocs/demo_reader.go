package demoinfocs

import (
	"errors"
	"io"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
)

// demoReader wraps the demo stream passed to NewParser() for the bit reader and normalizes its errors:
//   - io.EOF, also wrapped (e.g. by *cstv.Reader), becomes a bare io.EOF.
//   - io.ErrUnexpectedEOF, also wrapped (e.g. by compress/gzip for truncated input), becomes a bare io.ErrUnexpectedEOF,
//     so the demo still counts as truncated (ErrUnexpectedEndOfDemo).
//   - Any other error becomes a *panics.ReadError.
//     The bit reader panics with it. It isn't treated differently from other panics yet, it's re-panicked
//     by ParseToEnd() / ParseNextFrame() (before, the stream's error was re-panicked as is).
//
// If an error comes with data (n > 0), the data is returned without error and the next Read() returns the error:
// the bit reader drops the data of a Read() that returns an error.
//
// Panics of the stream itself are tagged as user-code panics.
//
// It deliberately doesn't implement io.Closer, otherwise the bit reader would close the stream.
// Parser.Close() closes it instead.
type demoReader struct {
	r          io.Reader
	pendingErr error // the last Read() of r returned data together with this (normalized) error, returned by the next Read()
}

// seekingDemoReader is a demoReader for streams that implement io.Seeker,
// the bit reader uses Seek() to skip data if the stream supports it.
type seekingDemoReader struct {
	*demoReader
	s io.Seeker
}

// newDemoReader wraps r, see demoReader.
func newDemoReader(r io.Reader) io.Reader {
	dr := &demoReader{r: r}

	if s, ok := r.(io.Seeker); ok {
		return seekingDemoReader{demoReader: dr, s: s}
	}

	return dr
}

// Read implements io.Reader.
func (r *demoReader) Read(b []byte) (n int, err error) {
	if r.pendingErr != nil {
		err, r.pendingErr = r.pendingErr, nil

		return 0, err
	}

	defer panics.GuardUser()

	n, err = r.r.Read(b)
	if err == nil {
		return n, nil
	}

	switch {
	case errors.Is(err, io.ErrUnexpectedEOF):
		err = io.ErrUnexpectedEOF

	case errors.Is(err, io.EOF):
		err = io.EOF

	default:
		err = &panics.ReadError{Err: err}
	}

	if n > 0 {
		r.pendingErr = err

		return n, nil
	}

	return 0, err
}

// Seek implements io.Seeker.
func (r seekingDemoReader) Seek(offset int64, whence int) (int64, error) {
	defer panics.GuardUser()

	r.pendingErr = nil

	pos, err := r.s.Seek(offset, whence)
	if err != nil {
		return pos, &panics.ReadError{Err: err}
	}

	return pos, nil
}
