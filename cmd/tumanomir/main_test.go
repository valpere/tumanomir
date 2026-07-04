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
			wantDCProse:   5,
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
			wantDCProse:   7,
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
			wantDCProse:   36,
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
			wantDCProse:   9,
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

func TestPrintMeasureResultDiscardWarningVisibility(t *testing.T) {
	warnMR := measureResult{
		Dispersion:  internal.DispersionResult{N: 2, Discarded: 8},
		Config:      internal.InstrumentConfig{Backend: "ollama", Model: "test"},
		DiscardRate: 0.8,
		DiscardWarn: true,
	}
	out, _ := captureStdout(t, func() int { printMeasureResult(warnMR, testThresholds); return 0 })
	if !strings.Contains(out, "discard rate") {
		t.Fatalf("want a discard-rate warning line for DiscardWarn=true, got:\n%s", out)
	}

	noWarnMR := warnMR
	noWarnMR.DiscardWarn = false
	out, _ = captureStdout(t, func() int { printMeasureResult(noWarnMR, testThresholds); return 0 })
	if strings.Contains(out, "discard rate") {
		t.Fatalf("must not print the discard-rate warning when DiscardWarn=false, got:\n%s", out)
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
