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

// RenderCheck writes the deterministic layer's report (K_drift, D_const) to
// w, including the hanging-requirement-ID list, per REQ-OUT-01. It does not
// decide or print the exit code — the caller (cmd/tumanomir's runCheck)
// stays responsible for exit-code semantics, keeping this package free of
// process-exit concerns.
func RenderCheck(w io.Writer, r CheckResult, th internal.Thresholds) {
	if r.KDVerdict == internal.VerdictSkipped {
		fmt.Fprintf(w, "  K_drift:  —     [n/a]%s(no [REQ-*] tags found)\n", pad(r.KDVerdict))
	} else {
		fmt.Fprintf(w, "  K_drift:  %.2f  [%s]%s(threshold %.2f, %d/%d requirements untraced)\n",
			r.KD.Value, r.KDVerdict, pad(r.KDVerdict), th.KDriftMax, r.KD.Hanging, r.KD.Requirements)
	}
	fmt.Fprintf(w, "  D_const:  %.2f  [%s]%s(threshold %.2f, %d markers / %d prose tokens)\n",
		r.DC.Value, r.DCVerdict, pad(r.DCVerdict), th.DConstMin, r.DC.ConstraintMarkers, r.DC.ProseTokens)
	fmt.Fprintf(w, "  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)\n")

	for _, id := range r.KD.HangingIDs {
		fmt.Fprintf(w, "    hanging: %s\n", id)
	}
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
func RenderMeasure(w io.Writer, r MeasureResult, th internal.Thresholds) {
	cfg := r.Config

	if r.DiscardWarn {
		fmt.Fprintf(w, "⚠ discard rate: %.0f%% (%d/%d generations invalid) — exceeds the %.0f%% hypothesis threshold (REQ-MSR-05); results may be unreliable\n\n",
			r.DiscardRate*100, r.Dispersion.Discarded, r.Dispersion.Discarded+r.Dispersion.N, internal.DiscardWarnThreshold*100)
	}

	if r.Truncated > 0 {
		fmt.Fprintf(w, "⚠ %d/%d accepted generations had done_reason=length (truncated by num_predict) — their AST may not reflect the model's full intended output; consider raising --num-predict\n\n",
			r.Truncated, r.Dispersion.N)
	}

	if r.PromptUnderestimated > 0 {
		fmt.Fprintf(w, "⚠ %d generation(s) had an actual prompt-token count over %.1fx the preflight estimate — the byte/3 heuristic under-counts non-ASCII (e.g. Cyrillic) prompts and may not have caught a real truncation risk; verify --num-ctx has enough headroom\n\n",
			r.PromptUnderestimated, internal.PromptEstimateDivergenceFactor)
	}

	fmt.Fprintln(w, "Instrument config (REQ-MSR-04):")
	fmt.Fprintf(w, "  backend:        %s\n", cfg.Backend)
	fmt.Fprintf(w, "  model:          %s\n", cfg.Model)
	fmt.Fprintf(w, "  temperature:    %.2f\n", cfg.Temperature)
	fmt.Fprintf(w, "  samples (N):    %d\n", cfg.Samples)
	fmt.Fprintf(w, "  think:          %t\n", cfg.Think)
	fmt.Fprintf(w, "  num_ctx:        %d\n", cfg.NumCtx)
	fmt.Fprintf(w, "  num_predict:    %d\n", cfg.NumPredict)
	fmt.Fprintf(w, "  sim_threshold:  %.2f\n", cfg.SimThreshold)
	fmt.Fprintf(w, "  prompt:         %s (%d bytes)\n\n", cfg.PromptVersion, len(cfg.Prompt))

	if r.DPairVerdict == internal.VerdictSkipped {
		fmt.Fprintf(w, "  D_pair:   —     [%s]%s(only %d valid sample(s); need >=2 to compute pairwise similarity)\n",
			internal.VerdictSkipped, pad(internal.VerdictSkipped), r.Dispersion.N)
		fmt.Fprintf(w, "  H:        —     [%s]%s(ordinal signal only, not gated)\n", internal.VerdictSkipped, pad(internal.VerdictSkipped))
		fmt.Fprintf(w, "  H_norm:   —     [%s]%s(ordinal signal only, not gated)\n", internal.VerdictSkipped, pad(internal.VerdictSkipped))
		return
	}

	fmt.Fprintf(w, "  D_pair:   %.2f  [%s]%s(threshold %.2f, mean sim %.2f, N=%d valid, %d discarded)\n",
		r.Dispersion.DPair, r.DPairVerdict, pad(r.DPairVerdict), th.DPairMax, r.Dispersion.MeanSim, r.Dispersion.N, r.Dispersion.Discarded)
	fmt.Fprintf(w, "  H:        %.2f  bits (ordinal signal only, not gated)\n", r.Dispersion.H)
	fmt.Fprintf(w, "  H_norm:   %.2f  (ordinal signal only, not gated)\n", r.Dispersion.HNorm)
}
