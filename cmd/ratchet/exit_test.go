package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{nil, exitOK},
		{violationErr(fmt.Errorf("drift")), exitViolation},
		{systemErr(fmt.Errorf("boom")), exitSystem},
		{fmt.Errorf("plain"), exitSystem},
		{fmt.Errorf("wrap: %w", violationErr(fmt.Errorf("x"))), exitViolation},
	}
	for _, tc := range tests {
		if got := exitCode(tc.err); got != tc.want {
			t.Fatalf("exitCode(%v)=%d want %d", tc.err, got, tc.want)
		}
	}
	var ce *codedError
	if !errors.As(violationErr(fmt.Errorf("x")), &ce) {
		t.Fatal("expected As")
	}
}

func TestCLI_MissingConfig(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	root := newRootCommand()
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"check", "--format=json"})
	err = root.Execute()
	if err == nil {
		t.Fatalf("expected error without %s", tokens.ConfigFileName)
	}
	if exitCode(err) != exitSystem {
		t.Fatalf("exit=%d want %d (%v)", exitCode(err), exitSystem, err)
	}
	if !strings.Contains(err.Error(), tokens.ConfigFileName) && !strings.Contains(err.Error(), "load") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestCLI_InitHooksWithoutGit(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	root := newRootCommand()
	root.SetArgs([]string{"init-hooks"})
	err = root.Execute()
	if err == nil {
		t.Fatalf("expected error without %s", tokens.GitDir)
	}
	if exitCode(err) != exitSystem {
		t.Fatalf("exit=%d want %d", exitCode(err), exitSystem)
	}
	if !strings.Contains(err.Error(), tokens.GitDir) {
		t.Fatalf("err=%v", err)
	}
}

func TestCLI_ConfigPermissionDenied(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, tokens.ConfigFileName)
	if err := os.WriteFile(cfgPath, []byte(`{"module":"x"}`), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(cfgPath, 0o644) })

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	root := newRootCommand()
	root.SetArgs([]string{"check", "--format=text"})
	err = root.Execute()
	if err == nil {
		t.Fatal("expected permission error")
	}
	if exitCode(err) != exitSystem {
		t.Fatalf("exit=%d want %d (%v)", exitCode(err), exitSystem, err)
	}
}

func TestCLI_CompletionGeneratesShellScripts(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		root := newRootCommand()
		out := &bytes.Buffer{}
		root.SetOut(out)
		root.SetArgs([]string{"completion", shell})
		if err := root.Execute(); err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		if out.Len() < 20 {
			t.Fatalf("%s completion too short", shell)
		}
	}
}
