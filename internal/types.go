// Package internal holds types shared across tumanomir packages: the
// Verdict/Thresholds gate vocabulary, and the four measurement result
// structs (KDriftResult, DConstResult, DispersionResult, InstrumentConfig)
// that every metric package and cmd/tumanomir depend on. Kept here rather
// than in the packages that produce them so that cmd/tumanomir can
// reference all four without internal/metrics, internal/dispersion, and
// internal/instrument needing to import each other.
package internal

// Verdict classifies a single metric's value against its threshold, and
// doubles as the exit-code driver: cmd/tumanomir maps VerdictBlock to a
// non-zero process exit, everything else to a clean run (see
// cmd/tumanomir/main.go's runCheck/runMeasureImpl).
type Verdict string

const (
	// VerdictOK means the metric is within its threshold — no action needed.
	VerdictOK Verdict = "ok"
	// VerdictWarn means the metric crossed an advisory threshold (D_const,
	// H_norm) — reported, but never gates the exit code.
	VerdictWarn Verdict = "warn"
	// VerdictBlock means a gating metric (K_drift, D_pair) crossed its
	// threshold — the only Verdict that produces a non-zero exit code.
	VerdictBlock Verdict = "block"
	// VerdictSkipped means the metric could not be meaningfully computed
	// (e.g. K_drift on a spec with zero [REQ-*] tags) — distinct from a
	// genuine 0.00 pass, so the report doesn't misrepresent "no signal" as
	// "clean signal".
	VerdictSkipped Verdict = "skipped"
)

// Thresholds are the gate boundaries for every metric tumanomir computes.
// The zero value is meaningless (all-zero thresholds would gate on
// anything); always obtain a populated value via DefaultThresholds and
// override individual fields from CLI flags as needed. Values are
// hypotheses from the source article, not calibrated constants — CLAUDE.md
// requires a docs/requirements.md update before changing any of them.
type Thresholds struct {
	// KDriftMax is the maximum acceptable fraction of requirements without
	// a trace edge (K_drift gate, deterministic layer).
	KDriftMax float64
	// DConstMin is the minimum acceptable lexical constraint density
	// (D_const, advisory-only — never gates the exit code).
	DConstMin float64
	// DPairMax is the maximum acceptable 1-minus-mean-pairwise-AST-similarity
	// (D_pair gate, stochastic layer).
	DPairMax float64
}

// DefaultThresholds returns the source article's starting-point hypothesis
// values (0.20 / 0.35 / 0.30) — not empirically calibrated for any specific
// team's spec corpus. Callers are expected to override via CLI flags once
// they have enough measured history to know what "normal" looks like for
// their own specs.
func DefaultThresholds() Thresholds {
	return Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30}
}

// DiscardWarnThreshold is REQ-MSR-05's hypothesis discard-rate threshold
// above which the measure report must flag the run as potentially
// unreliable. Stated here as a hypothesis, not a calibrated constant, the
// same treatment given to the 0.20/0.35/0.30 thresholds in
// DefaultThresholds.
const DiscardWarnThreshold = 0.40

// PromptEstimateDivergenceFactor flags a generation whose actual
// PromptEvalCount exceeds the pre-flight byte/3 estimate by more than this
// multiple — a signal the estimate under-counted (e.g. non-ASCII input),
// not a calibrated constant. See issue #57.
const PromptEstimateDivergenceFactor = 1.5

// KDriftResult is the deterministic-layer traceability metric: what
// fraction of a spec's requirements have no downstream trace edge at all.
// Produced by internal/metrics.KDrift from a pure byte scan — no LLM, no
// network — so it's cheap enough to run on every commit (see
// cmd/tumanomir's `check` subcommand and the Makefile's `dogfood` target).
type KDriftResult struct {
	// Requirements is the total count of well-formed [REQ-*] tags found.
	Requirements int
	// Hanging is how many of those requirements have no
	// "-> [FUN-*|LOG-*|PHY-*]" edge anywhere before the next [REQ-*] tag.
	Hanging int
	// HangingIDs holds the identifiers of every hanging requirement, so a
	// failing report can point at exactly which requirements need tracing
	// rather than just a bare count.
	HangingIDs []string
	// Value is Hanging/Requirements — the fraction compared against
	// Thresholds.KDriftMax. Left at its zero value (0) when Requirements
	// is 0, since a 0/0 ratio has no meaningful "drift" to report (see
	// VerdictSkipped, which is what a zero-requirement spec renders as).
	Value float64
}

// DConstResult is the lexical constraint-density proxy for D_const: a
// cheap textual approximation of "how much of this spec's prose is
// actually load-bearing constraint, versus filler." It is intentionally
// not a structural/graph-based measure (see docs/roadmap.md's RFLP-graph
// exploratory item) — this is the v0.1 lexical stand-in.
type DConstResult struct {
	// ConstraintMarkers counts recognized constraint-bearing tokens
	// (@schema/@constraint annotations, etc.) in the scanned spec.
	ConstraintMarkers int
	// ProseTokens counts ordinary word tokens outside constraint markers —
	// the denominator's other half.
	ProseTokens int
	// Value is ConstraintMarkers / (ConstraintMarkers + ProseTokens),
	// compared against Thresholds.DConstMin. Left at its zero value when
	// both counts are 0 (an empty document has no density to report).
	Value float64
}

