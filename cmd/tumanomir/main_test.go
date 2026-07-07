package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
	"github.com/valpere/tumanomir/internal/instrument"
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

// captureStderr runs fn with os.Stderr redirected to a pipe and returns
// everything written to it, using the same pipe-deadlock-safe
// goroutine-drain design as captureStdout.
func captureStderr(t *testing.T, fn func() int) (string, int) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

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

// TestRunCheckHangingIDsPrinted guards runCheck's hanging-ID print loop
// (main.go's "hanging: <id>" lines) through its own stdout, not just at the
// lower aggregate() level (TestAggregate already covers the data; this
// covers the rendering) — issue #74.
func TestRunCheckHangingIDsPrinted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	spec := "[REQ-X-01] traced\n-> [FUN-X-01] Do()\n[REQ-X-02] not traced\n"
	if err := os.WriteFile(path, []byte(spec), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}

	out, code := captureStdout(t, func() int { return runCheck([]string{path}) })

	if code != 1 {
		t.Fatalf("want exit code 1 (K_drift gate failed), got %d\noutput:\n%s", code, out)
	}
	want := "    hanging: " + path + ": REQ-X-02\n"
	if !strings.Contains(out, want) {
		t.Fatalf("want hanging-ID line %q, got output:\n%s", want, out)
	}
}

// TestRunCheckArgCountValidation guards runCheck's "exactly one argument"
// validation branch through its own exit code/stderr, mirroring
// TestRunMeasureFlagValidation's style — no runCheck-level equivalent
// existed before (issue #74).
func TestRunCheckArgCountValidation(t *testing.T) {
	errOut, code := captureStderr(t, func() int { return runCheck(nil) })
	if code != 2 {
		t.Fatalf("code = %d, want 2; stderr:\n%s", code, errOut)
	}
	if errOut == "" {
		t.Fatal("want a non-empty actionable stderr message")
	}
}

// TestRunCheckSpecLoadFailure guards runCheck's spec.Load error branch — a
// non-existent path must return 2 with an actionable stderr message
// (issue #74).
func TestRunCheckSpecLoadFailure(t *testing.T) {
	errOut, code := captureStderr(t, func() int {
		return runCheck([]string{filepath.Join(t.TempDir(), "does-not-exist.md")})
	})
	if code != 2 {
		t.Fatalf("code = %d, want 2; stderr:\n%s", code, errOut)
	}
	if errOut == "" {
		t.Fatal("want a non-empty actionable stderr message")
	}
}

