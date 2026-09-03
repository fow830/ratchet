package fuzzinit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fow830/ratchet/pkg/fuzzinit"
)

func TestInitCreatesSeed(t *testing.T) {
	root := t.TempDir()
	path, err := fuzzinit.Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "seed\n" {
		t.Fatalf("body=%q", data)
	}
}

func TestInitIdempotent(t *testing.T) {
	root := t.TempDir()
	p1, err := fuzzinit.Init(root)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := fuzzinit.Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Fatalf("%s != %s", p1, p2)
	}
	if filepath.Base(p1) != "seed.txt" {
		t.Fatalf("path=%s", p1)
	}
}
