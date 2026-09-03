package tokens_test

import (
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
)

func TestMigrateV0ToCurrent(t *testing.T) {
	cfg := tokens.Config{
		Module: "m",
		Layers: map[string]string{"/domain": "domain"},
		AllowedEdges: map[string][]string{"domain": {}},
	}
	got, err := tokens.Migrate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != tokens.CurrentSchemaVersion {
		t.Fatalf("schema version = %d", got.SchemaVersion)
	}
	if got.Profile == "" {
		t.Fatal("profile should default")
	}
}

func TestMigrateAlreadyCurrent(t *testing.T) {
	cfg := tokens.DefaultConfig("m")
	cfg.SchemaVersion = tokens.CurrentSchemaVersion
	got, err := tokens.Migrate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != tokens.CurrentSchemaVersion {
		t.Fatal("should stay current")
	}
}
