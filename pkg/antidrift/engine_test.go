package antidrift_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fow830/ratchet/pkg/antidrift"
	"github.com/fow830/ratchet/pkg/tokens"
)

func TestEngine_LockAndVerifyOK(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	path := filepath.Join(root, "contract.txt")
	if err := os.WriteFile(path, []byte("stable"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	writeConfig(t, root, []string{"contract.txt"})

	eng := antidrift.New(root)
	if err := eng.Lock(ctx, []string{"contract.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	diff, err := eng.Verify(ctx)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !diff.OK() {
		t.Fatalf("expected OK, got %#v", diff)
	}
}

func TestEngine_VerifyDetectsDrift(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	path := filepath.Join(root, "contract.txt")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	writeConfig(t, root, []string{"contract.txt"})
	eng := antidrift.New(root)
	if err := eng.Lock(ctx, []string{"contract.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if err := os.WriteFile(path, []byte("v2-drifted"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	diff, err := eng.Verify(ctx)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].Path != "contract.txt" {
		t.Fatalf("unexpected changed: %#v", diff.Changed)
	}
}

func TestEngine_VerifyDetectsMissing(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	path := filepath.Join(root, "contract.txt")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	writeConfig(t, root, []string{"contract.txt"})
	eng := antidrift.New(root)
	if err := eng.Lock(ctx, []string{"contract.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	diff, err := eng.Verify(ctx)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(diff.Missing) != 1 || diff.Missing[0] != "contract.txt" {
		t.Fatalf("expected missing, got %#v", diff)
	}
}

func TestEngine_VerifyDetectsExtra(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	_ = os.WriteFile(filepath.Join(root, "locked.txt"), []byte("a"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "extra.txt"), []byte("b"), 0o644)
	writeConfig(t, root, []string{"locked.txt", "extra.txt"})
	eng := antidrift.New(root)
	if err := eng.Lock(ctx, []string{"locked.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	diff, err := eng.Verify(ctx)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(diff.Extra) != 1 || diff.Extra[0] != "extra.txt" {
		t.Fatalf("expected Extra=[extra.txt], got %#v", diff)
	}
}

func TestEngine_VerifyIgnoresUndeclaredNeighbors(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	_ = os.WriteFile(filepath.Join(root, "locked.txt"), []byte("a"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "README.md"), []byte("noise"), 0o644)
	writeConfig(t, root, []string{"locked.txt"})
	eng := antidrift.New(root)
	if err := eng.Lock(ctx, []string{"locked.txt"}); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	diff, err := eng.Verify(ctx)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !diff.OK() {
		t.Fatalf("unexpected: %#v", diff)
	}
}

func TestEngine_CrashScenarios(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
		want  string // substring in error
	}{
		{
			name: "missing_lock",
			setup: func(t *testing.T, root string) {
				writeConfig(t, root, []string{"a.txt"})
			},
			want: "read lock",
		},
		{
			name: "invalid_json_lock",
			setup: func(t *testing.T, root string) {
				writeConfig(t, root, nil)
				if err := os.WriteFile(filepath.Join(root, tokens.LockFileName), []byte("{nope"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "parse lock",
		},
		{
			name: "yaml_masquerading_as_lock",
			setup: func(t *testing.T, root string) {
				writeConfig(t, root, nil)
				if err := os.WriteFile(filepath.Join(root, tokens.LockFileName), []byte("files:\n  a: b\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "parse lock",
		},
		{
			name: "empty_lock_object",
			setup: func(t *testing.T, root string) {
				writeConfig(t, root, []string{})
				if err := os.WriteFile(filepath.Join(root, tokens.LockFileName), []byte("{}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "", // should succeed with empty files map
		},
		{
			name: "nil_contract_files_slice",
			setup: func(t *testing.T, root string) {
				cfg := tokens.DefaultConfig("example.com/app")
				cfg.ContractFiles = nil
				data, _ := json.Marshal(cfg)
				_ = os.WriteFile(filepath.Join(root, tokens.ConfigFileName), data, 0o644)
				_ = os.WriteFile(filepath.Join(root, "c.txt"), []byte("x"), 0o644)
				eng := antidrift.New(root)
				if err := eng.Lock(context.Background(), []string{"c.txt"}); err != nil {
					t.Fatal(err)
				}
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			tc.setup(t, root)
			_, err := antidrift.New(root).Verify(context.Background())
			if tc.want == "" {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v want substring %q", err, tc.want)
			}
		})
	}
}

func TestEngine_WithMockFS(t *testing.T) {
	ctx := context.Background()
	cfgBytes, err := json.Marshal(tokens.Config{Module: "m", ContractFiles: []string{"a.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	mem := &memFS{files: map[string][]byte{
		"/repo/ratchet.lock": []byte(`{"version":1,"files":{"a.txt":"deadbeef"}}` + "\n"),
		"/repo/ratchet.json": append(cfgBytes, '\n'),
	}}
	eng := &antidrift.Engine{Root: "/repo", FS: mem}
	diff, err := eng.Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Missing) != 1 || diff.Missing[0] != "a.txt" {
		t.Fatalf("mock missing: %#v", diff)
	}
}

func TestEngine_CanceledContext(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "c.txt"), []byte("x"), 0o644)
	writeConfig(t, root, []string{"c.txt"})
	eng := antidrift.New(root)
	_ = eng.Lock(context.Background(), []string{"c.txt"})
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	_, err := eng.Verify(ctx)
	if err == nil {
		t.Fatal("expected timeout/cancel")
	}
}

func writeConfig(t *testing.T, root string, contracts []string) {
	t.Helper()
	cfg := tokens.DefaultConfig("example.com/app")
	cfg.ContractFiles = contracts
	if err := tokens.Save(context.Background(), root, cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

type memFS struct {
	files map[string][]byte
}

func (m *memFS) ReadFile(name string) ([]byte, error) {
	b, ok := m.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), b...), nil
}

func (m *memFS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	m.files[name] = append([]byte(nil), data...)
	return nil
}

func (m *memFS) Stat(name string) (fs.FileInfo, error) {
	b, ok := m.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return fakeInfo{name: filepath.Base(name), size: int64(len(b))}, nil
}

type fakeInfo struct {
	name string
	size int64
}

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return f.size }
func (f fakeInfo) Mode() fs.FileMode  { return 0o644 }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return false }
func (f fakeInfo) Sys() any           { return nil }
