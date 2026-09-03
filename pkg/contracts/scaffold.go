// Package contracts scaffolds architecture and feature contract tests.
package contracts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fow830/ratchet/pkg/tokens"
)

// ScaffoldOpts configures a new contract test file.
type ScaffoldOpts struct {
	ID       string
	Title    string
	Negative bool
}

// Scaffold writes <ContractsDirDefault>/<slug>_contract_test.go.
func Scaffold(root string, opts ScaffoldOpts) (string, error) {
	dir := filepath.Join(root, filepath.FromSlash(tokens.ContractsDirDefault))
	if err := os.MkdirAll(dir, tokens.FileModeDir); err != nil {
		return "", err
	}
	slug := strings.ToLower(strings.ReplaceAll(opts.ID, "-", "_"))
	name := slug + tokens.ContractTestSuffix
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("contract file already exists: %s", path)
	}
	body := scaffoldBody(opts)
	if err := os.WriteFile(path, []byte(body), tokens.FileModeFile); err != nil {
		return "", err
	}
	return path, nil
}

// ScaffoldSuite writes suite_test.go under dir if missing.
func ScaffoldSuite(dir string) (string, error) {
	path := filepath.Join(dir, tokens.ContractSuiteFile)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	body := fmt.Sprintf(`package contracts_test

import (
	"os"
	"path/filepath"
	"testing"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, %q)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal(%q + " not found")
		}
		dir = parent
	}
}
`, tokens.GoModFileName, tokens.GoModFileName)
	if err := os.WriteFile(path, []byte(body), tokens.FileModeFile); err != nil {
		return "", err
	}
	return path, nil
}

func scaffoldBody(opts ScaffoldOpts) string {
	testName := "TestContract_" + strings.ReplaceAll(opts.ID, "-", "_")
	comment := fmt.Sprintf("// %s: %s", opts.ID, opts.Title)
	msg := "RED: implement contract proof"
	if opts.Negative {
		msg = "RED: implement negative contract proof"
	}
	return fmt.Sprintf(`package contracts_test

import "testing"

%s
func %s(t *testing.T) {
	t.Fatal(%q)
}
`, comment, testName, msg)
}

// PackagePath returns the standard contracts directory relative to module root.
func PackagePath(root string) string {
	return filepath.Join(root, filepath.FromSlash(tokens.ContractsDirDefault))
}
