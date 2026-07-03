// Package internal holds types shared across tumanomir packages.
package internal

// Verdict classifies a metric value against its threshold.
type Verdict string

const (
	VerdictOK      Verdict = "ok"
	VerdictWarn    Verdict = "warn"
	VerdictBlock   Verdict = "block"
	VerdictSkipped Verdict = "skipped"
)

// Thresholds are gate boundaries for all metrics. Defaults are hypotheses
// from the methodology article; calibrate them on your own spec corpus.
type Thresholds struct {
	KDriftMax float64 // requirements without trace edges, fraction
	DConstMin float64 // lexical constraint density
	DPairMax  float64 // 1 - mean pairwise AST similarity
}

// DefaultThresholds returns the article's starting-point values.
func DefaultThresholds() Thresholds {
	return Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30}
}

// KDriftResult is the deterministic traceability metric.
type KDriftResult struct {
	Requirements int      // total [REQ-*] found
	Hanging      int      // requirements without any -> [FUN/LOG/PHY-*] edge
	HangingIDs   []string // their identifiers, for actionable output
	Value        float64  // Hanging / Requirements; 0 when no requirements
}

// DConstResult is the lexical constraint-density proxy.
type DConstResult struct {
	ConstraintMarkers int
	ProseTokens       int
	Value             float64 // markers / (markers + prose)
}

// DispersionResult is the stochastic layer output (D_pair, H) for one spec
// under one fixed instrument configuration.
type DispersionResult struct {
	Instrument string  // e.g. "ollama:qwen3-coder:30b"
	N          int     // valid samples measured
	Discarded  int     // invalid generations replaced by retries
	MeanSim    float64 // mean pairwise cosine similarity of AST features
	DPair      float64 // 1 - MeanSim
	Clusters   int     // single-linkage clusters at SimThreshold
	SimThresh  float64
	H          float64 // Shannon entropy over cluster sizes, bits
	HNorm      float64 // H / log2(N), in [0,1]
}
