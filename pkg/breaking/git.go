package breaking

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fow830/ratchet/pkg/tokens"
)

// DiffAgainstGit compares current config file to git HEAD version.
func DiffAgainstGit(ctx context.Context, root, configPath string) ([]Change, error) {
	rel, err := filepath.Rel(root, configPath)
	if err != nil {
		rel = filepath.Base(configPath)
	}
	rel = filepath.ToSlash(rel)
	cmd := exec.CommandContext(ctx, "git", "-C", root, "show", "HEAD:"+rel)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git show HEAD:%s: %w\n%s", rel, err, out)
	}
	var old tokens.Config
	if err := json.Unmarshal(out, &old); err != nil {
		return nil, fmt.Errorf("parse HEAD config: %w", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cur tokens.Config
	if err := json.Unmarshal(data, &cur); err != nil {
		return nil, err
	}
	return DiffConfig(old, cur), nil
}
