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

	var specs []Spec
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
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
		return nil, fmt.Errorf("no *.md files under %s", path)
	}
	return specs, nil
}
