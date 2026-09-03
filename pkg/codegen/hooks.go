// Package codegen provides optional Go-ecosystem generator drift hooks.
package codegen

import (
	"os"
	"path/filepath"

	"github.com/fow830/ratchet/pkg/tokens"
)

const (
	ToolSQLC    = "sqlc"
	ToolBuf     = "buf"
	ToolOpenAPI = "openapi"
	ToolCUE     = "cue"
)

// IsConfigured reports whether a codegen hook is enabled in config.
func IsConfigured(cfg tokens.Config, tool string) bool {
	switch tool {
	case ToolSQLC:
		return cfg.Codegen.SQLCPath != "" && fileExists(cfg.Codegen.SQLCPath)
	case ToolBuf:
		return cfg.Codegen.BufPath != "" && fileExists(cfg.Codegen.BufPath)
	case ToolOpenAPI:
		return cfg.Codegen.OpenAPIPath != ""
	case ToolCUE:
		return cfg.Codegen.CUEPath != ""
	default:
		return false
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// RootPath joins generator config path under module root.
func RootPath(root, rel string) string {
	if rel == "" {
		return ""
	}
	return filepath.Join(root, filepath.FromSlash(rel))
}
