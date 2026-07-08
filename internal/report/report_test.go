package report

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
)

var testThresholds = internal.DefaultThresholds()

// failingWriter fails every Write call with err — used to verify errWriter
// (report.go) propagates a real write failure and stops after the first
// one, rather than continuing to write against an already-broken io.Writer.
type failingWriter struct {
	err   error
	calls int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.calls++
	return 0, w.err
}

// TestRenderCheckPropagatesWriteError guards errWriter's error path: no
// existing test exercises what happens when the underlying io.Writer
// fails, since every other test uses strings.Builder (whose Write never
// fails). Asserts RenderCheck returns the write error and stops after
// exactly one write attempt (fix-review, glm-5.1:cloud).
func TestRenderCheckPropagatesWriteError(t *testing.T) {
	wantErr := errors.New("boom")
	fw := &failingWriter{err: wantErr}

	err := RenderCheck(fw, CheckResult{KDVerdict: internal.VerdictOK, DCVerdict: internal.VerdictOK}, testThresholds)
	if !errors.Is(err, wantErr) {
		t.Fatalf("RenderCheck() error = %v, want %v", err, wantErr)
	}
	if fw.calls != 1 {
		t.Fatalf("want exactly 1 write attempt (errWriter must stop after the first error), got %d", fw.calls)
	}
}

// TestRenderCheckExactOutput is a full-string golden test — every existing
// RenderCheck assertion in this file uses strings.Contains, which would NOT
// catch a reordered line or dropped spacing. Captured as a faithful snapshot
// of RenderCheck's output *before* extracting writeCheckMetrics out of its
// body (issue #87); it must keep passing unmodified through that extraction
// as the actual proof the output stayed byte-identical.
func TestRenderCheckExactOutput(t *testing.T) {
	cr := CheckResult{
		KD: internal.KDriftResult{
			Requirements: 2,
			Hanging:      1,
			HangingIDs:   []string{"a.md: REQ-A-02"},
			Value:        0.5,
		},
		DC: internal.DConstResult{
			ConstraintMarkers: 3,
			ProseTokens:       6,
			Value:             3.0 / 9.0,
		},
		KDVerdict: internal.VerdictBlock,
		DCVerdict: internal.VerdictWarn,
	}
	var buf strings.Builder
	if err := RenderCheck(&buf, cr, testThresholds); err != nil {
		t.Fatalf("RenderCheck: %v", err)
	}

	want := "  K_drift:  0.50  [block]  (threshold 0.20, 1/2 requirements untraced)\n" +
		"  D_const:  0.33  [warn]   (threshold 0.35, 3 markers / 6 prose tokens)\n" +
		"  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)\n" +
		"    hanging: a.md: REQ-A-02\n"
	if buf.String() != want {
		t.Fatalf("RenderCheck() =\n%q\nwant\n%q", buf.String(), want)
	}
}

