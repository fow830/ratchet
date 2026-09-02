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
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	first := strings.SplitN(string(data), "\n", 2)[0]
	want := "module " + tokens.ModulePath
	if first != want {
		t.Fatalf("go.mod = %q, tokens.ModulePath wants %q", first, want)
	}
}

func TestIdentityTokens(t *testing.T) {
	if tokens.ToolName == "" || tokens.BinaryRel != "bin/"+tokens.ToolName {
		t.Fatalf("tool=%q binary=%q", tokens.ToolName, tokens.BinaryRel)
	}
	if tokens.ModuleHTTPSURL() != "https://"+tokens.ModulePath {
		t.Fatalf("url=%s", tokens.ModuleHTTPSURL())
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
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found from %s", wd)
		}
		dir = parent
	}
}
