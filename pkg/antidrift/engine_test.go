package antidrift_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestEngine_LockAndVerifyOK(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "contract.txt")
	if err := os.WriteFile(path, []byte("stable"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	writeConfig(t, root, []string{"contract.txt"})

	eng := antidrift.New(root)
	if err := eng.Lock([]string{"contract.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}

	diff, err := eng.Verify()
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !diff.OK() {
		t.Fatalf("expected OK, got %#v", diff)
	}
}

func TestEngine_VerifyDetectsDrift(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "contract.txt")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	writeConfig(t, root, []string{"contract.txt"})

	eng := antidrift.New(root)
	if err := eng.Lock([]string{"contract.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}

	if err := os.WriteFile(path, []byte("v2-drifted"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	diff, err := eng.Verify()
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if diff.OK() {
		t.Fatal("expected drift to be detected")
	}
	if len(diff.Changed) != 1 || diff.Changed[0].Path != "contract.txt" {
		t.Fatalf("unexpected changed entries: %#v", diff.Changed)
	}
	if diff.Changed[0].Expected == "" || diff.Changed[0].Actual == "" {
		t.Fatal("expected hashes in changed entry")
	}
	report := diff.String()
	if !strings.Contains(report, "contract.txt") {
		t.Fatalf("report missing path: %s", report)
	}
}

func TestEngine_VerifyDetectsMissing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "contract.txt")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	writeConfig(t, root, []string{"contract.txt"})

	eng := antidrift.New(root)
	if err := eng.Lock([]string{"contract.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}

	diff, err := eng.Verify()
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(diff.Missing) != 1 || diff.Missing[0] != "contract.txt" {
		t.Fatalf("expected missing contract.txt, got %#v", diff)
	}
}

func TestEngine_VerifyDetectsExtra(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "locked.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write locked: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "extra.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write extra: %v", err)
	}
	// Declared contracts include extra.txt, but lock only covers locked.txt.
	writeConfig(t, root, []string{"locked.txt", "extra.txt"})

	eng := antidrift.New(root)
	if err := eng.Lock([]string{"locked.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}

	diff, err := eng.Verify()
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if diff.OK() {
		t.Fatal("expected Extra to fail verification")
	}
	if len(diff.Extra) != 1 || diff.Extra[0] != "extra.txt" {
		t.Fatalf("expected Extra=[extra.txt], got %#v", diff)
	}
	if !strings.Contains(diff.String(), "extra   extra.txt") {
		t.Fatalf("human report missing extra: %s", diff.String())
	}
}

func TestEngine_VerifyIgnoresUndeclaredNeighbors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "locked.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write locked: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("noise"), 0o644); err != nil {
		t.Fatalf("write neighbor: %v", err)
	}
	writeConfig(t, root, []string{"locked.txt"})

	eng := antidrift.New(root)
	if err := eng.Lock([]string{"locked.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}

	diff, err := eng.Verify()
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !diff.OK() {
		t.Fatalf("undeclared neighbors must not be Extra: %#v", diff)
	}
}

func writeConfig(t *testing.T, root string, contracts []string) {
	t.Helper()
	cfg := tokens.DefaultConfig("example.com/app")
	cfg.ContractFiles = contracts
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, tokens.ConfigFileName), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