// TestRenderMeasureExactOutput is RenderMeasure's counterpart to
// TestRenderCheckExactOutput: a full-string golden test exercising every
// warning line (discard-rate, truncation, prompt-underestimate) plus the
// instrument-config block and the D_pair/H/H_norm lines in one fixture, so
// the extraction of writeMeasureWarnings/writeMeasureMetrics out of
// RenderMeasure's body (issue #87) has an exact byte-identical baseline to
// keep passing.
func TestRenderMeasureExactOutput(t *testing.T) {
	mr := MeasureResult{
		Dispersion: internal.DispersionResult{
			N:           5,
			Discarded:   3,
			MeanSim:     0.82,
			DPair:       0.18,
			DPairCILow:  0.09,
			DPairCIHigh: 0.27,
			Clusters:    2,
			SimThresh:   0.95,
			H:           1.37,
			HNorm:       0.59,
		},
		Config: internal.InstrumentConfig{
			Backend:       "ollama",
			Model:         "qwen3-coder:30b",
			Temperature:   1.0,
			Samples:       8,
			Think:         false,
			NumCtx:        8192,
			NumPredict:    2048,
			SimThreshold:  0.95,
			Prompt:        "abcde",
			PromptVersion: "PromptV1",
		},
		DPairVerdict:         internal.VerdictOK,
		DiscardRate:          3.0 / 8.0,
		DiscardWarn:          true,
		Truncated:            2,
		PromptUnderestimated: 1,
	}
	out := mustRenderMeasure(t, mr)

	want := "⚠ discard rate: 38% (3/8 generations invalid) — exceeds the 40% hypothesis threshold (REQ-MSR-05); results may be unreliable\n\n" +
		"⚠ 2/5 accepted generations had done_reason=length (truncated by num_predict) — their AST may not reflect the model's full intended output; consider raising --num-predict\n\n" +
		"⚠ 1 generation(s) had an actual prompt-token count over 1.5x the preflight estimate — the byte/3 heuristic under-counts non-ASCII (e.g. Cyrillic) prompts and may not have caught a real truncation risk; verify --num-ctx has enough headroom\n\n" +
		"Instrument config (REQ-MSR-04):\n" +
		"  backend:        ollama\n" +
		"  model:          qwen3-coder:30b\n" +
		"  temperature:    1.00\n" +
		"  samples (N):    8\n" +
		"  think:          false\n" +
		"  num_ctx:        8192\n" +
		"  num_predict:    2048\n" +
		"  sim_threshold:  0.95\n" +
		"  prompt:         PromptV1 (5 bytes)\n\n" +
		"  D_pair:   0.18  [ok]     (95% CI [0.09, 0.27]; threshold 0.30, mean sim 0.82, N=5 valid, 3 discarded)\n" +
		"  H:        1.37  bits (ordinal signal only, not gated)\n" +
		"  H_norm:   0.59  (ordinal signal only, not gated)\n"
	if out != want {
		t.Fatalf("RenderMeasure() =\n%q\nwant\n%q", out, want)
	}
}

