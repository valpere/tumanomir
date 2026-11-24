package samples

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
	"github.com/valpere/tumanomir/internal/report"
)

func fixture() report.MeasureResult {
	return report.MeasureResult{
		Config: internal.InstrumentConfig{Backend: "ollama", Model: "m", Samples: 3, NumCtx: 8, NumPredict: 8},
		Attempts: []report.SampleRecord{
			{Kind: report.SampleDiscard, Slot: 0, Attempt: 0, Data: []byte("not go")},
			{Kind: report.SampleAccepted, Slot: 0, Attempt: 1, Data: []byte("package a\n")},
			{Kind: report.SampleTimeout, Slot: 1, Attempt: 0, Data: []byte("context deadline exceeded")},
			{Kind: report.SampleAccepted, Slot: 1, Attempt: 1, Data: []byte("package b\n")},
		},
	}
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

func TestWriteLayout(t *testing.T) {
	root := t.TempDir()
	mr := fixture()
	got, err := Write(root, []byte("spec"), mr)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if filepath.Dir(got) != root {
		t.Fatalf("run dir %q not directly under %q", got, root)
	}
	want := []string{"00.go", "01.go", "instrument.json", "s00-a0.discard.txt", "s01-a0.timeout.txt"}
	if n := names(t, got); !reflect.DeepEqual(n, want) {
		t.Fatalf("files = %v, want %v", n, want)
	}
	b, _ := os.ReadFile(filepath.Join(got, "01.go"))
	if string(b) != "package b\n" {
		t.Fatalf("01.go = %q, want verbatim accepted bytes", b)
	}
	b, _ = os.ReadFile(filepath.Join(got, "s00-a0.discard.txt"))
	if string(b) != "not go" {
		t.Fatalf("discard = %q", b)
	}
	b, _ = os.ReadFile(filepath.Join(got, "instrument.json"))
	var cfg internal.InstrumentConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("instrument.json: %v", err)
	}
	if !reflect.DeepEqual(cfg, mr.Config) {
		t.Fatalf("instrument.json = %+v, want %+v", cfg, mr.Config)
	}
}

func TestWritePaddingWidensWithSamples(t *testing.T) {
	mr := report.MeasureResult{
		Config:   internal.InstrumentConfig{Samples: 101},
		Attempts: []report.SampleRecord{{Kind: report.SampleAccepted, Slot: 100, Data: []byte("package a\n")}},
	}
	got, err := Write(t.TempDir(), nil, mr)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n := names(t, got); !reflect.DeepEqual(n, []string{"000.go", "instrument.json"}) {
		t.Fatalf("files = %v", n)
	}
}

func TestWriteRerunNeverMixes(t *testing.T) {
	root := t.TempDir()
	a, err := Write(root, []byte("spec"), fixture())
	if err != nil {
		t.Fatalf("Write 1: %v", err)
	}
	b, err := Write(root, []byte("spec"), fixture())
	if err != nil {
		t.Fatalf("Write 2: %v", err)
	}
	if a == b {
		t.Fatalf("rerun reused %q", a)
	}
	if filepath.Base(a)[:21] != filepath.Base(b)[:21] || !strings.Contains(filepath.Base(a), "-") {
		t.Fatalf("run dirs %q / %q should share the spec+instrument prefix", a, b)
	}
	if len(names(t, a)) != 5 || len(names(t, b)) != 5 {
		t.Fatalf("each run dir must hold exactly its own 5 files")
	}
}

func TestWriteCreatesMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nested", "keep")
	if _, err := Write(root, nil, fixture()); err != nil {
		t.Fatalf("Write into missing root: %v", err)
	}
}

func TestWriteUnwritableRootErrors(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(filepath.Join(blocker, "sub"), nil, fixture()); err == nil {
		t.Fatal("want error when root is under a regular file")
	}
}

// A failed write must not leave a partial run dir that looks like a
// complete audit trail; an unknown Kind is such a failure.
func TestWriteFailureRemovesPartialRun(t *testing.T) {
	root := t.TempDir()
	mr := fixture()
	mr.Attempts = append(mr.Attempts, report.SampleRecord{Kind: "bogus"})
	if _, err := Write(root, []byte("spec"), mr); err == nil {
		t.Fatal("want error for unknown SampleKind")
	}
	if got := names(t, root); len(got) != 0 {
		t.Fatalf("partial run left behind: %v", got)
	}
}
