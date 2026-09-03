package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fow830/ratchet/pkg/generate"
	"github.com/fow830/ratchet/pkg/tokens"
)

func main() {
	os.Exit(run())
}

func run() int {
	root, err := findModuleRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tokensgen:", err)
		return 2
	}
	ctx := context.Background()
	cfg, err := tokens.Load(ctx, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tokensgen:", err)
		return 2
	}
	written, err := generate.WriteAll(ctx, root, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tokensgen:", err)
		return 2
	}
	fmt.Printf("tokensgen: wrote %v\n", written)
	return 0
}

func findModuleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, tokens.GoModFileName)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%s not found", tokens.GoModFileName)
		}
		dir = parent
	}
}
