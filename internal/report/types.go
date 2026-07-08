// Package report renders CheckResult/MeasureResult into human-readable TTY
// output (REQ-OUT-01). It depends only on internal — never on
// internal/metrics, internal/spec, internal/dispersion, or
// internal/instrument — so cmd/tumanomir's aggregation/measurement logic
// and this package's rendering stay decoupled.
package report

import "github.com/valpere/tumanomir/internal"

// CheckResult holds the deterministic layer's aggregated metric values and
// their gate verdicts, computed by cmd/tumanomir's aggregate.
type CheckResult struct {
	KD        internal.KDriftResult `json:"k_drift"`         // deterministic traceability metric, aggregated across all specs
	DC        internal.DConstResult `json:"d_const"`         // deterministic lexical constraint-density metric, aggregated
	KDVerdict internal.Verdict      `json:"k_drift_verdict"` // gates the exit code (VerdictBlock -> exit 1)
	DCVerdict internal.Verdict      `json:"d_const_verdict"` // VerdictOK or VerdictWarn; advisory only, never gates the exit code
}

// MeasureResult holds the stochastic layer's aggregated metric values,
// discard-rate warning state, and gate verdict.
type MeasureResult struct {
	// Dispersion is the raw D_pair/H/H_norm computation from
	// dispersion.Analyze over the run's surviving valid samples.
	Dispersion internal.DispersionResult `json:"dispersion"`
	// Config is the instrument configuration this run measured under —
	// printed verbatim in the report per REQ-MSR-04's instrument-relative
	// reporting requirement.
	Config internal.InstrumentConfig `json:"config"`
	// DPairVerdict gates the exit code (VerdictBlock -> exit 1); may also
	// be VerdictSkipped if too many discards left fewer than 2 valid
	// samples to compare.
	DPairVerdict internal.Verdict `json:"d_pair_verdict"`
	DiscardRate  float64          `json:"discard_rate"` // Discarded / (Discarded + N), 0 if no attempts made
	DiscardWarn  bool             `json:"discard_warn"` // DiscardRate > 0.40 (REQ-MSR-05's hypothesis threshold)
	// Truncated is the count of accepted (valid) generations with
	// DoneReason == instrument.DoneReasonLength (REQ-MSR-06). It lives
	// here rather than on internal.DispersionResult because it's an
	// instrument/generation-loop concept (which backend, why a
	// generation stopped), not something dispersion.Analyze's pure
	// AST-similarity computation has any business knowing about.
	Truncated int `json:"truncated"`
	// PromptUnderestimated is the count of generations (valid or not)
	// whose actual PromptEvalCount exceeded the pre-flight byte/3
	// estimate by more than internal.PromptEstimateDivergenceFactor — the
	// heuristic under-counts non-ASCII prompts, so this is a diagnostic
	// signal that the preflight's "errs toward refusing" guarantee may
	// not have held for this run (issue #57).
	PromptUnderestimated int `json:"prompt_underestimated"`
}

// Report is gate's unified result (@schema Report, REQ-GATE-01): it wraps
// CheckResult/MeasureResult rather than flattening their fields — flattening
// would duplicate every field CheckResult/MeasureResult already declare, a
// DRY violation gate's own additive design (issue #87) is meant to avoid.
type Report struct {
	// Check is always populated — gate always runs the deterministic layer.
	Check CheckResult `json:"check"`
	// Measure is nil when the stochastic layer was never attempted (no
	// instrument resolved from CLI flags or .tumanomir.yaml), distinct from
	// a Measure that ran and produced VerdictSkipped (tried, couldn't).
	// omitempty: JSON callers distinguish "never ran" (key absent) from a
	// populated Measure the same way Go callers distinguish it via nil.
	Measure *MeasureResult `json:"measure,omitempty"`
	// Verdict is the worst-case precedence (block > warn > skipped > ok)
	// over {Check.KDVerdict, Check.DCVerdict, Measure.DPairVerdict if
	// present} — see cmd/tumanomir's gateVerdict (REQ-GATE-03).
	Verdict internal.Verdict `json:"verdict"`
	// ExitCode is 1 iff Check.KDVerdict or Measure.DPairVerdict is
	// VerdictBlock; DCVerdict/H/H_norm never independently produce
	// ExitCode == 1 (REQ-CHK-06/REQ-MSR-02). 2 is reserved for execution
	// errors that never reach a rendered Report.
	ExitCode int `json:"exit_code"`
}
