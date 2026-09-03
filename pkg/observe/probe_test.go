package observe_test

import (
	"context"
	"testing"

	"github.com/fow830/ratchet/pkg/observe"
)

func TestProbeTools(t *testing.T) {
	rep := observe.Probe(context.Background())
	if len(rep.Tools) == 0 {
		t.Fatal("expected tool probes")
	}
	found := false
	for _, tool := range rep.Tools {
		if tool.Name == observe.ToolGo {
			found = true
			if !tool.Available {
				t.Fatal("go must be available")
			}
		}
	}
	if !found {
		t.Fatal("missing go probe")
	}
}
