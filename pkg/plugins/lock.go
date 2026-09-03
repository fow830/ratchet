package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fow830/ratchet/pkg/tokens"
)

// LockFileName stores WASM plugin hashes.
const LockFileName = tokens.PluginsLockFileName

type pluginLock struct {
	Version int               `json:"version"`
	Files   map[string]string `json:"files"`
}

// LockPlugins writes SHA-256 digests for plugin paths.
func LockPlugins(ctx context.Context, root string, refs []tokens.PluginRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	files := map[string]string{}
	for _, p := range refs {
		sum, err := hashRel(root, p.Path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(p.Path)] = sum
	}
	lf := pluginLock{Version: tokens.LockVersion, Files: files}
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(root, LockFileName), data, tokens.FileModeFile)
}

// VerifyLock checks plugin files against the plugins lock when present.
func VerifyLock(ctx context.Context, root string, refs []tokens.PluginRef) error {
	path := filepath.Join(root, LockFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LockPlugins(ctx, root, refs)
		}
		return err
	}
	var lf pluginLock
	if err := json.Unmarshal(data, &lf); err != nil {
		return err
	}
	for _, p := range refs {
		rel := filepath.ToSlash(p.Path)
		expected, ok := lf.Files[rel]
		if !ok {
			return fmt.Errorf("plugin %s not in %s", rel, LockFileName)
		}
		actual, err := hashRel(root, p.Path)
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("plugin drift %s expected=%s actual=%s", rel, expected, actual)
		}
	}
	return nil
}

func hashRel(root, rel string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
