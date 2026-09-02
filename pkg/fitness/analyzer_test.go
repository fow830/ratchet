package fitness_test

import (
	"context"
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

func TestAnalyzer_TableDrivenEdges(t *testing.T) {
	mod := "example.com/app"
	cfg := tokens.DefaultConfig(mod)
	a := fitness.NewAnalyzer(cfg)

	tests := []struct {
		name      string
		from, to  string
		wantAllow bool
	}{
		{"domain_self", "domain", "domain", true},
		{"domain_to_usecase", "domain", "usecase", false},
		{"domain_to_delivery", "domain", "delivery", false},
		{"usecase_to_domain", "usecase", "domain", true},
		{"usecase_to_delivery", "usecase", "delivery", false},
		{"delivery_to_usecase", "delivery", "usecase", true},
		{"delivery_to_domain", "delivery", "domain", true},
		{"unknown_layer", "infra", "domain", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := a.Allowed(tc.from, tc.to); got != tc.wantAllow {
				t.Fatalf("Allowed(%s,%s)=%v want %v", tc.from, tc.to, got, tc.wantAllow)
			}
		})
	}
}

func TestAnalyzer_TableDrivenLayerOf(t *testing.T) {
	cfg := tokens.DefaultConfig("example.com/app")
	cfg.Layers["/domain/sub"] = "domain_sub"
	a := fitness.NewAnalyzer(cfg)

	tests := []struct {
		path      string
		wantLayer string
		wantOK    bool
	}{
		{"example.com/app/internal/domain", "domain", true},
		{"example.com/app/internal/domain/sub", "domain_sub", true},
		{"example.com/app/internal/usecase", "usecase", true},
		{"example.com/app/pkg/util", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got, ok := a.LayerOf(tc.path)
			if ok != tc.wantOK || got != tc.wantLayer {
				t.Fatalf("LayerOf(%q)=(%q,%v) want (%q,%v)", tc.path, got, ok, tc.wantLayer, tc.wantOK)
			}
		})
	}
}

func TestAnalyzer_TableDrivenAnalyze(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, root string)
		wantViolations int
		wantEdge       [2]string // importer, imported layers; empty if none expected specifically
	}{
		{
			name: "illegal_domain_to_usecase",
			setup: func(t *testing.T, root string) {
				writePkg(t, root, "internal/domain", "package domain\nimport _ \"example.com/app/internal/usecase\"\n")
				writePkg(t, root, "internal/usecase", "package usecase\n")
			},
			wantViolations: 1,
			wantEdge:       [2]string{"domain", "usecase"},
		},
		{
			name: "legal_clean_arch",
			setup: func(t *testing.T, root string) {
				writePkg(t, root, "internal/domain", "package domain\n")
				writePkg(t, root, "internal/usecase", "package usecase\nimport _ \"example.com/app/internal/domain\"\n")
				writePkg(t, root, "internal/delivery", "package delivery\nimport (\n_ \"example.com/app/internal/usecase\"\n_ \"example.com/app/internal/domain\"\n)\n")
			},
			wantViolations: 0,
		},
		{
			name: "illegal_usecase_to_delivery",
			setup: func(t *testing.T, root string) {
				writePkg(t, root, "internal/delivery", "package delivery\n")
				writePkg(t, root, "internal/usecase", "package usecase\nimport _ \"example.com/app/internal/delivery\"\n")
			},
			wantViolations: 1,
			wantEdge:       [2]string{"usecase", "delivery"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			tc.setup(t, root)
			a := fitness.NewAnalyzer(tokens.DefaultConfig("example.com/app"))
			vs, err := a.Analyze(context.Background(), root)
			if err != nil {
				t.Fatalf("Analyze: %v", err)
			}
			if len(vs) != tc.wantViolations {
				t.Fatalf("violations=%d want %d: %#v", len(vs), tc.wantViolations, vs)
			}
			if tc.wantViolations > 0 && tc.wantEdge[0] != "" {
				found := false
				for _, v := range vs {
					if v.ImporterLayer == tc.wantEdge[0] && v.ImportedLayer == tc.wantEdge[1] {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("missing edge %v in %#v", tc.wantEdge, vs)
				}
			}
		})
	}
}

func TestAnalyzer_CanceledContext(t *testing.T) {
	root := t.TempDir()
	writePkg(t, root, "internal/domain", "package domain\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := fitness.NewAnalyzer(tokens.DefaultConfig("example.com/app")).Analyze(ctx, root)
	if err == nil {
		t.Fatal("expected context error")
	}
}
