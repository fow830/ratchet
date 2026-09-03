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

func TestScaffoldSuite(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "tests/contracts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := contracts.ScaffoldSuite(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
