package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
	"github.com/valpere/tumanomir/internal/spec"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything written to it.
func captureStdout(t *testing.T, fn func() int) (string, int) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	// Drain the pipe concurrently: the OS pipe buffer is finite (~64KB on
	// Linux), so reading only after fn() returns would deadlock once
	// output exceeds it.
	readDone := make(chan struct{})
	var buf bytes.Buffer
	var copyErr error
	go func() {
		_, copyErr = io.Copy(&buf, r)
		close(readDone)
	}()

	code := fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	<-readDone
	if copyErr != nil {
		t.Fatalf("read pipe: %v", copyErr)
	}
	return buf.String(), code
}

func TestRunCheckNoRequirementsIsSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nospec.md")
	if err := os.WriteFile(path, []byte("# No requirements here\n\nJust prose with no tags.\n"), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}

	out, code := captureStdout(t, func() int { return runCheck([]string{path}) })

	if code != 0 {
		t.Fatalf("want exit code 0 for a spec with zero requirements, got %d\noutput:\n%s", code, out)
	}
	want := "  K_drift:  —     [n/a]    (no [REQ-*] tags found)\n"
	if !strings.Contains(out, want) {
		t.Fatalf("want K_drift line %q, got output:\n%s", want, out)
	}
	if strings.Contains(out, "K_drift:  0.00  [ok]") {
		t.Fatalf("zero-requirement spec must not render as a numeric ok pass, got:\n%s", out)
	}
}

func TestRunCheckWithRequirementsIsNumeric(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	spec := "1. [REQ-X-01] Do the thing.\n   -> [FUN-X-01] DoThing()\n"
	if err := os.WriteFile(path, []byte(spec), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}

	out, code := captureStdout(t, func() int { return runCheck([]string{path}) })

	if code != 0 {
		t.Fatalf("want exit code 0, got %d\noutput:\n%s", code, out)
	}
	if !strings.Contains(out, "K_drift:  0.00  [ok]") {
		t.Fatalf("fully-traced spec must render numeric ok verdict, got:\n%s", out)
	}
	if strings.Contains(out, "n/a") {
		t.Fatalf("fully-traced spec must not render the n/a signal, got:\n%s", out)
	}
}

// TestAggregate covers the aggregation+gating logic extracted from
// runCheck at the spec.Spec -> checkResult level, without going through
// file I/O or stdout capture (that's the runCheck-level tests above).
func TestAggregate(t *testing.T) {
	th := internal.DefaultThresholds()

	tests := []struct {
		name          string
		specs         []spec.Spec
		wantKDVerdict internal.Verdict
		wantDCVerdict internal.Verdict
		wantReqs      int
		wantHanging   int
		wantHangingID []string
	}{
		{
			name: "single spec pass",
			specs: []spec.Spec{
				{Path: "a.md", Content: []byte("[REQ-A-01]\n-> [FUN-A-01]\n@constraint\n@schema\n")},
			},
			wantKDVerdict: internal.VerdictOK,
			wantDCVerdict: internal.VerdictOK,
			wantReqs:      1,
			wantHanging:   0,
			wantHangingID: nil,
		},
		{
			name: "K_drift block",
			specs: []spec.Spec{
				{Path: "a.md", Content: []byte("[REQ-A-01] x\n-> [FUN-A-01] y\n[REQ-A-02] unlinked\n")},
			},
			wantKDVerdict: internal.VerdictBlock,
			wantDCVerdict: internal.VerdictWarn,
			wantReqs:      2,
			wantHanging:   1,
			wantHangingID: []string{"a.md: REQ-A-02"},
		},
		{
			name: "D_const warn",
			specs: []spec.Spec{
				{Path: "a.md", Content: []byte(`# Payments

@schema Transaction { id: UUID, amount: Decimal(10,2) @constraint(min: 0.01) }

1. [REQ-PAY-01] Validate incoming request against Transaction.
   -> [FUN-PAY-01] AcceptTransaction(tx Transaction) (Receipt, error)
2. [REQ-PAY-02] Log all operation results.
   -> [FUN-PAY-02] LogResult(txID, status, errorCode)
`)},
			},
			wantKDVerdict: internal.VerdictOK,
			wantDCVerdict: internal.VerdictWarn,
			wantReqs:      2,
			wantHanging:   0,
			wantHangingID: nil,
		},
		{
			name: "multi-file aggregation",
			specs: []spec.Spec{
				{Path: "a.md", Content: []byte("[REQ-A-01] x\n-> [FUN-A-01] y\n[REQ-A-02] unlinked\n")},
				{Path: "b.md", Content: []byte("[REQ-B-01] solo\n")},
			},
			wantKDVerdict: internal.VerdictBlock,
			wantDCVerdict: internal.VerdictWarn,
			wantReqs:      3,
			wantHanging:   2,
			wantHangingID: []string{"a.md: REQ-A-02", "b.md: REQ-B-01"},
		},
		{
			name: "zero requirements",
			specs: []spec.Spec{
				{Path: "a.md", Content: []byte("just prose, no requirement markup at all\n")},
			},
			wantKDVerdict: internal.VerdictSkipped,
			wantDCVerdict: internal.VerdictWarn,
			wantReqs:      0,
			wantHanging:   0,
			wantHangingID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := aggregate(tt.specs, th)

			if cr.KDVerdict != tt.wantKDVerdict {
				t.Fatalf("KDVerdict = %q, want %q; got %+v", cr.KDVerdict, tt.wantKDVerdict, cr)
			}
			if cr.DCVerdict != tt.wantDCVerdict {
				t.Fatalf("DCVerdict = %q, want %q; got %+v", cr.DCVerdict, tt.wantDCVerdict, cr)
			}
			if cr.KD.Requirements != tt.wantReqs {
				t.Fatalf("KD.Requirements = %d, want %d; got %+v", cr.KD.Requirements, tt.wantReqs, cr)
			}
			if cr.KD.Hanging != tt.wantHanging {
				t.Fatalf("KD.Hanging = %d, want %d; got %+v", cr.KD.Hanging, tt.wantHanging, cr)
			}
			if len(cr.KD.HangingIDs) != len(tt.wantHangingID) {
				t.Fatalf("KD.HangingIDs = %v, want %v", cr.KD.HangingIDs, tt.wantHangingID)
			}
			for i, id := range tt.wantHangingID {
				if cr.KD.HangingIDs[i] != id {
					t.Fatalf("KD.HangingIDs[%d] = %q, want %q; got %+v", i, cr.KD.HangingIDs[i], id, cr)
				}
			}
		})
	}
}
