package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// Load reads Config from root/ratchet.json using a streaming JSON decoder.
func Load(ctx context.Context, root string) (Config, error) {
	return LoadFile(ctx, filepath.Join(root, ConfigFileName))
}

// LoadFile streams and parses a ratchet config JSON file.
func LoadFile(ctx context.Context, path string) (Config, error) {
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("load %s: %w", filepath.Base(path), err)
	}
	defer f.Close()

	cfg, err := Decode(ctx, f)
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return cfg, nil
}

// Decode streams JSON config from r with cancellation support.
func Decode(ctx context.Context, r io.Reader) (Config, error) {
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}
	dec := json.NewDecoder(ctxReader{ctx: ctx, r: r})
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, err
	}
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes Config to root/ratchet.json (deterministic JSON).
func Save(ctx context.Context, root string, cfg Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return SaveFile(ctx, filepath.Join(root, ConfigFileName), cfg)
}

// SaveFile writes Config to path.
func SaveFile(ctx context.Context, path string, cfg Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}

// ModuleFromGoMod parses the module path from root/go.mod.
func ModuleFromGoMod(ctx context.Context, root string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path := filepath.Join(root, "go.mod")
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(ctxReader{ctx: ctx, r: f})
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	mf, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return "", fmt.Errorf("parse go.mod: %w", err)
	}
	if mf.Module == nil || mf.Module.Mod.Path == "" {
		return "", fmt.Errorf("go.mod: missing module statement")
	}
	return mf.Module.Mod.Path, nil
}

// ctxReader checks context between reads for cancel/timeout.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
