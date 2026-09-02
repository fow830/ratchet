package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/skills"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestGenerator_WritesCursorRulesAndClaudeSkill(t *testing.T) {
	root := t.TempDir()
	cfg := tokens.DefaultConfig("github.com/fow830/ratchet")

	g := skills.NewGenerator(cfg)
	if err := g.Generate(root); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	cursorRules, err := os.ReadFile(filepath.Join(root, ".cursorrules"))
	if err != nil {
		t.Fatalf("read .cursorrules: %v", err)
	}
	body := string(cursorRules)
	if !strings.Contains(body, "Zero Architectural Regression") {
		t.Fatalf(".cursorrules missing mission: %s", body)
	}
	if !strings.Contains(body, "Pure Go SSOT") {
		t.Fatalf(".cursorrules missing SSOT guardrail")
	}

	skill, err := os.ReadFile(filepath.Join(root, ".claude", "skills", "ratchet.md"))
	if err != nil {
		t.Fatalf("read claude skill: %v", err)
	}
	if !strings.Contains(string(skill), "ratchet check") {
		t.Fatalf("skill missing check command: %s", skill)
	}
}