// TestDispatch covers dispatch's top-level command routing — the switch
// previously lived directly in main() and called os.Exit inline, making it
// untestable without a subprocess (issue #74).
func TestDispatch(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte("[REQ-X-01] x\n-> [FUN-X-01] y\n"), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}

	t.Run("check", func(t *testing.T) {
		out, code := captureStdout(t, func() int { return dispatch([]string{"check", specPath}) })
		if code != 0 {
			t.Fatalf("code = %d, want 0; output:\n%s", code, out)
		}
		if !strings.Contains(out, "K_drift") {
			t.Fatalf("want check output, got:\n%s", out)
		}
	})

	t.Run("measure", func(t *testing.T) {
		errOut, code := captureStderr(t, func() int { return dispatch([]string{"measure"}) })
		if code != 2 {
			t.Fatalf("code = %d, want 2 (missing --instrument); stderr:\n%s", code, errOut)
		}
	})

	t.Run("version", func(t *testing.T) {
		out, code := captureStdout(t, func() int { return dispatch([]string{"version"}) })
		if code != 0 {
			t.Fatalf("code = %d, want 0; output:\n%s", code, out)
		}
		if !strings.Contains(out, "tumanomir") {
			t.Fatalf("want version output, got:\n%s", out)
		}
	})

	for _, flag := range []string{"-h", "--help", "help"} {
		t.Run("help_"+flag, func(t *testing.T) {
			out, code := captureStdout(t, func() int { return dispatch([]string{flag}) })
			if code != 0 {
				t.Fatalf("code = %d, want 0; output:\n%s", code, out)
			}
			if !strings.Contains(out, "Usage:") {
				t.Fatalf("want usage output, got:\n%s", out)
			}
		})
	}

	t.Run("unknown command", func(t *testing.T) {
		errOut, code := captureStderr(t, func() int { return dispatch([]string{"bogus"}) })
		if code != 2 {
			t.Fatalf("code = %d, want 2; stderr:\n%s", code, errOut)
		}
		if !strings.Contains(errOut, `unknown command "bogus"`) {
			t.Fatalf("want unknown-command message, got stderr:\n%s", errOut)
		}
	})

	t.Run("no arguments", func(t *testing.T) {
		errOut, code := captureStderr(t, func() int { return dispatch(nil) })
		if code != 2 {
			t.Fatalf("code = %d, want 2; stderr:\n%s", code, errOut)
		}
		if !strings.Contains(errOut, "Usage:") {
			t.Fatalf("want usage output, got stderr:\n%s", errOut)
		}
	})
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
		wantKDValue   float64
		wantDCMarkers int
		wantDCProse   int
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
			wantKDValue:   0,
			wantDCMarkers: 4,
			wantDCProse:   2,
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
			wantKDValue:   0.5,
			wantDCMarkers: 3,
			wantDCProse:   6,
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
			wantKDValue:   0,
			wantDCMarkers: 6,
			wantDCProse:   33,
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
			wantKDValue:   2.0 / 3.0,
			wantDCMarkers: 4,
			wantDCProse:   8,
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
			wantKDValue:   0,
			wantDCMarkers: 0,
			wantDCProse:   7,
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
			const epsilon = 1e-9
			if diff := cr.KD.Value - tt.wantKDValue; diff > epsilon || diff < -epsilon {
				t.Fatalf("KD.Value = %v, want %v; got %+v", cr.KD.Value, tt.wantKDValue, cr)
			}
			if cr.DC.ConstraintMarkers != tt.wantDCMarkers {
				t.Fatalf("DC.ConstraintMarkers = %d, want %d; got %+v", cr.DC.ConstraintMarkers, tt.wantDCMarkers, cr)
			}
			if cr.DC.ProseTokens != tt.wantDCProse {
				t.Fatalf("DC.ProseTokens = %d, want %d; got %+v", cr.DC.ProseTokens, tt.wantDCProse, cr)
			}
			wantDCValue := 0.0
			if total := tt.wantDCMarkers + tt.wantDCProse; total > 0 {
				wantDCValue = float64(tt.wantDCMarkers) / float64(total)
			}
			if diff := cr.DC.Value - wantDCValue; diff > epsilon || diff < -epsilon {
				t.Fatalf("DC.Value = %v, want %v; got %+v", cr.DC.Value, wantDCValue, cr)
			}
		})
	}
}

// errFakeGenerate is a sentinel error simulating a hard instrument
// failure (network/HTTP/preflight) from a fake Generator.
var errFakeGenerate = errors.New("fake generator: simulated hard failure")

// fakeGenerator is a test double for instrument.Generator: no real network
// calls, responses driven by an injected function keyed on call index.
type fakeGenerator struct {
	fn    func(call int) (instrument.Generation, error)
	calls int
}

func (f *fakeGenerator) Generate(_ context.Context, _ string) (instrument.Generation, error) {
	call := f.calls
	f.calls++
	return f.fn(call)
}

// goBlock wraps src in the exact fenced form ExtractGoBlock expects.
func goBlock(src string) []byte {
	return []byte("```go\n" + src + "\n```\n")
}

const (
	testSrcFoo = `package a

type Foo struct {
	X int
}

func DoFoo(x int) error { return nil }
`
	testSrcBar = `package b

type Bar interface {
	Baz()
}

const Qux = 1
`
)

var testThresholds = internal.DefaultThresholds()

func genOK(text []byte) (instrument.Generation, error) {
	return instrument.Generation{Text: text}, nil
}

func TestRunMeasureWithGeneratorLowDPairAllValid(t *testing.T) {
	// All four samples are byte-identical Foo sources, so every pairwise
	// cosine similarity is 1.0 and D_pair == 0.00 — well under DPairMax.
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return genOK(goBlock(testSrcFoo))
	}}

	mr, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, []byte("spec"), 4, testThresholds)
	if err != nil {
		t.Fatalf("runMeasureWithGenerator() error = %v", err)
	}
	if mr.Dispersion.Discarded != 0 {
		t.Fatalf("Discarded = %d, want 0; got %+v", mr.Dispersion.Discarded, mr)
	}
	if mr.Dispersion.N != 4 {
		t.Fatalf("N = %d, want 4; got %+v", mr.Dispersion.N, mr)
	}
	if mr.DPairVerdict != internal.VerdictOK {
		t.Fatalf("DPairVerdict = %q, want ok; got %+v", mr.DPairVerdict, mr)
	}
	if mr.Dispersion.DPair != 0 {
		t.Fatalf("DPair = %v, want 0; got %+v", mr.Dispersion.DPair, mr)
	}
}

