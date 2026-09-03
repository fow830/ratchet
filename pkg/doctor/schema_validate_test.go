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

func TestDoctorSchemaInvalidJSONFails(t *testing.T) {
	root := t.TempDir()
	writeDoctorBase(t, root)
	_ = os.MkdirAll(filepath.Join(root, tokens.SchemaDir), tokens.FileModeDir)
	_ = os.WriteFile(filepath.Join(root, filepath.FromSlash(tokens.SchemaRel)), []byte(`{not-json`), tokens.FileModeFile)

	cfg := tokens.DefaultConfig("m")
	rep, err := doctor.Run(context.Background(), root, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range rep.Checks {
		if c.Name == doctor.CheckNameJSONSchema {
			if c.OK {
				t.Fatalf("expected schema fail: %+v", c)
			}
			return
		}
	}
	t.Fatal("missing json-schema check")
}

func TestDoctorWorkflowOptional(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, tokens.GoModFileName), []byte("module m\n\ngo 1.22\n"), tokens.FileModeFile)
	cfg := tokens.DefaultConfig("m")
	_ = tokens.Save(context.Background(), root, cfg)
	_ = os.WriteFile(filepath.Join(root, tokens.CursorRules), []byte("x"), tokens.FileModeFile)
	_ = antidrift.New(root).Lock(context.Background(), cfg.ContractFiles)
	_ = os.Mkdir(filepath.Join(root, tokens.GitDir), tokens.FileModeDir)

	rep, err := doctor.Run(context.Background(), root, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Healthy() {
		t.Fatalf("expected healthy without workflow: %+v", rep.Checks)
	}
	found := false
	for _, c := range rep.Checks {
		if c.Name == doctor.CheckNameWorkflow {
			found = true
			if !c.OK {
				t.Fatalf("workflow should be optional: %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("missing workflow check")
	}
}

func writeDoctorBase(t *testing.T, root string) {
	t.Helper()
	_ = os.WriteFile(filepath.Join(root, tokens.GoModFileName), []byte("module m\n\ngo 1.22\n"), tokens.FileModeFile)
	cfg := tokens.DefaultConfig("m")
	_ = tokens.Save(context.Background(), root, cfg)
	_ = os.WriteFile(filepath.Join(root, tokens.CursorRules), []byte("x"), tokens.FileModeFile)
	_ = antidrift.New(root).Lock(context.Background(), cfg.ContractFiles)
	_, _ = gha.WriteWorkflow(root)
	_ = os.Mkdir(filepath.Join(root, tokens.GitDir), tokens.FileModeDir)
}
