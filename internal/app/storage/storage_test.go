package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadJSONRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	type doc struct {
		Version int      `json:"version"`
		Names   []string `json:"names"`
	}
	in := doc{Version: 1, Names: []string{"a", "b"}}
	if err := store.SaveJSON("demo.json", in); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "demo.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	var out doc
	if err := store.LoadJSON("demo.json", &out); err != nil {
		t.Fatal(err)
	}
	if out.Version != 1 || len(out.Names) != 2 || out.Names[0] != "a" {
		t.Fatalf("unexpected: %+v", out)
	}
}

func TestLoadJSONNotExist(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var dst map[string]any
	err = store.LoadJSON("missing.json", &dst)
	if !errors.Is(err, ErrNotExist) {
		t.Fatalf("want ErrNotExist, got %v", err)
	}
}

func TestSaveJSONAtomicNoTmpLeft(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveJSON("x.json", map[string]int{"n": 1}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if len(name) >= 4 && name[len(name)-4:] == ".tmp" {
			t.Fatalf("tmp left behind: %s", name)
		}
	}
}
