package report

import (
	"fmt"
	"io"

	"github.com/valpere/tumanomir/internal"
)

// pad aligns verdict columns for ok/warn/block widths.
func pad(v internal.Verdict) string {
	switch v {
	case internal.VerdictOK:
		return "     "
	case internal.VerdictWarn:
		return "   "
	case internal.VerdictSkipped:
		return "    "
	default:
		return "  "
	}
}

// errWriter wraps an io.Writer and records the first write error
// encountered, silently no-oping subsequent writes afterward — the standard
// pattern for a sequence of writes where only the first failure matters
// (Rob Pike, "Errors are values": https://go.dev/blog/errors-are-values).
// PURPOSE: RenderCheck/RenderMeasure accept a caller-supplied io.Writer —
// unlike a literal os.Stdout/os.Stderr, it could be any Writer (a file, a
// pipe, a network connection), so a write failure is a real possibility
// worth propagating to the caller rather than silently discarding, without
// an error check after every single Fprintf/Fprintln call in the render body.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) printf(format string, a ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, a...)
}

func (ew *errWriter) println(a ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintln(ew.w, a...)
}

// RenderCheck writes the deterministic layer's report (K_drift, D_const) to
// w, including the hanging-requirement-ID list, per REQ-OUT-01. It does not
// decide or print the exit code — the caller (cmd/tumanomir's runCheck)
// stays responsible for exit-code semantics, keeping this package free of
// process-exit concerns. Returns the first error writing to w, if any.
func RenderCheck(w io.Writer, r CheckResult, th internal.Thresholds) error {
	ew := &errWriter{w: w}

	if r.KDVerdict == internal.VerdictSkipped {
		ew.printf("  K_drift:  —     [n/a]%s(no [REQ-*] tags found)\n", pad(r.KDVerdict))
	} else {
		ew.printf("  K_drift:  %.2f  [%s]%s(threshold %.2f, %d/%d requirements untraced)\n",
			r.KD.Value, r.KDVerdict, pad(r.KDVerdict), th.KDriftMax, r.KD.Hanging, r.KD.Requirements)
	}
	ew.printf("  D_const:  %.2f  [%s]%s(threshold %.2f, %d markers / %d prose tokens)\n",
		r.DC.Value, r.DCVerdict, pad(r.DCVerdict), th.DConstMin, r.DC.ConstraintMarkers, r.DC.ProseTokens)
	ew.printf("  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)\n")

	for _, id := range r.KD.HangingIDs {
		ew.printf("    hanging: %s\n", id)
	}

	return ew.err
}

// RenderMeasure writes REQ-MSR-04's instrument config, the discard-rate and
// truncation warnings (if triggered), and the D_pair/H/H_norm lines to w. H
// and H_norm are always printed as ordinal/advisory signals — they never
// gate, per the methodological invariant in CLAUDE.md. It does not decide or
// print the exit code — the caller (cmd/tumanomir's runMeasureImpl) stays
// responsible for exit-code semantics.
//
// The discard-rate warning (REQ-MSR-05) and the truncation warning
// (REQ-MSR-06) are printed as two separate lines rather than folded
// together: they flag two distinct failure modes — generations that never
// became valid Go at all (discarded, excluded from N) vs. generations that
// parsed as valid Go but were cut off by num_predict (accepted into N, but
// their AST may not reflect the model's full intended output). Merging the
// two would blur which failure mode a reader needs to act on.
// Returns the first error writing to w, if any.
func RenderMeasure(w io.Writer, r MeasureResult, th internal.Thresholds) error {
	ew := &errWriter{w: w}
	cfg := r.Config

	if r.DiscardWarn {
		ew.printf("⚠ discard rate: %.0f%% (%d/%d generations invalid) — exceeds the %.0f%% hypothesis threshold (REQ-MSR-05); results may be unreliable\n\n",
			r.DiscardRate*100, r.Dispersion.Discarded, r.Dispersion.Discarded+r.Dispersion.N, internal.DiscardWarnThreshold*100)
	}

	if r.Truncated > 0 {
		ew.printf("⚠ %d/%d accepted generations had done_reason=length (truncated by num_predict) — their AST may not reflect the model's full intended output; consider raising --num-predict\n\n",
			r.Truncated, r.Dispersion.N)
	}

	if r.PromptUnderestimated > 0 {
		ew.printf("⚠ %d generation(s) had an actual prompt-token count over %.1fx the preflight estimate — the byte/3 heuristic under-counts non-ASCII (e.g. Cyrillic) prompts and may not have caught a real truncation risk; verify --num-ctx has enough headroom\n\n",
			r.PromptUnderestimated, internal.PromptEstimateDivergenceFactor)
	}

	ew.println("Instrument config (REQ-MSR-04):")
	ew.printf("  backend:        %s\n", cfg.Backend)
	ew.printf("  model:          %s\n", cfg.Model)
	ew.printf("  temperature:    %.2f\n", cfg.Temperature)
	ew.printf("  samples (N):    %d\n", cfg.Samples)
	ew.printf("  think:          %t\n", cfg.Think)
	ew.printf("  num_ctx:        %d\n", cfg.NumCtx)
	ew.printf("  num_predict:    %d\n", cfg.NumPredict)
	ew.printf("  sim_threshold:  %.2f\n", cfg.SimThreshold)
	ew.printf("  prompt:         %s (%d bytes)\n\n", cfg.PromptVersion, len(cfg.Prompt))

	if r.DPairVerdict == internal.VerdictSkipped {
		ew.printf("  D_pair:   —     [%s]%s(only %d valid sample(s); need >=2 to compute pairwise similarity)\n",
			internal.VerdictSkipped, pad(internal.VerdictSkipped), r.Dispersion.N)
		ew.printf("  H:        —     [%s]%s(ordinal signal only, not gated)\n", internal.VerdictSkipped, pad(internal.VerdictSkipped))
		ew.printf("  H_norm:   —     [%s]%s(ordinal signal only, not gated)\n", internal.VerdictSkipped, pad(internal.VerdictSkipped))
		return ew.err
	}

	ew.printf("  D_pair:   %.2f  [%s]%s(threshold %.2f, mean sim %.2f, N=%d valid, %d discarded)\n",
		r.Dispersion.DPair, r.DPairVerdict, pad(r.DPairVerdict), th.DPairMax, r.Dispersion.MeanSim, r.Dispersion.N, r.Dispersion.Discarded)
	ew.printf("  H:        %.2f  bits (ordinal signal only, not gated)\n", r.Dispersion.H)
	ew.printf("  H_norm:   %.2f  (ordinal signal only, not gated)\n", r.Dispersion.HNorm)

	return ew.err
}
