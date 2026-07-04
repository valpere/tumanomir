package internal

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// TestNoNetworkImports enforces REQ-CHK-05: internal/metrics and internal/spec
// must stay network-free, including transitively. It shells out to `go list`
// to inspect the full dependency graph rather than scanning imports directly,
// since the realistic risk is an indirect network import pulled in through a
// future shared helper, not a literal `import "net/http"` in these packages.
func TestNoNetworkImports(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH, skipping network-import check")
	}

	pkgPatterns := []string{
		"github.com/valpere/tumanomir/internal/metrics/...",
		"github.com/valpere/tumanomir/internal/spec/...",
	}

	for _, pattern := range pkgPatterns {
		cmd := exec.CommandContext(t.Context(), "go", "list", "-f", "{{.Deps}}", pattern)
		out, err := cmd.Output()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				t.Fatalf("go list -f {{.Deps}} %s: %v\nstderr:\n%s", pattern, err, exitErr.Stderr)
			}
			t.Fatalf("go list -f {{.Deps}} %s: %v", pattern, err)
		}

		for _, dep := range parseDeps(string(out)) {
			if dep == "net" || strings.HasPrefix(dep, "net/") {
				t.Errorf("%s: found network dependency %q; internal/metrics and internal/spec must stay network-free (REQ-CHK-05)", pattern, dep)
			}
		}
	}
}

// parseDeps parses the output of `go list -f '{{.Deps}}'`, which is a
// Go-syntax slice literal such as "[fmt strings ...]", one line per package
// listed by the pattern.
func parseDeps(out string) []string {
	var deps []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "[")
		line = strings.TrimSuffix(line, "]")
		if line == "" {
			continue
		}
		deps = append(deps, strings.Fields(line)...)
	}
	return deps
}

func TestParseDeps(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "[]\n", nil},
		{"single", "[fmt]\n", []string{"fmt"}},
		{"multiple", "[fmt strings errors]\n", []string{"fmt", "strings", "errors"}},
		{"multi-line from ... pattern", "[fmt strings]\n[errors net/http]\n", []string{"fmt", "strings", "errors", "net/http"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDeps(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("parseDeps(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i, dep := range tt.want {
				if got[i] != dep {
					t.Fatalf("parseDeps(%q)[%d] = %q, want %q", tt.in, i, got[i], dep)
				}
			}
		})
	}
}