// mustRenderMeasure renders mr and fails the test on any write error —
// bytes.Buffer's Write never actually fails, so this should always
// succeed; the check exists for lint-cleanliness and future-proofing if
// RenderMeasure's contract ever changes.
func mustRenderMeasure(t *testing.T, mr MeasureResult) string {
	t.Helper()
	var buf strings.Builder
	if err := RenderMeasure(&buf, mr, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	return buf.String()
}

func TestRenderMeasureDiscardWarningVisibility(t *testing.T) {
	warnMR := MeasureResult{
		Dispersion:  internal.DispersionResult{N: 2, Discarded: 8},
		Config:      internal.InstrumentConfig{Backend: "ollama", Model: "test"},
		DiscardRate: 0.8,
		DiscardWarn: true,
	}
	out := mustRenderMeasure(t, warnMR)
	if !strings.Contains(out, "discard rate") {
		t.Fatalf("want a discard-rate warning line for DiscardWarn=true, got:\n%s", out)
	}

	noWarnMR := warnMR
	noWarnMR.DiscardWarn = false
	out = mustRenderMeasure(t, noWarnMR)
	if strings.Contains(out, "discard rate") {
		t.Fatalf("must not print the discard-rate warning when DiscardWarn=false, got:\n%s", out)
	}
}

// TestRenderMeasureTruncationWarningVisibility verifies the
// done_reason=length truncation warning (REQ-MSR-06) appears when
// Truncated > 0 and is absent when Truncated == 0 — a separate line from
// the discard-rate warning, since the two flag distinct failure modes.
func TestRenderMeasureTruncationWarningVisibility(t *testing.T) {
	truncMR := MeasureResult{
		Dispersion: internal.DispersionResult{N: 10, Discarded: 0},
		Config:     internal.InstrumentConfig{Backend: "ollama", Model: "test"},
		Truncated:  3,
	}
	out := mustRenderMeasure(t, truncMR)
	wantWarn := "3/10 accepted generations had done_reason=length"
	if !strings.Contains(out, wantWarn) {
		t.Fatalf("want a truncation warning line containing %q for Truncated=3, got:\n%s", wantWarn, out)
	}

	noTruncMR := truncMR
	noTruncMR.Truncated = 0
	out = mustRenderMeasure(t, noTruncMR)
	if strings.Contains(out, "done_reason=length") {
		t.Fatalf("must not print the truncation warning when Truncated=0, got:\n%s", out)
	}
}

// TestRenderMeasurePromptUnderestimateWarningVisibility verifies the
// prompt-token-underestimate warning (issue #57) appears when
// PromptUnderestimated > 0 and is absent when it's 0 — a separate line
// from the discard-rate and truncation warnings, since it flags a
// distinct failure mode (preflight estimate divergence, not invalid or
// truncated output).
func TestRenderMeasurePromptUnderestimateWarningVisibility(t *testing.T) {
	underMR := MeasureResult{
		Dispersion:           internal.DispersionResult{N: 10, Discarded: 0},
		Config:               internal.InstrumentConfig{Backend: "ollama", Model: "test"},
		PromptUnderestimated: 4,
	}
	out := mustRenderMeasure(t, underMR)
	wantWarn := "4 generation(s) had an actual prompt-token count"
	if !strings.Contains(out, wantWarn) {
		t.Fatalf("want a prompt-underestimate warning line containing %q for PromptUnderestimated=4, got:\n%s", wantWarn, out)
	}

	noneMR := underMR
	noneMR.PromptUnderestimated = 0
	out = mustRenderMeasure(t, noneMR)
	if strings.Contains(out, "preflight estimate") {
		t.Fatalf("must not print the prompt-underestimate warning when PromptUnderestimated=0, got:\n%s", out)
	}
}

// TestRenderMeasureOKVerdict covers the normal (non-skipped) output path
// with a passing D_pair verdict, asserting both the REQ-MSR-04
// instrument-config block and the D_pair/H/H_norm lines render with the
// exact format strings RenderMeasure uses.
func TestRenderMeasureOKVerdict(t *testing.T) {
	mr := MeasureResult{
		Dispersion: internal.DispersionResult{
			N:           5,
			Discarded:   1,
			MeanSim:     0.82,
			DPair:       0.18,
			DPairCILow:  0.09,
			DPairCIHigh: 0.27,
			Clusters:    2,
			SimThresh:   0.95,
			H:           1.37,
			HNorm:       0.59,
		},
		Config: internal.InstrumentConfig{
			Backend:       "ollama",
			Model:         "qwen3-coder:30b",
			Temperature:   1.0,
			Samples:       5,
			Think:         false,
			NumCtx:        8192,
			NumPredict:    2048,
			SimThreshold:  0.95,
			Prompt:        "abcde",
			PromptVersion: "PromptV1",
		},
		DPairVerdict: internal.VerdictOK,
		DiscardRate:  1.0 / 6.0,
		DiscardWarn:  false,
	}

	out := mustRenderMeasure(t, mr)

	if strings.Contains(out, "discard rate") {
		t.Fatalf("must not print the discard-rate warning when DiscardWarn=false, got:\n%s", out)
	}

	wantConfigLines := []string{
		"Instrument config (REQ-MSR-04):",
		"  backend:        ollama",
		"  model:          qwen3-coder:30b",
		"  temperature:    1.00",
		"  samples (N):    5",
		"  think:          false",
		"  num_ctx:        8192",
		"  num_predict:    2048",
		"  sim_threshold:  0.95",
		"  prompt:         PromptV1 (5 bytes)",
	}
	for _, line := range wantConfigLines {
		if !strings.Contains(out, line) {
			t.Fatalf("want instrument-config line %q in output, got:\n%s", line, out)
		}
	}

	wantDPair := fmt.Sprintf("  D_pair:   %.2f  [%s]%s(95%% CI [%.2f, %.2f]; threshold %.2f, mean sim %.2f, N=%d valid, %d discarded)",
		mr.Dispersion.DPair, internal.VerdictOK, pad(internal.VerdictOK), mr.Dispersion.DPairCILow, mr.Dispersion.DPairCIHigh, testThresholds.DPairMax, mr.Dispersion.MeanSim, mr.Dispersion.N, mr.Dispersion.Discarded)
	if !strings.Contains(out, wantDPair) {
		t.Fatalf("want D_pair line %q, got output:\n%s", wantDPair, out)
	}
	if !strings.Contains(out, "[ok]") {
		t.Fatalf("want [ok] verdict label, got:\n%s", out)
	}

	wantH := fmt.Sprintf("  H:        %.2f  bits (ordinal signal only, not gated)", mr.Dispersion.H)
	wantHNorm := fmt.Sprintf("  H_norm:   %.2f  (ordinal signal only, not gated)", mr.Dispersion.HNorm)
	if !strings.Contains(out, wantH) {
		t.Fatalf("want H line %q, got output:\n%s", wantH, out)
	}
	if !strings.Contains(out, wantHNorm) {
		t.Fatalf("want H_norm line %q, got output:\n%s", wantHNorm, out)
	}
}

// TestRenderMeasurePromptVersionIsNotHardcoded guards issue #56: the
// printed prompt-version label must come from cfg.PromptVersion, not a
// hardcoded "PromptV1" literal at the print site — otherwise the report
// would keep claiming "PromptV1" even after a hypothetical PromptV2 lands,
// silently violating REQ-MSR-04's reproducibility requirement. Using a
// fixture value distinct from any real prompt constant name proves the
// print path threads the config field through unmodified.
func TestRenderMeasurePromptVersionIsNotHardcoded(t *testing.T) {
	mr := MeasureResult{
		Dispersion: internal.DispersionResult{N: 5, MeanSim: 0.82, DPair: 0.18},
		Config: internal.InstrumentConfig{
			Backend:       "ollama",
			Model:         "qwen3-coder:30b",
			Prompt:        "xyz",
			PromptVersion: "TotallyDifferentXYZ",
		},
		DPairVerdict: internal.VerdictOK,
	}

	out := mustRenderMeasure(t, mr)

	want := "  prompt:         TotallyDifferentXYZ (3 bytes)"
	if !strings.Contains(out, want) {
		t.Fatalf("want prompt line %q (from cfg.PromptVersion, not a hardcoded literal), got output:\n%s", want, out)
	}
	if strings.Contains(out, "PromptV1") {
		t.Fatalf("prompt line must not contain a hardcoded \"PromptV1\" literal when cfg.PromptVersion is something else, got:\n%s", out)
	}
}

// TestRenderMeasureBlockVerdict mirrors the OK case but with a D_pair value
// over threshold, asserting the [block] label and correct values.
func TestRenderMeasureBlockVerdict(t *testing.T) {
	mr := MeasureResult{
		Dispersion: internal.DispersionResult{
			N:           4,
			Discarded:   0,
			MeanSim:     0.55,
			DPair:       0.45,
			DPairCILow:  0.30,
			DPairCIHigh: 0.60,
			Clusters:    4,
			SimThresh:   0.95,
			H:           2.0,
			HNorm:       1.0,
		},
		Config: internal.InstrumentConfig{
			Backend:       "ollama",
			Model:         "qwen3-coder:30b",
			Temperature:   1.0,
			Samples:       4,
			Think:         true,
			NumCtx:        8192,
			NumPredict:    2048,
			SimThreshold:  0.95,
			Prompt:        "abcde",
			PromptVersion: "PromptV1",
		},
		DPairVerdict: internal.VerdictBlock,
		DiscardRate:  0,
		DiscardWarn:  false,
	}

	out := mustRenderMeasure(t, mr)

	wantDPair := fmt.Sprintf("  D_pair:   %.2f  [%s]%s(95%% CI [%.2f, %.2f]; threshold %.2f, mean sim %.2f, N=%d valid, %d discarded)",
		mr.Dispersion.DPair, internal.VerdictBlock, pad(internal.VerdictBlock), mr.Dispersion.DPairCILow, mr.Dispersion.DPairCIHigh, testThresholds.DPairMax, mr.Dispersion.MeanSim, mr.Dispersion.N, mr.Dispersion.Discarded)
	if !strings.Contains(out, wantDPair) {
		t.Fatalf("want D_pair line %q, got output:\n%s", wantDPair, out)
	}
	if !strings.Contains(out, "[block]") {
		t.Fatalf("want [block] verdict label, got:\n%s", out)
	}

	wantH := fmt.Sprintf("  H:        %.2f  bits (ordinal signal only, not gated)", mr.Dispersion.H)
	wantHNorm := fmt.Sprintf("  H_norm:   %.2f  (ordinal signal only, not gated)", mr.Dispersion.HNorm)
	if !strings.Contains(out, wantH) {
		t.Fatalf("want H line %q, got output:\n%s", wantH, out)
	}
	if !strings.Contains(out, wantHNorm) {
		t.Fatalf("want H_norm line %q, got output:\n%s", wantHNorm, out)
	}
}

// TestRenderMeasureSkippedVerdict covers the early-return branch for fewer
// than 2 valid samples: D_pair, H and H_norm must all render as the
// "—  [skipped]" form, never as a misleading numeric "D_pair: 0.00".
func TestRenderMeasureSkippedVerdict(t *testing.T) {
	mr := MeasureResult{
		Dispersion: internal.DispersionResult{
			N:         1,
			Discarded: 4,
		},
		Config: internal.InstrumentConfig{
			Backend:       "ollama",
			Model:         "qwen3-coder:30b",
			Temperature:   1.0,
			Samples:       5,
			Think:         false,
			NumCtx:        8192,
			NumPredict:    2048,
			SimThreshold:  0.95,
			Prompt:        "abcde",
			PromptVersion: "PromptV1",
		},
		DPairVerdict: internal.VerdictSkipped,
		// DiscardRate is deliberately above internal.DiscardWarnThreshold
		// while DiscardWarn is false: this fixture only exercises the
		// VerdictSkipped rendering branch of RenderMeasure, not the
		// discard-warning branch (covered separately by
		// TestRenderMeasureDiscardWarningVisibility) — a real
		// runMeasureWithGenerator run would set DiscardWarn=true here.
		DiscardRate: 0.8,
		DiscardWarn: false,
	}

	out := mustRenderMeasure(t, mr)

	wantDPair := fmt.Sprintf("  D_pair:   —     [%s]%s(only %d valid sample(s); need >=2 to compute pairwise similarity)",
		internal.VerdictSkipped, pad(internal.VerdictSkipped), mr.Dispersion.N)
	wantH := fmt.Sprintf("  H:        —     [%s]%s(ordinal signal only, not gated)", internal.VerdictSkipped, pad(internal.VerdictSkipped))
	wantHNorm := fmt.Sprintf("  H_norm:   —     [%s]%s(ordinal signal only, not gated)", internal.VerdictSkipped, pad(internal.VerdictSkipped))

	if !strings.Contains(out, wantDPair) {
		t.Fatalf("want skipped D_pair line %q, got output:\n%s", wantDPair, out)
	}
	if !strings.Contains(out, wantH) {
		t.Fatalf("want skipped H line %q, got output:\n%s", wantH, out)
	}
	if !strings.Contains(out, wantHNorm) {
		t.Fatalf("want skipped H_norm line %q, got output:\n%s", wantHNorm, out)
	}
	if strings.Contains(out, "D_pair:   0.00") {
		t.Fatalf("skipped verdict must not render a numeric D_pair value, got:\n%s", out)
	}
	if strings.Contains(out, "H:        0.00") {
		t.Fatalf("skipped verdict must not render a numeric H value, got:\n%s", out)
	}
	if strings.Contains(out, "H_norm:   0.00") {
		t.Fatalf("skipped verdict must not render a numeric H_norm value, got:\n%s", out)
	}
}

// mustRenderReport renders r and fails the test on any write error, mirroring
// mustRenderMeasure's style.
func mustRenderReport(t *testing.T, r Report) string {
	t.Helper()
	var buf strings.Builder
	if err := RenderReport(&buf, r, testThresholds); err != nil {
		t.Fatalf("RenderReport: %v", err)
	}
	return buf.String()
}

// TestRenderReportDeterministicOnly covers gate's Measure == nil path
// (REQ-GATE-02, no instrument resolved): the output must be the check
// content plus the exit-code line, and must NOT print any measure-specific
// content (instrument config, D_pair/H/H_norm, or the warning lines) since
// the stochastic layer was never attempted.
func TestRenderReportDeterministicOnly(t *testing.T) {
	r := Report{
		Check: CheckResult{
			KD:        internal.KDriftResult{Requirements: 2, Hanging: 0, Value: 0},
			DC:        internal.DConstResult{ConstraintMarkers: 4, ProseTokens: 2, Value: 4.0 / 6.0},
			KDVerdict: internal.VerdictOK,
			DCVerdict: internal.VerdictOK,
		},
		Measure:  nil,
		Verdict:  internal.VerdictOK,
		ExitCode: 0,
	}
	out := mustRenderReport(t, r)

	if !strings.Contains(out, "K_drift:  0.00  [ok]") {
		t.Fatalf("want K_drift line, got:\n%s", out)
	}
	if !strings.Contains(out, "D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)") {
		t.Fatalf("want the unmeasured D_pair placeholder line (RenderCheck's own text) for Measure == nil, got:\n%s", out)
	}
	if strings.Contains(out, "Instrument config") {
		t.Fatalf("must not print the instrument-config block when Measure == nil, got:\n%s", out)
	}
	if !strings.Contains(out, "\nexit code: 0 (gates pass)\n") {
		t.Fatalf("want the exit-code line, got:\n%s", out)
	}
}

// TestRenderReportFull covers gate's full path (both layers ran): the
// output must contain both the check content and the measure content, and
// must NOT print the unmeasured-D_pair placeholder line (the real D_pair
// value is available and takes its place).
func TestRenderReportFull(t *testing.T) {
	mr := MeasureResult{
		Dispersion: internal.DispersionResult{
			N: 5, MeanSim: 0.82, DPair: 0.18, H: 1.37, HNorm: 0.59,
		},
		Config:       internal.InstrumentConfig{Backend: "ollama", Model: "qwen3-coder:30b", PromptVersion: "PromptV1"},
		DPairVerdict: internal.VerdictOK,
	}
	r := Report{
		Check: CheckResult{
			KD:        internal.KDriftResult{Requirements: 2, Hanging: 0, Value: 0},
			DC:        internal.DConstResult{ConstraintMarkers: 4, ProseTokens: 2, Value: 4.0 / 6.0},
			KDVerdict: internal.VerdictOK,
			DCVerdict: internal.VerdictOK,
		},
		Measure:  &mr,
		Verdict:  internal.VerdictOK,
		ExitCode: 0,
	}
	out := mustRenderReport(t, r)

	if !strings.Contains(out, "K_drift:  0.00  [ok]") {
		t.Fatalf("want K_drift line, got:\n%s", out)
	}
	if strings.Contains(out, "run `tumanomir measure` with an instrument") {
		t.Fatalf("must not print the unmeasured-D_pair placeholder when Measure is present, got:\n%s", out)
	}
	if !strings.Contains(out, "Instrument config (REQ-MSR-04):") {
		t.Fatalf("want the instrument-config block when Measure is present, got:\n%s", out)
	}
	if !strings.Contains(out, "D_pair:   0.18  [ok]") {
		t.Fatalf("want the real D_pair line, got:\n%s", out)
	}
	if !strings.Contains(out, "\nexit code: 0 (gates pass)\n") {
		t.Fatalf("want the exit-code line, got:\n%s", out)
	}
}

// TestRenderReportBlockExitCode guards the exit-code-1 rendering branch.
func TestRenderReportBlockExitCode(t *testing.T) {
	r := Report{
		Check: CheckResult{
			KD:        internal.KDriftResult{Requirements: 2, Hanging: 1, Value: 0.5},
			DC:        internal.DConstResult{ConstraintMarkers: 4, ProseTokens: 2, Value: 4.0 / 6.0},
			KDVerdict: internal.VerdictBlock,
			DCVerdict: internal.VerdictOK,
		},
		Measure:  nil,
		Verdict:  internal.VerdictBlock,
		ExitCode: 1,
	}
	out := mustRenderReport(t, r)
	if !strings.Contains(out, "\nexit code: 1 (gate failed)\n") {
		t.Fatalf("want the exit-code-1 line, got:\n%s", out)
	}
}