func TestRunMeasureWithGeneratorHighDPairBlocks(t *testing.T) {
	// Alternating between two structurally disjoint sources (no shared
	// feature keys) drives mean pairwise similarity toward 0 and D_pair
	// toward 1 — well over DPairMax, so the gate must block.
	srcs := []string{testSrcFoo, testSrcBar, testSrcFoo, testSrcBar}
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return genOK(goBlock(srcs[call%len(srcs)]))
	}}

	mr, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, []byte("spec"), 4, testThresholds)
	if err != nil {
		t.Fatalf("runMeasureWithGenerator() error = %v", err)
	}
	if mr.DPairVerdict != internal.VerdictBlock {
		t.Fatalf("DPairVerdict = %q, want block; got %+v", mr.DPairVerdict, mr)
	}
	if mr.Dispersion.DPair <= testThresholds.DPairMax {
		t.Fatalf("DPair = %v, want > %v; got %+v", mr.Dispersion.DPair, testThresholds.DPairMax, mr)
	}
}

func TestRunMeasureWithGeneratorRetriesThenDiscards(t *testing.T) {
	const invalidText = "no fenced go block here at all\n"

	// slot 0 (call 0): valid immediately.
	// slot 1 (calls 1-3): invalid on all 3 attempts -> discarded.
	// slot 2 (calls 4-6): invalid, invalid, then valid on the 3rd attempt.
	responses := [][]byte{
		goBlock(testSrcFoo),
		[]byte(invalidText), []byte(invalidText), []byte(invalidText),
		[]byte(invalidText), []byte(invalidText), goBlock(testSrcFoo),
	}
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return genOK(responses[call])
	}}

	mr, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, []byte("spec"), 3, testThresholds)
	if err != nil {
		t.Fatalf("runMeasureWithGenerator() error = %v", err)
	}
	if gen.calls != len(responses) {
		t.Fatalf("calls = %d, want %d (no padding back up after a discard)", gen.calls, len(responses))
	}
	if mr.Dispersion.Discarded != 1 {
		t.Fatalf("Discarded = %d, want 1; got %+v", mr.Dispersion.Discarded, mr)
	}
	if mr.Dispersion.N != 2 {
		t.Fatalf("N = %d, want 2; got %+v", mr.Dispersion.N, mr)
	}
}

func TestRunMeasureWithGeneratorErrorFailsFast(t *testing.T) {
	wantErr := errFakeGenerate
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return instrument.Generation{}, wantErr
	}}

	_, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, []byte("spec"), 5, testThresholds)
	if err == nil {
		t.Fatal("runMeasureWithGenerator() error = nil, want non-nil on Generate failure")
	}
	if gen.calls != 1 {
		t.Fatalf("calls = %d, want 1 (must fail fast, no retry on a Generate error)", gen.calls)
	}
}

func TestRunMeasureWithGeneratorValidSamplesBelowTwoIsSkipped(t *testing.T) {
	const invalidText = "no fenced go block here at all\n"
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return genOK([]byte(invalidText))
	}}

	mr, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, []byte("spec"), 2, testThresholds)
	if err != nil {
		t.Fatalf("runMeasureWithGenerator() error = %v, want nil (this is a valid, if disappointing, measurement outcome)", err)
	}
	if mr.Dispersion.N >= 2 {
		t.Fatalf("N = %d, want < 2 for this fixture", mr.Dispersion.N)
	}
	if mr.DPairVerdict != internal.VerdictSkipped {
		t.Fatalf("DPairVerdict = %q, want skipped; got %+v", mr.DPairVerdict, mr)
	}
}

