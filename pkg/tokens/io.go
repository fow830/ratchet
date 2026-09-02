package tokens

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// Load reads Config from root/ratchet.json.
func Load(root string) (Config, error) {
	path := filepath.Join(root, ConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("load %s: %w", ConfigFileName, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", ConfigFileName, err)
	}
	return cfg, nil
}

// Save writes Config to root/ratchet.json (deterministic JSON).
func Save(root string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(root, ConfigFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", ConfigFileName, err)
	}
	return nil
}

// ModuleFromGoMod parses the module path from root/go.mod.
func ModuleFromGoMod(root string) (string, error) {
	path := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return "", fmt.Errorf("parse go.mod: %w", err)
	}
	if f.Module == nil || f.Module.Mod.Path == "" {
		return "", fmt.Errorf("go.mod: missing module statement")
	}
	return f.Module.Mod.Path, nil
}
