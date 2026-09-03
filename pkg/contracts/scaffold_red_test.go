package contracts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/contracts"
)

func TestScaffoldContractIsRED(t *testing.T) {
	root := t.TempDir()
	path, err := contracts.Scaffold(root, contracts.ScaffoldOpts{ID: "FEAT-001", Title: "feature"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if strings.Contains(body, "t.Skip") {
		t.Fatal("scaffold must not Skip — RED should fail")
	}
	if !strings.Contains(body, "t.Fatal(\"RED:") {
		t.Fatalf("want Fatal RED marker:\n%s", body)
	}
	if !strings.Contains(filepath.Base(path), "feat_001") {
		t.Fatalf("path=%s", path)
	}
}