func TestRunMeasureWithGeneratorDiscardWarnThreshold(t *testing.T) {
	const invalidText = "no fenced go block here at all\n"

	tests := []struct {
		name        string
		discards    int // number of always-invalid slots, out of 5
		wantDiscard int
		wantWarn    bool
	}{
		{name: "60% discard warns", discards: 3, wantDiscard: 3, wantWarn: true},
		{name: "20% discard does not warn", discards: 1, wantDiscard: 1, wantWarn: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const samples = 5

			// Build an explicit response table: the first tt.discards slots
			// consume all 3 retry attempts with invalid text (discarded),
			// the rest succeed on their first attempt.
			var responses [][]byte
			for i := 0; i < samples; i++ {
				if i < tt.discards {
					responses = append(responses, []byte(invalidText), []byte(invalidText), []byte(invalidText))
				} else {
					responses = append(responses, goBlock(testSrcFoo))
				}
			}
			g := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
				return genOK(responses[call])
			}}

			mr, err := runMeasureWithGenerator(g, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, []byte("spec"), samples, testThresholds)
			if err != nil {
				t.Fatalf("runMeasureWithGenerator() error = %v", err)
			}
			if mr.Dispersion.Discarded != tt.wantDiscard {
				t.Fatalf("Discarded = %d, want %d; got %+v", mr.Dispersion.Discarded, tt.wantDiscard, mr)
			}
			if mr.DiscardWarn != tt.wantWarn {
				t.Fatalf("DiscardWarn = %v, want %v; got %+v (rate %.2f)", mr.DiscardWarn, tt.wantWarn, mr, mr.DiscardRate)
			}
		})
	}
}

// TestRunMeasureWithGeneratorCountsTruncated verifies that an accepted
// (valid-Go) generation with DoneReason=="length" is counted in
// measureResult.Truncated, distinct from Discarded — REQ-MSR-06's
// measurement-integrity gap: a truncated-but-parseable sample is still
// accepted into N, not silently treated as a clean sample.
func TestRunMeasureWithGeneratorCountsTruncated(t *testing.T) {
	// slot 0: valid, done_reason=stop (clean).
	// slot 1: valid, done_reason=length (truncated, still accepted).
	// slot 2: valid, done_reason=length (truncated, still accepted).
	responses := []instrument.Generation{
		{Text: goBlock(testSrcFoo), DoneReason: "stop"},
		{Text: goBlock(testSrcFoo), DoneReason: "length"},
		{Text: goBlock(testSrcFoo), DoneReason: "length"},
	}
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return responses[call], nil
	}}

	mr, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, []byte("spec"), 3, testThresholds)
	if err != nil {
		t.Fatalf("runMeasureWithGenerator() error = %v", err)
	}
	if mr.Dispersion.Discarded != 0 {
		t.Fatalf("Discarded = %d, want 0; got %+v", mr.Dispersion.Discarded, mr)
	}
	if mr.Dispersion.N != 3 {
		t.Fatalf("N = %d, want 3 (truncated-but-valid samples are still accepted); got %+v", mr.Dispersion.N, mr)
	}
	if mr.Truncated != 2 {
		t.Fatalf("Truncated = %d, want 2; got %+v", mr.Truncated, mr)
	}
}

// TestRunMeasureWithGeneratorNoTruncation verifies Truncated stays 0 when
// no accepted generation reports done_reason=length.
func TestRunMeasureWithGeneratorNoTruncation(t *testing.T) {
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return instrument.Generation{Text: goBlock(testSrcFoo), DoneReason: "stop"}, nil
	}}

	mr, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, []byte("spec"), 3, testThresholds)
	if err != nil {
		t.Fatalf("runMeasureWithGenerator() error = %v", err)
	}
	if mr.Truncated != 0 {
		t.Fatalf("Truncated = %d, want 0; got %+v", mr.Truncated, mr)
	}
}

// TestRunMeasureWithGeneratorFlagsPromptUnderestimate verifies that a
// generation whose actual PromptEvalCount exceeds the preflight byte/3
// estimate by more than promptEstimateDivergenceFactor is counted in
// measureResult.PromptUnderestimated (issue #57): the heuristic
// under-counts non-ASCII prompts, so this is a diagnostic signal that the
// preflight's "errs toward refusing" guarantee may not have held.
func TestRunMeasureWithGeneratorFlagsPromptUnderestimate(t *testing.T) {
	specContent := []byte("spec")
	estimate := instrument.EstimatePromptTokens(instrument.BuildPrompt(specContent))

	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return instrument.Generation{
			Text:            goBlock(testSrcFoo),
			PromptEvalCount: int(float64(estimate) * 2), // well above the 1.5x threshold
		}, nil
	}}

	mr, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, specContent, 3, testThresholds)
	if err != nil {
		t.Fatalf("runMeasureWithGenerator() error = %v", err)
	}
	if mr.PromptUnderestimated != 3 {
		t.Fatalf("PromptUnderestimated = %d, want 3 (all 3 accepted generations exceed the estimate); got %+v", mr.PromptUnderestimated, mr)
	}
}

