package stfake

import (
	"runtime"
	"testing"
	"time"
	"weak"
)

func TestParsersEntryRemovedAfterGC(t *testing.T) {
	e := New(Spec{Parser: NewParser("CFoo"), Class: "CFoo", ID: 1})
	Create(e)
	Destroy(e)

	key := weak.Make(e)

	_, ok := parsers.Load(key)
	if !ok {
		t.Fatal("expected an entry for the entity")
	}

	e = nil //nolint:ineffassign,wastedassign // drop the last strong reference

	deadline := time.Now().Add(10 * time.Second)

	for {
		runtime.GC()

		if _, ok := parsers.Load(key); !ok {
			return
		}

		if time.Now().After(deadline) {
			t.Fatal("entry wasn't removed after the entity was garbage collected")
		}

		time.Sleep(time.Millisecond)
	}
}
