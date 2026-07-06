package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
)

var testThresholds = internal.DefaultThresholds()

func TestRenderMeasureDiscardWarningVisibility(t *testing.T) {
	warnMR := MeasureResult{
		Dispersion:  internal.DispersionResult{N: 2, Discarded: 8},
		Config:      internal.InstrumentConfig{Backend: "ollama", Model: "test"},
		DiscardRate: 0.8,
		DiscardWarn: true,
	}
	var buf bytes.Buffer
	if err := RenderMeasure(&buf, warnMR, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "discard rate") {
		t.Fatalf("want a discard-rate warning line for DiscardWarn=true, got:\n%s", out)
	}

	noWarnMR := warnMR
	noWarnMR.DiscardWarn = false
	buf.Reset()
	if err := RenderMeasure(&buf, noWarnMR, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out = buf.String()
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
	var buf bytes.Buffer
	if err := RenderMeasure(&buf, truncMR, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out := buf.String()
	wantWarn := "3/10 accepted generations had done_reason=length"
	if !strings.Contains(out, wantWarn) {
		t.Fatalf("want a truncation warning line containing %q for Truncated=3, got:\n%s", wantWarn, out)
	}

	noTruncMR := truncMR
	noTruncMR.Truncated = 0
	buf.Reset()
	if err := RenderMeasure(&buf, noTruncMR, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out = buf.String()
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
	var buf bytes.Buffer
	if err := RenderMeasure(&buf, underMR, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out := buf.String()
	wantWarn := "4 generation(s) had an actual prompt-token count"
	if !strings.Contains(out, wantWarn) {
		t.Fatalf("want a prompt-underestimate warning line containing %q for PromptUnderestimated=4, got:\n%s", wantWarn, out)
	}

	noneMR := underMR
	noneMR.PromptUnderestimated = 0
	buf.Reset()
	if err := RenderMeasure(&buf, noneMR, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out = buf.String()
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
			N:         5,
			Discarded: 1,
			MeanSim:   0.82,
			DPair:     0.18,
			Clusters:  2,
			SimThresh: 0.95,
			H:         1.37,
			HNorm:     0.59,
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

	var buf bytes.Buffer
	if err := RenderMeasure(&buf, mr, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out := buf.String()

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

	wantDPair := fmt.Sprintf("  D_pair:   %.2f  [%s]%s(threshold %.2f, mean sim %.2f, N=%d valid, %d discarded)",
		mr.Dispersion.DPair, internal.VerdictOK, pad(internal.VerdictOK), testThresholds.DPairMax, mr.Dispersion.MeanSim, mr.Dispersion.N, mr.Dispersion.Discarded)
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

	var buf bytes.Buffer
	if err := RenderMeasure(&buf, mr, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out := buf.String()

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
			N:         4,
			Discarded: 0,
			MeanSim:   0.55,
			DPair:     0.45,
			Clusters:  4,
			SimThresh: 0.95,
			H:         2.0,
			HNorm:     1.0,
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

	var buf bytes.Buffer
	if err := RenderMeasure(&buf, mr, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out := buf.String()

	wantDPair := fmt.Sprintf("  D_pair:   %.2f  [%s]%s(threshold %.2f, mean sim %.2f, N=%d valid, %d discarded)",
		mr.Dispersion.DPair, internal.VerdictBlock, pad(internal.VerdictBlock), testThresholds.DPairMax, mr.Dispersion.MeanSim, mr.Dispersion.N, mr.Dispersion.Discarded)
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

	var buf bytes.Buffer
	if err := RenderMeasure(&buf, mr, testThresholds); err != nil {
		t.Fatalf("RenderMeasure: %v", err)
	}
	out := buf.String()

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