// TestRunMeasureWithGeneratorNoPromptUnderestimateWarning verifies
// PromptUnderestimated stays 0 when the actual PromptEvalCount tracks the
// preflight estimate closely (ASCII-sized prompt, no real divergence).
func TestRunMeasureWithGeneratorNoPromptUnderestimateWarning(t *testing.T) {
	specContent := []byte("spec")
	estimate := instrument.EstimatePromptTokens(instrument.BuildPrompt(specContent))

	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return instrument.Generation{
			Text:            goBlock(testSrcFoo),
			PromptEvalCount: estimate, // exactly at the estimate: no divergence
		}, nil
	}}

	mr, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, specContent, 3, testThresholds)
	if err != nil {
		t.Fatalf("runMeasureWithGenerator() error = %v", err)
	}
	if mr.PromptUnderestimated != 0 {
		t.Fatalf("PromptUnderestimated = %d, want 0; got %+v", mr.PromptUnderestimated, mr)
	}
}

// TestRunMeasureWithGeneratorPromptUnderestimateBoundary guards the
// strict '>' in the threshold comparison (fix-review, glm-5.1:cloud +
// kimi-k2.6:cloud, independently): a PromptEvalCount landing exactly at
// internal.PromptEstimateDivergenceFactor x the estimate must NOT trigger
// the counter — only strictly exceeding it should. Protects against a
// future accidental change to '>='.
func TestRunMeasureWithGeneratorPromptUnderestimateBoundary(t *testing.T) {
	specContent := []byte("spec")
	estimate := instrument.EstimatePromptTokens(instrument.BuildPrompt(specContent))
	atThreshold := int(float64(estimate) * internal.PromptEstimateDivergenceFactor)

	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return instrument.Generation{
			Text:            goBlock(testSrcFoo),
			PromptEvalCount: atThreshold, // exactly 1.5x: must NOT trigger (strict >)
		}, nil
	}}

	mr, err := runMeasureWithGenerator(gen, internal.InstrumentConfig{Backend: "ollama", Model: "test", SimThreshold: 0.95}, specContent, 3, testThresholds)
	if err != nil {
		t.Fatalf("runMeasureWithGenerator() error = %v", err)
	}
	if mr.PromptUnderestimated != 0 {
		t.Fatalf("PromptUnderestimated = %d, want 0 at exactly the threshold (%d); got %+v", mr.PromptUnderestimated, atThreshold, mr)
	}
}

func TestRunMeasureFlagValidation(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte("[REQ-X-01] x\n"), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing --instrument",
			args: []string{"--num-ctx", "8192", "--num-predict", "2048", specPath},
		},
		{
			name: "malformed --instrument (no colon)",
			args: []string{"--instrument", "ollama", "--num-ctx", "8192", "--num-predict", "2048", specPath},
		},
		{
			name: "empty model in --instrument",
			args: []string{"--instrument", "ollama:", "--num-ctx", "8192", "--num-predict", "2048", specPath},
		},
		{
			name: "empty backend in --instrument",
			args: []string{"--instrument", ":qwen3-coder:30b", "--num-ctx", "8192", "--num-predict", "2048", specPath},
		},
		{
			name: "unsupported backend",
			args: []string{"--instrument", "openai:gpt-4", "--num-ctx", "8192", "--num-predict", "2048", specPath},
		},
		{
			name: "samples < 2",
			args: []string{"--instrument", "ollama:m", "-n", "1", "--num-ctx", "8192", "--num-predict", "2048", specPath},
		},
		{
			name: "sim-threshold out of range",
			args: []string{"--instrument", "ollama:m", "--sim-threshold", "1.5", "--num-ctx", "8192", "--num-predict", "2048", specPath},
		},
		{
			name: "missing --num-ctx",
			args: []string{"--instrument", "ollama:m", "--num-predict", "2048", specPath},
		},
		{
			name: "missing --num-predict",
			args: []string{"--instrument", "ollama:m", "--num-ctx", "8192", specPath},
		},
		{
			name: "directory positional argument",
			args: []string{"--instrument", "ollama:m", "--num-ctx", "8192", "--num-predict", "2048", subdir},
		},
		{
			name: "wrong number of positional arguments",
			args: []string{"--instrument", "ollama:m", "--num-ctx", "8192", "--num-predict", "2048"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errOut, code := captureStderr(t, func() int { return runMeasure(tt.args) })
			if code != 2 {
				t.Fatalf("code = %d, want 2; stderr:\n%s", code, errOut)
			}
			if errOut == "" {
				t.Fatal("want a non-empty actionable stderr message")
			}
		})
	}
}

