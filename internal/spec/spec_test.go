package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "requirements.md")
	if err := os.WriteFile(path, []byte("# Spec"), 0o644); err != nil {
		t.Fatal(err)
	}

	specs, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 || specs[0].Path != path || string(specs[0].Content) != "# Spec" {
		t.Fatalf("want 1 spec matching %s, got %+v", path, specs)
	}
}

func TestLoadDirectoryRecursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "a.md"), "A")
	writeFile(t, filepath.Join(sub, "b.md"), "B")
	writeFile(t, filepath.Join(dir, "ignore.txt"), "not markdown")

	specs, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("want 2 *.md specs (non-.md excluded), got %d: %+v", len(specs), specs)
	}
}

func TestLoadDirectoryNoMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "notes.txt"), "not markdown")

	if _, err := Load(dir); err == nil {
		t.Fatal("want error for directory with no *.md files, got nil")
	}
}

func TestLoadNonExistentPath(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.md")); err == nil {
		t.Fatal("want error for non-existent path, got nil")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
