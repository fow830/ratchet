package benchlock_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/benchlock"
)

func TestLockAndVerifyOK(t *testing.T) {
	root := t.TempDir()
	entries := []benchlock.Entry{
		{Name: "BenchmarkFoo", NsPerOp: 100, BytesPerOp: 0, AllocsPerOp: 0},
	}
	if err := benchlock.New(root).Lock(context.Background(), entries); err != nil {
		t.Fatal(err)
	}
	diff, err := benchlock.New(root).Verify(context.Background(), entries)
	if err != nil {
		t.Fatal(err)
	}
	if !diff.OK() {
		t.Fatalf("diff: %+v", diff)
	}
}

func TestVerifyRegression(t *testing.T) {
	root := t.TempDir()
	eng := benchlock.New(root)
	base := []benchlock.Entry{{Name: "BenchmarkFoo", NsPerOp: 100}}
	if err := eng.Lock(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	current := []benchlock.Entry{{Name: "BenchmarkFoo", NsPerOp: 500}}
	diff, err := eng.Verify(context.Background(), current)
	if err != nil {
		t.Fatal(err)
	}
	if diff.OK() {
		t.Fatal("expected regression")
	}
	if len(diff.Regressed) != 1 {
		t.Fatalf("regressed: %+v", diff.Regressed)
	}
}

func TestParseBenchOutput(t *testing.T) {
	const sample = `BenchmarkFoo-8   	 1000000	      1234 ns/op	       0 B/op	       0 allocs/op
BenchmarkLayerOf-8   	 9078184	       123.2 ns/op	       0 B/op	       0 allocs/op
`
	entries, err := benchlock.ParseOutput(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries: %+v", entries)
	}
	if entries[0].Name != "BenchmarkFoo" || entries[0].NsPerOp != 1234 {
		t.Fatalf("%+v", entries[0])
	}
	if entries[1].Name != "BenchmarkLayerOf" || entries[1].NsPerOp != 123 {
		t.Fatalf("%+v", entries[1])
	}
}

func TestLockFileWritten(t *testing.T) {
	root := t.TempDir()
	_ = benchlock.New(root).Lock(context.Background(), []benchlock.Entry{{Name: "B", NsPerOp: 1}})
	if _, err := os.Stat(filepath.Join(root, benchlock.FileName)); err != nil {
		t.Fatal(err)
	}
}
