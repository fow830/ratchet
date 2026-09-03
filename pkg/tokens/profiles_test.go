package tokens_test

import (
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
)

func TestProfileGatesStandard(t *testing.T) {
	g := tokens.ProfileGates(tokens.ProfileStandard)
	if !g.Arch || !g.Lock || !g.Vet {
		t.Fatalf("standard: %#v", g)
	}
	if g.Race || g.Fuzz || g.Mutation {
		t.Fatal("standard must not enable race/fuzz/mutation")
	}
}

func TestProfileGatesStrict(t *testing.T) {
	g := tokens.ProfileGates(tokens.ProfileStrict)
	for _, want := range []bool{g.Arch, g.Lock, g.Vet, g.Race, g.Fuzz, g.Contracts, g.PBT} {
		if !want {
			t.Fatal("strict profile missing gate")
		}
	}
}

func TestProfileGatesParanoid(t *testing.T) {
	g := tokens.ProfileGates(tokens.ProfileParanoid)
	if !g.Mutation || !g.Bench || !g.WASM {
		t.Fatalf("paranoid: %#v", g)
	}
}

func TestNormalizeProfileDefault(t *testing.T) {
	if got := tokens.NormalizeProfile(""); got != tokens.ProfileStandard {
		t.Fatalf("got %q", got)
	}
}
