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
	writeFile(t, filepath.Join(normal, ".dotfile-in-normal-dir.md"), "also included")
	writeFile(t, filepath.Join(hidden, "excluded.md"), "excluded")
	writeFile(t, filepath.Join(skip, "excluded.md"), "excluded")

	specs, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2, not 1: the exclusion rule targets directory names during the
	// walk, never individual file names — a dot-prefixed .md FILE sitting
	// inside a non-excluded directory must still be picked up (fix-review,
	// glm-5.1:cloud + kimi-k2.6:cloud, independently).
	if len(specs) != 2 {
		t.Fatalf("want 2 specs (hidden/underscore dirs excluded, dotfile in normal dir included), got %d: %+v", len(specs), specs)
	}
	wantPaths := map[string]bool{
		filepath.Join(normal, "included.md"):               false,
		filepath.Join(normal, ".dotfile-in-normal-dir.md"): false,
	}
	for _, s := range specs {
		if _, ok := wantPaths[s.Path]; !ok {
			t.Fatalf("unexpected spec path %s in %+v", s.Path, specs)
		}
		wantPaths[s.Path] = true
	}
	for p, seen := range wantPaths {
		if !seen {
			t.Fatalf("want spec path %s in results, got %+v", p, specs)
		}
	}
}

// TestLoadUnderscorePrefixedRootIsNotExcluded mirrors
// TestLoadDotPrefixedRootIsNotExcluded for the "_" prefix (fix-review,
// glm-5.1:cloud): the exclusion applies only to descendant directories
// encountered during a walk, never to the explicitly-passed root itself,
// regardless of which excluded prefix the root's own name happens to use.
func TestLoadUnderscorePrefixedRootIsNotExcluded(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "_myspecs")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "spec.md"), "spec")

	specs, err := Load(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("want 1 spec (underscore-prefixed root must not be excluded), got %d: %+v", len(specs), specs)
	}
}

// TestLoadSingleFilePathInsideExcludedDir guards REQ-CHK-04's carve-out
// end-to-end (fix-review, glm-5.1:cloud): an explicitly-passed single
// file path is never filtered by the dot/underscore exclusion rule, even
// when that file happens to live inside a directory that WOULD be
// excluded if reached via a directory walk instead.
func TestLoadSingleFilePathInsideExcludedDir(t *testing.T) {
	dir := t.TempDir()
	skip := filepath.Join(dir, "_skip")
	if err := os.MkdirAll(skip, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(skip, "spec.md")
	writeFile(t, path, "spec")

	specs, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 || specs[0].Path != path {
		t.Fatalf("want 1 spec matching %s (single-file path bypasses dir exclusion), got %+v", path, specs)
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

// TestLoadSingleFileReadPermissionDenied guards Load's os.ReadFile error
// path for a single-file argument (spec.go ~28-30): os.Stat succeeds (it
// only needs traversal permission on parent directories, not read
// permission on the file itself), but the subsequent os.ReadFile must fail
// and propagate a non-nil error (issue #73).
//
// Skipped under root: permission bits don't apply to root, so this test
// would silently no-op there rather than exercise the error path — some CI
// environments run as root. Permissions are restored via t.Cleanup
// registered after t.TempDir(), so it runs (LIFO) before TempDir's own
// removal cleanup — an unreadable file could otherwise interfere with it.
func TestLoadSingleFileReadPermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file permission bits")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	writeFile(t, path, "# Spec")

	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatalf("restore chmod: %v", err)
		}
	})

	if _, err := Load(path); err == nil {
		t.Fatal("want error for a single file with read permission denied, got nil")
	}
}

// TestLoadWalkDirUnreadableSubdirectory guards WalkDir's callback receiving
// a non-nil err (spec.go ~38-40) — a subdirectory that can't be listed
// during the walk (e.g. permission denied) must propagate as Load's
// returned error (also exercising the outer `if err != nil` at ~59-61,
// which is this same code path's only way to fire).
//
// Skipped under root: see TestLoadSingleFileReadPermissionDenied. The
// subdirectory's permissions are restored before removal for the same
// reason — an unreadable subdirectory would otherwise break t.TempDir's own
// cleanup (RemoveAll needs to traverse into it).
func TestLoadWalkDirUnreadableSubdirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permission bits")
	}

	dir := t.TempDir()
	sub := filepath.Join(dir, "locked")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sub, "inside.md"), "inside")

	if err := os.Chmod(sub, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(sub, 0o755); err != nil {
			t.Fatalf("restore chmod: %v", err)
		}
	})

	if _, err := Load(dir); err == nil {
		t.Fatal("want error when a subdirectory can't be read during the walk, got nil")
	}
}

// TestLoadWalkDirFileReadPermissionDenied guards Load's os.ReadFile error
// path for a .md file discovered during a directory walk (spec.go ~53-55):
// the file is listed successfully (its directory stays readable) but
// reading its content must fail and propagate.
//
// Skipped under root: see TestLoadSingleFileReadPermissionDenied.
func TestLoadWalkDirFileReadPermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file permission bits")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "locked.md")
	writeFile(t, path, "# Locked")

	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatalf("restore chmod: %v", err)
		}
	})

	if _, err := Load(dir); err == nil {
		t.Fatal("want error when a discovered .md file can't be read, got nil")
	}
}
