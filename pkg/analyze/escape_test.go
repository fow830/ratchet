package analyze_test

import (
	"testing"

	"github.com/fow830/ratchet/pkg/analyze"
)

func TestParseHints(t *testing.T) {
	const sample = `./foo.go:12:6: moved to heap: x
./bar.go:3:2: y escapes to heap
./baz.go:1:1: leaking param: p
`
	hints := analyze.ParseHints(sample)
	if len(hints) != 1 {
		t.Fatalf("hints=%+v", hints)
	}
	if hints[0].File != "./bar.go" || hints[0].Line != "3" {
		t.Fatalf("%+v", hints[0])
	}
}

func TestParseHintsEmpty(t *testing.T) {
	if hints := analyze.ParseHints("ok"); len(hints) != 0 {
		t.Fatalf("%+v", hints)
	}
}
