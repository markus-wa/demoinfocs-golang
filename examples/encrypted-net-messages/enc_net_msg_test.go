package main

import (
	"os"
	"testing"
)

// Just make sure the example runs
func TestEncryptedNetMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	os.Args = []string{"cmd", "-demo", "../../test/cs-demos/match730_003528806449641685104_1453182610_271.dem", "-info", "../../test/cs-demos/match730_003528806449641685104_1453182610_271.dem.info"}

	main()
}

// Make sure it doesn't error / crash
func TestEncryptedNetMessages_BadKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	os.Args = []string{"cmd", "-demo", "../../test/cs-demos/match730_003528806449641685104_1453182610_271.dem", "-info", "../../test/cs-demos/match730_003449478367177343081_1946274414_112.dem.info"}

	main()
}