// TestRunMeasureImplFlagMapping drives runMeasureImpl through its full CLI
// wiring (flag parse -> InstrumentConfig -> generator construction) and
// captures the InstrumentConfig the (faked) generator constructor actually
// received, guarding against a flag-to-field mapping typo that
// runMeasureWithGenerator's own direct-call tests can't catch (issue #70).
func TestRunMeasureImplFlagMapping(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte("[REQ-X-01] x\n"), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}

	args := []string{
		"--instrument", "ollama:my-model",
		"--temp", "0.7",
		"--samples", "4",
		"--think",
		"--num-ctx", "4096",
		"--num-predict", "512",
		"--sim-threshold", "0.8",
		specPath,
	}

	var gotCfg internal.InstrumentConfig
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return genOK(goBlock(testSrcFoo))
	}}
	_, code := captureStdout(t, func() int {
		return runMeasureImpl(args, func(cfg internal.InstrumentConfig) instrument.Generator {
			gotCfg = cfg
			return gen
		})
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}

	want := internal.InstrumentConfig{
		Backend:      "ollama",
		Model:        "my-model",
		Temperature:  0.7,
		Samples:      4,
		Think:        true,
		NumCtx:       4096,
		NumPredict:   512,
		SimThreshold: 0.8,
	}
	if gotCfg.Backend != want.Backend || gotCfg.Model != want.Model ||
		gotCfg.Temperature != want.Temperature || gotCfg.Samples != want.Samples ||
		gotCfg.Think != want.Think || gotCfg.NumCtx != want.NumCtx ||
		gotCfg.NumPredict != want.NumPredict || gotCfg.SimThreshold != want.SimThreshold {
		t.Fatalf("cfg = %+v, want fields matching %+v", gotCfg, want)
	}
	if gotCfg.Prompt == "" || gotCfg.PromptVersion == "" {
		t.Fatalf("cfg.Prompt/PromptVersion left unset; got %+v", gotCfg)
	}
}

// TestRunMeasureImplExitCode drives runMeasureImpl end-to-end (real flag
// parsing, not a direct runMeasureWithGenerator call) and asserts the exit
// code follows DPairVerdict — the branch at main.go:332-335 that no
// existing test reaches through runMeasure's own wiring (issue #70).
func TestRunMeasureImplExitCode(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte("[REQ-X-01] x\n"), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}
	args := []string{
		"--instrument", "ollama:m",
		"--samples", "4",
		"--num-ctx", "8192",
		"--num-predict", "2048",
		specPath,
	}

	tests := []struct {
		name     string
		newGen   func(internal.InstrumentConfig) instrument.Generator
		wantCode int
	}{
		{
			name: "DPairVerdict ok -> exit 0",
			newGen: func(internal.InstrumentConfig) instrument.Generator {
				return &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
					return genOK(goBlock(testSrcFoo))
				}}
			},
			wantCode: 0,
		},
		{
			name: "DPairVerdict block -> exit 1",
			newGen: func(internal.InstrumentConfig) instrument.Generator {
				srcs := []string{testSrcFoo, testSrcBar, testSrcFoo, testSrcBar}
				return &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
					return genOK(goBlock(srcs[call%len(srcs)]))
				}}
			},
			wantCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, code := captureStdout(t, func() int { return runMeasureImpl(args, tt.newGen) })
			if code != tt.wantCode {
				t.Fatalf("code = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

// TestRunMeasureImplGeneratorErrorReturns2 asserts that a hard Generate
// failure (network/HTTP/preflight, surfaced by runMeasureWithGenerator as a
// non-nil error) is printed to stderr and causes runMeasureImpl itself to
// return 2, through the full wiring rather than a direct
// runMeasureWithGenerator call (issue #70).
func TestRunMeasureImplGeneratorErrorReturns2(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte("[REQ-X-01] x\n"), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}
	args := []string{
		"--instrument", "ollama:m",
		"--samples", "4",
		"--num-ctx", "8192",
		"--num-predict", "2048",
		specPath,
	}

	errOut, code := captureStderr(t, func() int {
		return runMeasureImpl(args, func(internal.InstrumentConfig) instrument.Generator {
			return &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
				return instrument.Generation{}, errFakeGenerate
			}}
		})
	})
	if code != 2 {
		t.Fatalf("code = %d, want 2; stderr:\n%s", code, errOut)
	}
	if errOut == "" {
		t.Fatal("want a non-empty actionable stderr message")
	}
}

