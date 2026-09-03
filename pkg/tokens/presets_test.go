package tokens_test

import (
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
)

func TestPresetClean(t *testing.T) {
	cfg := tokens.PresetConfig(tokens.PresetClean, "example.com/mod")
	if cfg.Layers[tokens.SuffixDomain] != tokens.LayerDomain {
		t.Fatalf("layers: %#v", cfg.Layers)
	}
	if len(cfg.AllowedEdges[tokens.LayerDomain]) != 0 {
		t.Fatal("domain must be leaf")
	}
}

func TestPresetVitek(t *testing.T) {
	cfg := tokens.PresetConfig(tokens.PresetVitek, "vitek")
	for _, suffix := range []string{"/transport", "/service", "/repository", "/domain"} {
		if _, ok := cfg.Layers[suffix]; !ok {
			t.Fatalf("missing layer suffix %s", suffix)
		}
	}
	found := false
	for _, e := range cfg.ForbiddenEdges {
		if e.From == "transport" && e.To == "repository" {
			found = true
		}
	}
	if !found {
		t.Fatal("vitek preset must forbid transport→repository")
	}
}

func TestPresetHex(t *testing.T) {
	cfg := tokens.PresetConfig(tokens.PresetHex, "hex/mod")
	if cfg.Layers["/adapter"] != "adapter" {
		t.Fatalf("hex adapter layer missing: %#v", cfg.Layers)
	}
}

func TestPresetUnknown(t *testing.T) {
	cfg := tokens.PresetConfig("nope", "m")
	if cfg.Preset != "" {
		t.Fatal("unknown preset should use clean preset without preset tag")
	}
}
