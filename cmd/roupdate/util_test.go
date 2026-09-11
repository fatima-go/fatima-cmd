package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileReplacesInsteadOfRewriting(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "new"), filepath.Join(dir, "saturn")
	if err := os.WriteFile(src, []byte("new build"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old build with stale attributes"), 0644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(dst)

	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("destination was rewritten in place; its old extended attributes would survive")
	}
	if b, _ := os.ReadFile(dst); string(b) != "new build" || after.Mode().Perm() != 0755 {
		t.Fatal(string(b), after.Mode())
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 2 {
		t.Fatal("temporary file left behind", entries)
	}
}
