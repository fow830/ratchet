package breaking_test

import (
	"testing"

	"github.com/fow830/ratchet/pkg/breaking"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestDiffLayers(t *testing.T) {
	old := tokens.DefaultConfig("m")
	newC := old
	newC.AllowedEdges = map[string][]string{
		tokens.LayerDomain:   {},
		tokens.LayerUsecase:  {tokens.LayerDomain},
		tokens.LayerDelivery: {tokens.LayerDomain},
	}
	changes := breaking.DiffConfig(old, newC)
	if len(changes) == 0 {
		t.Fatal("expected breaking changes")
	}
}
