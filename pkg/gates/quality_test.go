package gates_test

import (
	"testing"

	"github.com/fow830/ratchet/pkg/gates"
)

func TestParseCoverFuncTotal(t *testing.T) {
	const out = `github.com/x/a.go:10:	Foo	100.0%
github.com/x/b.go:20:	Bar	50.0%
total:						(statements)		75.0%
`
	pct, err := gates.ParseCoverFuncTotal(out)
	if err != nil {
		t.Fatal(err)
	}
	if pct != 75 {
		t.Fatalf("got %v", pct)
	}
}

func TestParseCoverageTotal(t *testing.T) {
	const out = `ok  pkg  0.1s  coverage: 45.2% of statements
ok  other 0.1s coverage: 80.0% of statements
`
	pct, err := gates.ParseCoverageTotal(out)
	if err != nil {
		t.Fatal(err)
	}
	if pct < 45 || pct > 81 {
		t.Fatalf("pct=%v", pct)
	}
}

func TestParseMutationScore(t *testing.T) {
	const out = `Total: 100
Killed: 80
Score: 80.00%
`
	pct, err := gates.ParseMutationScore(out)
	if err != nil {
		t.Fatal(err)
	}
	if pct != 80 {
		t.Fatalf("got %v", pct)
	}
}

func TestZeroAllocCheck(t *testing.T) {
	entries := []gates.BenchEntry{
		{Name: "BenchmarkHot", AllocsPerOp: 2, MarkedZeroAlloc: true},
		{Name: "BenchmarkOther", AllocsPerOp: 5, MarkedZeroAlloc: false},
	}
	if err := gates.CheckZeroAlloc(entries); err == nil {
		t.Fatal("expected zero-alloc failure")
	}
	entries[0].AllocsPerOp = 0
	if err := gates.CheckZeroAlloc(entries); err != nil {
		t.Fatal(err)
	}
}
