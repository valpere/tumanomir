// Package calibrate implements the `calibrate` subcommand's corpus-based
// threshold-correlation tooling (REQ-CAL-*): reading a JSONL corpus of
// historical spec snapshots paired with pre-measured D_pair and
// caller-defined outcome scores, recomputing K_drift/D_const fresh from
// each snapshot, and reporting how well each metric actually correlates
// (Spearman rank correlation) with real downstream outcomes. Zero
// network, zero LLM: D_pair is read from the corpus, never re-measured
// (that would mean re-running an LLM instrument), so this package only
// ever depends on internal/spec and internal/metrics — both already
// network-free (see internal/nonetwork_test.go).
package calibrate

import "github.com/valpere/tumanomir/internal"

// Row is one historical corpus entry: an immutable spec snapshot's path,
// the opaque instrument identifier that produced DPair, and a
// caller-defined downstream outcome score (higher = worse pain). See
// docs/requirements.md REQ-CAL-01 for the on-disk JSONL schema this is
// parsed from.
type Row struct {
	SpecPath   string
	Instrument string
	DPair      float64
	Outcome    float64
}

// AnalyzedRow pairs a corpus Row with the K_drift/D_const values
// recomputed fresh from its pinned spec snapshot — DRY: the snapshot is
// the single source of truth for its own deterministic metrics. DPair
// itself is never recomputed here, since that would mean re-running an
// LLM instrument (see Row's own doc comment).
type AnalyzedRow struct {
	Row
	KDrift internal.KDriftResult
	DConst internal.DConstResult
}

// Range is one metric's min/mean/max within one half of a median-outcome
// split — reported to help a human eyeball a candidate threshold, never
// used to auto-select one (REQ-NFR-03: thresholds stay a human decision).
type Range struct {
	Min  float64
	Mean float64
	Max  float64
}

// MetricCorrelation is one metric's Spearman rank correlation against
// Outcome, plus its median-split distribution.
type MetricCorrelation struct {
	Name        string
	Correlation float64
	LowHalf     Range // this metric's values across rows at/below the median outcome
	HighHalf    Range // this metric's values across rows above the median outcome
}

// MinRowsForCalibration is the valid-row count below which
// CalibrationResult's correlation coefficients are too noisy to trust —
// reported as a warning (SmallSample), never a failure: calibrate always
// produces output regardless of corpus size.
const MinRowsForCalibration = 5

// CalibrationResult is calibrate's full output: one MetricCorrelation per
// metric (K_drift, D_const, D_pair) plus corpus-level bookkeeping.
type CalibrationResult struct {
	Rows        int // valid rows actually analyzed
	SmallSample bool
	Metrics     []MetricCorrelation
}
