package tokens_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
)

func TestModulePath_MatchesGoMod(t *testing.T) {
	root := findGoMod(t)
	data, err := os.ReadFile(filepath.Join(root, tokens.GoModFileName))
	if err != nil {
		t.Fatal(err)
	}
	first := strings.SplitN(string(data), "\n", 2)[0]
	want := "module " + tokens.ModulePath
	if first != want {
		t.Fatalf("%s = %q, tokens.ModulePath wants %q", tokens.GoModFileName, first, want)
	}
}

func TestIdentityTokens_Derived(t *testing.T) {
	if tokens.ToolName == "" {
		t.Fatal("ToolName empty")
	}
	cases := map[string]string{
		"ConfigFileName":      tokens.ToolName + ".json",
		"LockFileName":        tokens.ToolName + ".lock",
		"BenchFileName":       tokens.ToolName + ".bench",
		"PluginsLockFileName": tokens.ToolName + ".plugins.lock",
		"SchemaFileName":      tokens.ToolName + ".schema.json",
		"SchemaRel":           "schema/" + tokens.ToolName + ".schema.json",
		"BinaryRel":           "bin/" + tokens.ToolName,
		"CmdRel":              "cmd/" + tokens.ToolName,
		"ClaudeSkillRel":      ".claude/skills/" + tokens.ToolName + ".md",
		"ModulePath":          "github.com/fow830/" + tokens.ToolName,
		"DefaultBranch":       "main",
		"PreCommitRel":        tokens.GitDir + "/hooks/pre-commit",
		"CommitMsgRel":        tokens.GitDir + "/hooks/commit-msg",
		"ContractsDirDefault": "tests/contracts",
		"FuzzCorpusRel":       tokens.DirTestdata + "/fuzz/FuzzSeed",
		"BufAgainstGit":       tokens.GitDir + "#branch=main",
		"GoWorkFileName":      "go.work",
		"CPUProfileFile":      "cpu.pprof",
		"InstallStaticcheck":  "honnef.co/go/tools/cmd/staticcheck@latest",
		"InstallGovulncheck":  "golang.org/x/vuln/cmd/govulncheck@latest",
		"FitnessPkgRel":       "pkg/fitness",
		"DockerGoImage":       "golang:1.22-alpine",
	}
	got := map[string]string{
		"ConfigFileName":      tokens.ConfigFileName,
		"LockFileName":        tokens.LockFileName,
		"BenchFileName":       tokens.BenchFileName,
		"PluginsLockFileName": tokens.PluginsLockFileName,
		"SchemaFileName":      tokens.SchemaFileName,
		"SchemaRel":           tokens.SchemaRel,
		"BinaryRel":           tokens.BinaryRel,
		"CmdRel":              tokens.CmdRel,
		"ClaudeSkillRel":      tokens.ClaudeSkillRel,
		"ModulePath":          tokens.ModulePath,
		"DefaultBranch":       tokens.DefaultBranch,
		"PreCommitRel":        tokens.PreCommitRel,
		"CommitMsgRel":        tokens.CommitMsgRel,
		"ContractsDirDefault": tokens.ContractsDirDefault,
		"FuzzCorpusRel":       tokens.FuzzCorpusRel,
		"BufAgainstGit":       tokens.BufAgainstGit(),
		"GoWorkFileName":      tokens.GoWorkFileName,
		"CPUProfileFile":      tokens.CPUProfileFile,
		"ContractTestSuffix":  tokens.ContractTestSuffix,
		"InstallStaticcheck":  tokens.InstallStaticcheck,
		"InstallGovulncheck":  tokens.InstallGovulncheck,
		"FitnessPkgRel":       tokens.FitnessPkgRel,
		"DockerGoImage":       tokens.DockerGoImage,
	}
	for k, want := range cases {
		if got[k] != want {
			t.Fatalf("%s=%q want %q", k, got[k], want)
		}
	}
	if tokens.ModuleHTTPSURL() != "https://"+tokens.ModulePath {
		t.Fatalf("url=%s", tokens.ModuleHTTPSURL())
	}
	if tokens.LockVersion < 1 {
		t.Fatalf("LockVersion=%d", tokens.LockVersion)
	}
}

func TestDefaultConfig_UsesLayerTokens(t *testing.T) {
	cfg := tokens.DefaultConfig("example.com/app")
	if cfg.Layers[tokens.SuffixDomain] != tokens.LayerDomain {
		t.Fatalf("layers=%v", cfg.Layers)
	}
	if got := cfg.AllowedEdges[tokens.LayerUsecase]; len(got) != 1 || got[0] != tokens.LayerDomain {
		t.Fatalf("usecase edges=%v", got)
	}
}

func findGoMod(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, tokens.GoModFileName)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("%s not found from %s", tokens.GoModFileName, wd)
		}
		dir = parent
	}
}
