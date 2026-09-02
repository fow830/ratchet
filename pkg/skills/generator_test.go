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
	cfg := tokens.DefaultConfig("example.com/app")
	g := skills.NewGenerator(cfg)
	if err := g.Generate(root); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	cursorRules, err := os.ReadFile(filepath.Join(root, tokens.CursorRules))
	if err != nil {
		t.Fatalf("read %s: %v", tokens.CursorRules, err)
	}
	body := string(cursorRules)
	if !strings.Contains(body, "Zero Architectural Regression") {
		t.Fatalf("%s missing mission: %s", tokens.CursorRules, body)
	}
	skill, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(tokens.ClaudeSkillRel)))
	if err != nil {
		t.Fatalf("read claude skill: %v", err)
	}
	if !strings.Contains(string(skill), tokens.ToolName+" check") {
		t.Fatalf("skill missing check command: %s", skill)
	}
}

func TestGenerator_Deterministic(t *testing.T) {
	cfg := tokens.DefaultConfig("example.com/app")
	// Shuffle-prone map iteration must still yield identical output.
	cfg.AllowedEdges = map[string][]string{
		tokens.LayerDelivery: {tokens.LayerDomain, tokens.LayerUsecase},
		tokens.LayerDomain:   {},
		tokens.LayerUsecase:  {tokens.LayerDomain},
	}
	g := skills.NewGenerator(cfg)
	a1, b1 := g.CursorRules(), g.ClaudeSkill()
	a2, b2 := g.CursorRules(), g.ClaudeSkill()
	if a1 != a2 || b1 != b2 {
		t.Fatal("generator output must be byte-identical across runs")
	}

	root := t.TempDir()
	if err := g.Generate(root); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(root, tokens.ClaudeSkillRel))
	if err := g.Generate(root); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(root, tokens.ClaudeSkillRel))
	if string(first) != string(second) {
		t.Fatal("repeated Generate must not change skill file")
	}
}

func TestGenerator_EscapesSpecialChars(t *testing.T) {
	cfg := tokens.DefaultConfig("example.com/mod`evil<script>")
	g := skills.NewGenerator(cfg)
	out := g.ClaudeSkill() + g.CursorRules()
	if strings.Contains(out, "`evil") || strings.Contains(out, "<script>") {
		t.Fatalf("special chars not escaped:\n%s", out)
	}
	if !strings.Contains(out, "mod'evil") {
		t.Fatalf("backtick should become quote:\n%s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("angles should be escaped:\n%s", out)
	}
}
