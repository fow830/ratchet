package doctor_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/doctor"
	gha "github.com/fow830/ratchet/pkg/github"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestDoctorChecksSchemaFile(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n\ngo 1.22\n"), 0o644)
	cfg := tokens.DefaultConfig("m")
	_ = tokens.Save(context.Background(), root, cfg)
	_ = os.WriteFile(filepath.Join(root, tokens.CursorRules), []byte("x"), 0o644)
	_ = antidrift.New(root).Lock(context.Background(), cfg.ContractFiles)
	_, _ = gha.WriteWorkflow(root)
	_ = os.Mkdir(filepath.Join(root, tokens.GitDir), tokens.FileModeDir)
	_ = os.MkdirAll(filepath.Join(root, tokens.SchemaDir), tokens.FileModeDir)
	_ = os.WriteFile(filepath.Join(root, filepath.FromSlash(tokens.SchemaRel)), []byte(`{"$schema":"x"}`), tokens.FileModeFile)

	rep, err := doctor.Run(context.Background(), root, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range rep.Checks {
		if c.Name == doctor.CheckNameJSONSchema {
			found = true
			if !c.OK {
				t.Fatalf("%+v", c)
			}
		}
	}
	if !found {
		t.Fatalf("missing json-schema check: %+v", rep.Checks)
	}
}
