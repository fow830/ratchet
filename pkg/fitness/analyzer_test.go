package fitness_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/fitness"
	"github.com/fow830/ratchet/pkg/tokens"
)

func writePkg(t *testing.T, root, rel, src string) {
	t.Helper()
	dir := filepath.Join(root, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "doc.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestAnalyzer_CatchesIllegalDomainImport(t *testing.T) {
	root := t.TempDir()
	mod := "example.com/app"

	writePkg(t, root, "internal/domain", `package domain
import _ "example.com/app/internal/usecase"
`)
	writePkg(t, root, "internal/usecase", `package usecase
`)

	cfg := tokens.DefaultConfig(mod)
	a := fitness.NewAnalyzer(cfg)

	violations, err := a.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected at least one violation for domain -> usecase")
	}
	found := false
	for _, v := range violations {
		if v.ImporterLayer == "domain" && v.ImportedLayer == "usecase" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected domain->usecase violation, got %#v", violations)
	}
}

func TestAnalyzer_AllowsLegalEdges(t *testing.T) {
	root := t.TempDir()
	mod := "example.com/app"

	writePkg(t, root, "internal/domain", `package domain
`)
	writePkg(t, root, "internal/usecase", `package usecase
import _ "example.com/app/internal/domain"
`)
	writePkg(t, root, "internal/delivery", `package delivery
import (
	_ "example.com/app/internal/usecase"
	_ "example.com/app/internal/domain"
)
`)

	cfg := tokens.DefaultConfig(mod)
	a := fitness.NewAnalyzer(cfg)

	violations, err := a.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestAnalyzer_CatchesUsecaseImportingDelivery(t *testing.T) {
	root := t.TempDir()
	mod := "example.com/app"

	writePkg(t, root, "internal/delivery", `package delivery
`)
	writePkg(t, root, "internal/usecase", `package usecase
import _ "example.com/app/internal/delivery"
`)

	cfg := tokens.DefaultConfig(mod)
	a := fitness.NewAnalyzer(cfg)

	violations, err := a.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected usecase -> delivery violation")
	}
}
