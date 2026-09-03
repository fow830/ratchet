package generate_test

import (
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/generate"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestRenderEnvExample(t *testing.T) {
	cfg := tokens.DefaultConfig("example.com/app")
	out := generate.RenderEnvExample(cfg)
	if !strings.Contains(out, "RATCHET_MODULE=example.com/app") {
		t.Fatalf("output: %s", out)
	}
}

func TestRenderLayersGo(t *testing.T) {
	cfg := tokens.PresetConfig(tokens.PresetVitek, "vitek")
	out := generate.RenderLayersGo(cfg)
	if !strings.Contains(out, "LayerTransport") || !strings.Contains(out, "package generated") {
		t.Fatalf("output: %s", out)
	}
}

func TestOutputs(t *testing.T) {
	cfg := tokens.DefaultConfig("m")
	outs := generate.Outputs(cfg)
	if len(outs) < 2 {
		t.Fatalf("outputs: %+v", outs)
	}
}
