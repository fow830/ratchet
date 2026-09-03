package contracts_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ARCH-001: transport must not import repository; domain stays leaf.
func TestContract_ARCH_001(t *testing.T) {
	root := moduleRoot(t)
	for _, dir := range []string{
		"internal/transport",
		"internal/service",
		"internal/repository",
		"internal/domain",
	} {
		if _, err := os.Stat(filepath.Join(root, dir)); err != nil {
			t.Fatalf("layer path missing: %s", dir)
		}
	}

	fset := token.NewFileSet()
	err := filepath.Walk(filepath.Join(root, "internal/transport"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(ip, "/internal/repository") {
				t.Errorf("%s imports repository: %s", path, ip)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = filepath.Walk(filepath.Join(root, "internal/domain"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range []string{"/internal/transport", "/internal/service", "/internal/repository"} {
				if strings.Contains(ip, bad) {
					t.Errorf("domain imports %s via %s in %s", bad, ip, path)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