// InstrumentConfig is the full stochastic-layer instrument configuration.
// Measurements are instrument-relative and meaningless without it, so the
// configuration must be fixed per run and printed in the report (REQ-MSR-04).
// See docs/requirements.md @schema InstrumentConfig for field defaults and
// constraints; no Default constructor is provided here because SimThreshold
// has no calibrated default wired into code yet — callers must supply one.
type InstrumentConfig struct {
	// Backend selects the Generator implementation; v0.1 supports only
	// "ollama" (see internal/instrument.NewOllama).
	Backend string
	// Model is the backend-specific model identifier, e.g. "qwen3-coder:30b".
	Model string
	// Temperature is the sampling temperature passed to the backend —
	// higher values increase generation-to-generation variance, which is
	// exactly what D_pair is trying to measure, so this is not a knob to
	// "tune for better output," it's part of the instrument's identity.
	Temperature float64
	// Samples is how many generations to request per spec (N in D_pair's
	// "1 - mean pairwise AST similarity of N generations").
	Samples int
	// Think enables reasoning-model "thinking" mode. Per REQ-MSR-06 this
	// must be false for reasoning models in practice — true here would let
	// think-tokens leak into the measured output and corrupt the AST
	// feature extraction downstream.
	Think bool
	// NumCtx is the context window size in tokens; must exceed the actual
	// prompt token count or the backend will silently truncate the prompt —
	// a measurement-integrity bug, not a performance knob (REQ-MSR-06).
	NumCtx int
	// NumPredict is the max generated tokens; must exceed the natural
	// output length or generations get cut off mid-artifact, which would
	// masquerade as a parse failure rather than what it actually is (an
	// instrument misconfiguration).
	NumPredict int
	// SimThreshold is the single-linkage clustering distance threshold used
	// to compute H (cluster entropy) from the pairwise similarity matrix —
	// it only affects H/HNorm, never MeanSim/DPair (see
	// internal/dispersion.Analyze).
	SimThreshold float64
	// Prompt is the exact instrument.PromptV1 (or later version) text sent,
	// verbatim. It holds the constant's *value* (a plain string), not a
	// reference to it: internal/types.go (package internal) must not import
	// internal/instrument (which already imports internal), so runMeasure
	// is expected to populate this field with instrument.PromptV1.
	Prompt string
	// PromptVersion identifies which named prompt constant Prompt's value
	// came from (e.g. instrument.PromptVersion == "PromptV1"), so the
	// report can print the version without a hardcoded literal at the
	// print site — printing a literal would silently lie once a future
	// PromptV2 lands and this field isn't updated alongside it.
	PromptVersion string
}

// DispersionResult is the stochastic-layer output (D_pair, H/HNorm) for one
// spec measured under one fixed InstrumentConfig. Produced by
// internal/dispersion.Analyze from N generated Go artifacts.
type DispersionResult struct {
	// N is the count of valid (successfully parsed) samples actually used
	// in the computation — may be less than InstrumentConfig.Samples if
	// some generations were discarded after exhausting retries.
	N int
	// Discarded is how many generation slots never produced a valid sample
	// even after retrying (REQ-MSR-05's invalid-rate signal) — reported,
	// never hidden, since a high discard rate itself indicates instrument
	// or prompt trouble that would otherwise masquerade as "clean" D_pair.
	Discarded int
	// MeanSim is the mean pairwise cosine similarity across all N valid
	// samples' AST feature vectors. Threshold-independent — computed
	// before any clustering happens.
	MeanSim float64
	// DPair is 1 - MeanSim: the working stochastic-layer gate metric,
	// compared against Thresholds.DPairMax.
	DPair float64
	// DPairCILow and DPairCIHigh are the 2.5th/97.5th percentile bounds of
	// a bootstrap confidence interval around DPair (REQ-MSR-07) — advisory
	// only, like H/HNorm below; DPairVerdict still compares the point
	// estimate DPair against DPairMax, never these bounds. Both stay 0
	// when N<2, same as DPair does, since there is no pair to bootstrap
	// from.
	DPairCILow, DPairCIHigh float64
	// Clusters is the single-linkage cluster count at SimThresh — an
	// ordinal signal ("one cluster or many"), not a gate.
	Clusters int
	// SimThresh is the clustering distance threshold this result's
	// Clusters/H/HNorm were computed at, carried alongside the result so a
	// report can print which threshold produced them.
	SimThresh float64
	// H is the Shannon entropy over cluster sizes, in bits. Saturates at
	// log2(N) for small N, so it is not directly comparable across
	// different N by itself — see HNorm for the comparable form.
	H float64
	// HNorm is H / log2(N), normalized into [0,1] — an ordinal signal only
	// ("one cluster or many"), reported but never gated (D_pair is the
	// only stochastic-layer gate; see CLAUDE.md's methodological
	// invariants).
	HNorm float64
}
