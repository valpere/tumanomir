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

func TestLoadSkipsDotAndUnderscorePrefixedDirs(t *testing.T) {
	dir := t.TempDir()

	normal := filepath.Join(dir, "normal")
	hidden := filepath.Join(dir, ".hidden")
	skip := filepath.Join(dir, "_skip")
	for _, d := range []string{normal, hidden, skip} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(normal, "included.md"), "included")
	writeFile(t, filepath.Join(hidden, "excluded.md"), "excluded")
	writeFile(t, filepath.Join(skip, "excluded.md"), "excluded")

	specs, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("want 1 spec (hidden/underscore dirs excluded), got %d: %+v", len(specs), specs)
	}
	want := filepath.Join(normal, "included.md")
	if specs[0].Path != want {
		t.Fatalf("want spec path %s, got %s", want, specs[0].Path)
	}
}

func TestLoadDotPrefixedRootIsNotExcluded(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".myspecs")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "spec.md"), "spec")

	specs, err := Load(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("want 1 spec (dot-prefixed root must not be excluded), got %d: %+v", len(specs), specs)
	}
}

func TestLoadSingleDotfileIsNotExcluded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".hidden.md")
	if err := os.WriteFile(path, []byte("# Hidden"), 0o644); err != nil {
		t.Fatal(err)
	}

	specs, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 || specs[0].Path != path || string(specs[0].Content) != "# Hidden" {
		t.Fatalf("want 1 spec matching %s, got %+v", path, specs)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