// --- .tumanomir.yaml config support (issue #86) ---

// chdir switches the test's working directory to dir for the duration of
// the test, restoring the original on cleanup — needed for exercising the
// cwd-only ./.tumanomir.yaml discovery rule without touching the repo's
// own working directory (which has no such file, so this is safe).
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%s): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("restore os.Chdir(%s): %v", orig, err)
		}
	})
}

func TestScanConfigFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		wantOk   bool
	}{
		{"absent", []string{"--instrument", "ollama:m"}, "", false},
		{"--config value", []string{"--config", "foo.yaml", "spec.md"}, "foo.yaml", true},
		{"--config=value", []string{"--config=foo.yaml"}, "foo.yaml", true},
		{"-config value (single dash)", []string{"-config", "foo.yaml"}, "foo.yaml", true},
		{"appears after another flag's own value", []string{"--k-drift-max", "0.5", "--config", "foo.yaml"}, "foo.yaml", true},
		{"stops scanning at --", []string{"--", "--config", "foo.yaml"}, "", false},
		{"--config with no following value", []string{"--config"}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ok := scanConfigFlag(tt.args)
			if path != tt.wantPath || ok != tt.wantOk {
				t.Fatalf("scanConfigFlag(%v) = (%q, %v), want (%q, %v)", tt.args, path, ok, tt.wantPath, tt.wantOk)
			}
		})
	}
}

func TestResolveConfigExplicitMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	errOut, code := captureStderr(t, func() int {
		if _, ok := resolveConfig([]string{"--config", missing}, "check"); !ok {
			return 2
		}
		return 0
	})
	if code != 2 {
		t.Fatalf("code = %d, want 2; stderr:\n%s", code, errOut)
	}
	if errOut == "" {
		t.Fatal("want a non-empty actionable stderr message")
	}
}

func TestResolveConfigExplicitMalformedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("thresholds: [not a mapping\n"), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	errOut, code := captureStderr(t, func() int {
		if _, ok := resolveConfig([]string{"--config", path}, "check"); !ok {
			return 2
		}
		return 0
	})
	if code != 2 {
		t.Fatalf("code = %d, want 2; stderr:\n%s", code, errOut)
	}
	if errOut == "" {
		t.Fatal("want a non-empty actionable stderr message")
	}
}

func TestResolveConfigDefaultAbsentIsSilentlySkipped(t *testing.T) {
	chdir(t, t.TempDir())

	cfg, ok := resolveConfig(nil, "check")
	if !ok {
		t.Fatal("want ok=true when ./.tumanomir.yaml is absent")
	}
	if cfg.Thresholds != nil || cfg.Instrument != nil {
		t.Fatalf("want a zero-value Config when no config file is found, got %+v", cfg)
	}
}

func TestResolveConfigDefaultPresentIsLoaded(t *testing.T) {
	chdir(t, t.TempDir())
	if err := os.WriteFile(defaultConfigPath, []byte("thresholds:\n  k_drift_max: 0.10\n"), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	cfg, ok := resolveConfig(nil, "check")
	if !ok {
		t.Fatal("want ok=true when ./.tumanomir.yaml parses cleanly")
	}
	if cfg.Thresholds == nil || cfg.Thresholds.KDriftMax == nil || *cfg.Thresholds.KDriftMax != 0.10 {
		t.Fatalf("cfg.Thresholds = %+v, want KDriftMax=0.10", cfg.Thresholds)
	}
}

// TestRunCheckConfigLoosensThreshold guards the full runCheck wiring: a
// spec that would fail the default K_drift gate (0.20) passes once
// ./.tumanomir.yaml raises k_drift_max above its actual value.
func TestRunCheckConfigLoosensThreshold(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	specPath := filepath.Join(dir, "spec.md")
	spec := "[REQ-X-01] traced\n-> [FUN-X-01] Do()\n[REQ-X-02] not traced\n"
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}
	if err := os.WriteFile(defaultConfigPath, []byte("thresholds:\n  k_drift_max: 0.90\n"), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	out, code := captureStdout(t, func() int { return runCheck([]string{specPath}) })
	if code != 0 {
		t.Fatalf("want exit code 0 (K_drift 0.5 <= configured 0.90), got %d\noutput:\n%s", code, out)
	}
}

// TestRunCheckCLIFlagOverridesConfig asserts CLI flag > config precedence:
// an explicit --k-drift-max still gates even though the config file would
// otherwise loosen the threshold enough to pass.
func TestRunCheckCLIFlagOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	specPath := filepath.Join(dir, "spec.md")
	spec := "[REQ-X-01] traced\n-> [FUN-X-01] Do()\n[REQ-X-02] not traced\n"
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}
	if err := os.WriteFile(defaultConfigPath, []byte("thresholds:\n  k_drift_max: 0.90\n"), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	out, code := captureStdout(t, func() int {
		return runCheck([]string{"--k-drift-max", "0.0", specPath})
	})
	if code != 1 {
		t.Fatalf("want exit code 1 (--k-drift-max=0.0 overrides the looser config value), got %d\noutput:\n%s", code, out)
	}
}

