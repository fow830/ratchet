// Package workspace discovers modules for monorepo ratchet check.
package workspace

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fow830/ratchet/pkg/tokens"
)

// Module is a check target in a workspace.
type Module struct {
	Root       string
	Module     string
	Config     tokens.Config
	ConfigPath string
}

// Discover returns ratchet-managed modules under root (go.work or single module).
func Discover(ctx context.Context, root string) ([]Module, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	gowork := filepath.Join(root, tokens.GoWorkFileName)
	if _, err := os.Stat(gowork); err == nil {
		return discoverGoWork(ctx, root)
	}
	return discoverSingle(ctx, root)
}

func discoverSingle(ctx context.Context, root string) ([]Module, error) {
	mod, cfg, cfgPath, err := loadModuleConfig(ctx, root)
	if err != nil {
		return nil, err
	}
	return []Module{{Root: root, Module: mod, Config: cfg, ConfigPath: cfgPath}}, nil
}

func discoverGoWork(ctx context.Context, root string) ([]Module, error) {
	paths, err := ModulesFromGoWork(root)
	if err != nil {
		return nil, err
	}
	var mods []Module
	for _, rel := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		dir := filepath.Join(root, filepath.FromSlash(rel))
		mod, cfg, cfgPath, err := loadModuleConfig(ctx, dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		mods = append(mods, Module{Root: dir, Module: mod, Config: cfg, ConfigPath: cfgPath})
	}
	if len(mods) == 0 {
		return discoverSingle(ctx, root)
	}
	return mods, nil
}

func loadModuleConfig(ctx context.Context, root string) (string, tokens.Config, string, error) {
	cfgPath := filepath.Join(root, tokens.ConfigFileName)
	if _, err := os.Stat(cfgPath); err != nil {
		return "", tokens.Config{}, "", err
	}
	cfg, err := tokens.Load(ctx, root)
	if err != nil {
		return "", tokens.Config{}, "", err
	}
	return cfg.Module, cfg, cfgPath, nil
}
