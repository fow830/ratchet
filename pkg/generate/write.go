package generate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fow830/ratchet/pkg/tokens"
)

// WriteAll writes all generated outputs under root.
func WriteAll(ctx context.Context, root string, cfg tokens.Config) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var written []string
	for _, out := range Outputs(cfg) {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		path := filepath.Join(root, filepath.FromSlash(out.Path))
		if err := os.MkdirAll(filepath.Dir(path), tokens.FileModeDir); err != nil {
			return written, err
		}
		if err := os.WriteFile(path, []byte(out.Body), tokens.FileModeFile); err != nil {
			return written, fmt.Errorf("write %s: %w", out.Path, err)
		}
		written = append(written, out.Path)
	}
	return written, nil
}
