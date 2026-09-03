package contracts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/contracts"
)

func TestScaffoldContract(t *testing.T) {
	root := t.TempDir()
	path, err := contracts.Scaffold(root, contracts.ScaffoldOpts{
		ID:      "ARCH-001",
		Title:   "Layer isolation",
		Negative: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "ARCH-001") {
		t.Fatal("missing contract id")
	}
	if !strings.Contains(body, "package contracts_test") {
		t.Fatal("missing package")
	}
}

func TestScaffoldUsesCustomContractsDir(t *testing.T) {
	root := t.TempDir()
	path, err := contracts.Scaffold(root, contracts.ScaffoldOpts{
		ID:  "ARCH-002",
		Title: "custom dir",
		Dir: "custom/contracts",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := filepath.Join(root, "custom", "contracts")
	if filepath.Dir(path) != wantPrefix {
		t.Fatalf("path=%s want under %s", path, wantPrefix)
	}
}

func TestScaffoldRejectsEmptyID(t *testing.T) {
	_, err := contracts.Scaffold(t.TempDir(), contracts.ScaffoldOpts{ID: "", Title: "x"})
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}