func TestRunCheckExplicitConfigMissingFileReturns2(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte("[REQ-X-01] x\n-> [FUN-X-01] y\n"), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}
	missing := filepath.Join(dir, "does-not-exist.yaml")

	errOut, code := captureStderr(t, func() int {
		return runCheck([]string{"--config", missing, specPath})
	})
	if code != 2 {
		t.Fatalf("code = %d, want 2; stderr:\n%s", code, errOut)
	}
	if errOut == "" {
		t.Fatal("want a non-empty actionable stderr message")
	}
}

// TestRunMeasureImplConfigSeedsInstrumentDefaults drives runMeasureImpl
// with no --instrument/--samples/... flags at all, relying entirely on
// ./.tumanomir.yaml to supply them, mirroring
// TestRunMeasureImplFlagMapping's captured-InstrumentConfig style.
func TestRunMeasureImplConfigSeedsInstrumentDefaults(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte("[REQ-X-01] x\n"), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}
	configYAML := `
instrument:
  backend: ollama
  model: my-model
  temperature: 0.7
  samples: 4
  think: true
  num_ctx: 4096
  num_predict: 512
  sim_threshold: 0.8
`
	if err := os.WriteFile(defaultConfigPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	var gotCfg internal.InstrumentConfig
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return genOK(goBlock(testSrcFoo))
	}}
	out, code := captureStdout(t, func() int {
		return runMeasureImpl([]string{specPath}, func(cfg internal.InstrumentConfig) instrument.Generator {
			gotCfg = cfg
			return gen
		})
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0\noutput:\n%s", code, out)
	}

	want := internal.InstrumentConfig{
		Backend:      "ollama",
		Model:        "my-model",
		Temperature:  0.7,
		Samples:      4,
		Think:        true,
		NumCtx:       4096,
		NumPredict:   512,
		SimThreshold: 0.8,
	}
	if gotCfg.Backend != want.Backend || gotCfg.Model != want.Model ||
		gotCfg.Temperature != want.Temperature || gotCfg.Samples != want.Samples ||
		gotCfg.Think != want.Think || gotCfg.NumCtx != want.NumCtx ||
		gotCfg.NumPredict != want.NumPredict || gotCfg.SimThreshold != want.SimThreshold {
		t.Fatalf("cfg = %+v, want fields matching %+v", gotCfg, want)
	}
}

// TestRunMeasureImplCLIFlagOverridesConfig asserts CLI flag > config
// precedence on the instrument side: an explicit --samples still wins
// over the config file's value.
func TestRunMeasureImplCLIFlagOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	specPath := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(specPath, []byte("[REQ-X-01] x\n"), 0o644); err != nil {
		t.Fatalf("write temp spec: %v", err)
	}
	configYAML := `
instrument:
  backend: ollama
  model: my-model
  samples: 10
  num_ctx: 4096
  num_predict: 512
`
	if err := os.WriteFile(defaultConfigPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write default config: %v", err)
	}

	var gotCfg internal.InstrumentConfig
	gen := &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		return genOK(goBlock(testSrcFoo))
	}}
	out, code := captureStdout(t, func() int {
		return runMeasureImpl([]string{"--samples", "4", specPath}, func(cfg internal.InstrumentConfig) instrument.Generator {
			gotCfg = cfg
			return gen
		})
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0\noutput:\n%s", code, out)
	}
	if gotCfg.Samples != 4 {
		t.Fatalf("gotCfg.Samples = %d, want 4 (CLI flag must override config's 10)", gotCfg.Samples)
	}
}
