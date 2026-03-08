package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestScores(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	os.Args = []string{"cmd", "-demo", "../../test/cs-demos/s2/s2.dem"}

	outReader, outWriter, _ := os.Pipe()
	errReader, errWriter, _ := os.Pipe()
	os.Stdout = outWriter
	os.Stderr = errWriter

	main()

	outWriter.Close()
	errWriter.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	io.Copy(&stdoutBuf, outReader)
	io.Copy(&stderrBuf, errReader)

	expected := `> Round 4 / Site B
Bomb planted by Twister    [6m29.593759744s]
Bomb defused by Fynjy      [6m50.92186112s]
Time remaining on the bomb: 18.67 seconds

> Round 7 / Site B
Bomb planted by KEP      [10m34.531217408s]
Bomb defused by САВВА    [10m52.437487616s]
Time remaining on the bomb: 22.09 seconds

> Round 12 / Site A
Bomb planted by ezKenni        [19m18.85932544s]
Bomb defused by PLAKAL_KLEN    [19m40.750053376s]
Time remaining on the bomb: 18.11 seconds

> Round 15 / Site B
Bomb planted by ⸸ДУКАЛИС⸸    [23m3.43743488s]
Bomb defused by САВВА        [23m37.796911104s]
Time remaining on the bomb: 5.64 seconds

> Round 19 / Site B
Bomb planted by САВВА        [28m23.421935616s]
Bomb defused by ⸸ДУКАЛИС⸸    [28m59.374985216s]
Time remaining on the bomb: 4.05 seconds

`
	assert.Equal(t, expected, stdoutBuf.String())
	assert.Equal(t, "", stderrBuf.String())
}
