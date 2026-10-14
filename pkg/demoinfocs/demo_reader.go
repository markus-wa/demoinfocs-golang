package demoinfocs

import (
	"errors"
	"io"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
)

// demoReader wraps the demo stream passed to NewParser() for the bit reader and normalizes its errors:
//   - io.EOF, also wrapped (e.g. by *cstv.Reader), becomes a bare io.EOF.
//     If it comes with data (n > 0), the data is returned without error and the next Read() returns io.EOF.
//   - io.ErrUnexpectedEOF, also wrapped (e.g. by compress/gzip for truncated input), becomes a bare io.ErrUnexpectedEOF,
//     so the demo still counts as truncated (ErrUnexpectedEndOfDemo).
//   - Any other error becomes a *panics.ReadError.
//     The bit reader panics with it, ParseToEnd() / ParseNextFrame() return the stream's error (wrapped).
//
// Panics of the stream itself are tagged as user-code panics, ParseToEnd() / ParseNextFrame() re-panic them as *PanicError.
//
// It deliberately doesn't implement io.Closer, otherwise the bit reader would close the stream.
// Parser.Close() closes it instead.
type demoReader struct {
	r          io.Reader
	pendingEOF bool // the last Read() of r returned data together with io.EOF
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
	if r.pendingEOF {
		r.pendingEOF = false

		return 0, io.EOF
	}

	defer panics.GuardUser()

	n, err = r.r.Read(b)
	if err == nil {
		return n, nil
	}

	switch {
	case errors.Is(err, io.ErrUnexpectedEOF):
		return n, io.ErrUnexpectedEOF

	case errors.Is(err, io.EOF):
		if n > 0 {
			r.pendingEOF = true

			return n, nil
		}

		return 0, io.EOF

	default:
		return n, &panics.ReadError{Err: err}
	}
}

// Seek implements io.Seeker.
func (r seekingDemoReader) Seek(offset int64, whence int) (int64, error) {
	defer panics.GuardUser()

	r.pendingEOF = false

	pos, err := r.s.Seek(offset, whence)
	if err != nil {
		return pos, &panics.ReadError{Err: err}
	}

	return pos, nil
}
