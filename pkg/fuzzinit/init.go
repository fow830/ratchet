// Package fuzzinit scaffolds fuzz corpora for contract packages.
package fuzzinit

import (
	"os"
	"path/filepath"

	"github.com/fow830/ratchet/pkg/tokens"
)

// Init creates testdata/fuzz/FuzzSeed for the module root.
func Init(root string) (string, error) {
	dir := filepath.Join(root, filepath.FromSlash(tokens.FuzzCorpusRel))
	if err := os.MkdirAll(dir, tokens.FileModeDir); err != nil {
		return "", err
	}
	seed := filepath.Join(dir, tokens.FuzzSeedFileName)
	if _, err := os.Stat(seed); err == nil {
		return seed, nil
	}
	if err := os.WriteFile(seed, []byte("seed\n"), tokens.FileModeFile); err != nil {
		return "", err
	}
	return seed, nil
}
