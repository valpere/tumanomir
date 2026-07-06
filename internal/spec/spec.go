// Package spec loads markdown specification documents for measurement.
package spec

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Spec is one loaded specification document.
type Spec struct {
	Path    string // filesystem path as passed to or discovered by Load
	Content []byte // raw file bytes, unparsed
}

// Load reads a single markdown file, or all *.md files under a directory
// (recursive). Implements REQ-CHK-04.
//
// Two distinct code paths, deliberately: a single-file argument is loaded
// exactly as given, with no exclusion-rule filtering applied at all (see
// isExcludedDir below) — an explicitly-named file is never second-guessed,
// even if it happens to live inside a directory that would be skipped
// during a directory walk. A directory argument is walked recursively,
// collecting every *.md file found while pruning excluded subdirectories.
func Load(path string) ([]Spec, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return []Spec{{Path: path, Content: content}}, nil
	}

	root := filepath.Clean(path)

	var specs []Spec
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		// A non-nil err here means WalkDir itself couldn't stat/read this
		// entry (e.g. permission denied) — propagate rather than skip
		// silently, since a spec that failed to load must not be
		// mistaken for a spec that simply doesn't exist.
		if err != nil {
			return err
		}
		if d.IsDir() {
			// The root directory itself is never excluded, even if its own
			// name is dot- or underscore-prefixed — only descendants are.
			if p != root && isExcludedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		specs = append(specs, Spec{Path: p, Content: content})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(specs) == 0 {
		// A directory with zero *.md files anywhere under it (after
		// exclusions) is almost certainly a user error (wrong path,
		// wrong extension) rather than a legitimate "empty corpus" —
		// fail loudly instead of silently proceeding with a zero-spec
		// aggregate result.
		return nil, fmt.Errorf("no *.md files under %s", path)
	}
	return specs, nil
}

// isExcludedDir reports whether a directory name marks tooling/scratch/
// archival content that a spec walk must not recurse into (REQ-CHK-04).
func isExcludedDir(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}
