package calibrate

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/valpere/tumanomir/internal/metrics"
	"github.com/valpere/tumanomir/internal/spec"
)

// corpusRow is the on-disk JSONL shape (REQ-CAL-01): one row per
// historical spec, spec_path pinned to an immutable snapshot.
type corpusRow struct {
	SpecPath   string  `json:"spec_path"`
	Instrument string  `json:"instrument"`
	DPair      float64 `json:"d_pair"`
	Outcome    float64 `json:"outcome"`
}

// LoadCorpus reads a JSONL corpus file, one Row per line. A line that
// fails to parse as JSON, whose spec_path can't be opened, or whose
// d_pair/outcome falls outside [0,1] is skipped and counted in skipped —
// never aborting the whole run (REQ-CAL-04).
//
// The first valid row's Instrument becomes the corpus baseline; any later
// valid row naming a different Instrument aborts the load immediately
// with an error naming both values. This is a hard abort, not a per-row
// skip: mixing instruments would produce an authoritative-looking but
// methodologically meaningless correlation, since D_pair values measured
// under different instrument configurations aren't comparable
// (REQ-MSR-04's instrument-relative invariant, REQ-CAL-02).
func LoadCorpus(path string) (rows []Row, skipped int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = f.Close() }()

	var baseline string
	haveBaseline := false

	// The default 64KB token limit is plenty for one JSONL row (a
	// spec_path plus three scalars), but bufio.NewScanner's default max
	// is also 64KB — raise it defensively rather than risk a bufio.
	// ErrTooLong on some future corpus with unusually long paths.
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}

		var raw corpusRow
		if err := json.Unmarshal(line, &raw); err != nil {
			skipped++
			continue
		}
		if raw.DPair < 0 || raw.DPair > 1 || raw.Outcome < 0 || raw.Outcome > 1 {
			skipped++
			continue
		}
		specFile, err := os.Open(raw.SpecPath)
		if err != nil {
			skipped++
			continue
		}
		_ = specFile.Close()

		if !haveBaseline {
			baseline = raw.Instrument
			haveBaseline = true
		} else if raw.Instrument != baseline {
			return nil, 0, fmt.Errorf("corpus mixes instruments %q and %q — all rows in one run must share the same instrument (REQ-MSR-04)", baseline, raw.Instrument)
		}

		// raw and Row share identical fields (differing only in JSON
		// tags), so a type conversion is the idiomatic move here rather
		// than a field-by-field struct literal (staticcheck S1016).
		rows = append(rows, Row(raw))
	}
	if err := sc.Err(); err != nil {
		return nil, 0, err
	}
	return rows, skipped, nil
}

// BuildAnalyzedRows recomputes K_drift/D_const fresh from each row's
// pinned spec snapshot (DRY: the snapshot is the single source of truth
// for its own deterministic metrics), never recomputing DPair — that
// would mean re-running an LLM instrument.
func BuildAnalyzedRows(rows []Row) ([]AnalyzedRow, error) {
	analyzed := make([]AnalyzedRow, 0, len(rows))
	for _, r := range rows {
		specs, err := spec.Load(r.SpecPath)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", r.SpecPath, err)
		}
		var content []byte
		for _, s := range specs {
			content = append(content, s.Content...)
		}
		analyzed = append(analyzed, AnalyzedRow{
			Row:    r,
			KDrift: metrics.KDrift(content),
			DConst: metrics.DConst(content),
		})
	}
	return analyzed, nil
}

// Analyze computes each metric's (K_drift, D_const, D_pair) Spearman rank
// correlation against Outcome, plus a median-outcome-split min/mean/max
// summary — pure arithmetic over already-recomputed AnalyzedRows, no I/O.
func Analyze(rows []AnalyzedRow) CalibrationResult {
	outcomes := make([]float64, len(rows))
	kdrift := make([]float64, len(rows))
	dconst := make([]float64, len(rows))
	dpair := make([]float64, len(rows))
	for i, r := range rows {
		outcomes[i] = r.Outcome
		kdrift[i] = r.KDrift.Value
		dconst[i] = r.DConst.Value
		dpair[i] = r.DPair
	}

	named := []struct {
		name string
		vals []float64
	}{
		{"K_drift", kdrift},
		{"D_const", dconst},
		{"D_pair", dpair},
	}

	result := CalibrationResult{
		Rows:        len(rows),
		SmallSample: len(rows) < MinRowsForCalibration,
	}
	for _, m := range named {
		low, high := medianSplit(m.vals, outcomes)
		result.Metrics = append(result.Metrics, MetricCorrelation{
			Name:        m.name,
			Correlation: spearman(m.vals, outcomes),
			LowHalf:     low,
			HighHalf:    high,
		})
	}
	return result
}

// rank assigns 1-based average ranks to vs — Spearman's tie-handling
// rule: equal values share the mean of the rank positions they
// collectively occupy (e.g. two ties at positions 4 and 5 both get rank
// 4.5).
func rank(vs []float64) []float64 {
	n := len(vs)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return vs[idx[a]] < vs[idx[b]] })

	ranks := make([]float64, n)
	for i := 0; i < n; {
		j := i
		for j+1 < n && vs[idx[j+1]] == vs[idx[i]] {
			j++
		}
		avg := float64(i+j+2) / 2 // mean of 1-based positions i+1..j+1
		for k := i; k <= j; k++ {
			ranks[idx[k]] = avg
		}
		i = j + 1
	}
	return ranks
}

// pearson computes the Pearson product-moment correlation coefficient of
// xs and ys. Returns 0 when either series has zero variance — correlation
// is undefined there, not "perfectly correlated."
func pearson(xs, ys []float64) float64 {
	n := float64(len(xs))
	var sx, sy, sxy, sxx, syy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
		sxy += xs[i] * ys[i]
		sxx += xs[i] * xs[i]
		syy += ys[i] * ys[i]
	}
	den := math.Sqrt((n*sxx - sx*sx) * (n*syy - sy*sy))
	if den == 0 {
		return 0
	}
	return (n*sxy - sx*sy) / den
}

// spearman computes Spearman's rank correlation coefficient: the Pearson
// correlation of xs/ys after rank-transforming each with average-rank tie
// handling — the standard, simplest correct construction (REQ-CAL-03).
// Deliberately not Pearson on the raw values: Outcome is an arbitrary
// caller-defined scale, so only a monotonic relationship is meaningful to
// test, and Spearman degrades correctly to the binary "clear vs. fog"
// case via its tie handling.
func spearman(xs, ys []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	return pearson(rank(xs), rank(ys))
}

// median returns the median of vs (the average of the two middle
// elements for an even-length slice), without mutating vs.
func median(vs []float64) float64 {
	if len(vs) == 0 {
		return 0
	}
	sorted := append([]float64(nil), vs...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// medianSplit partitions metricVals by whether their paired outcome is
// at/below or above the median outcome, and returns each half's
// min/mean/max (REQ-CAL-03) — informational only, never used to
// auto-select a threshold (REQ-NFR-03).
func medianSplit(metricVals, outcomes []float64) (low, high Range) {
	med := median(outcomes)
	var lowVals, highVals []float64
	for i, o := range outcomes {
		if o <= med {
			lowVals = append(lowVals, metricVals[i])
		} else {
			highVals = append(highVals, metricVals[i])
		}
	}
	return rangeOf(lowVals), rangeOf(highVals)
}

// rangeOf returns vs's min/mean/max, or the zero Range if vs is empty (a
// degenerate median split where every row lands in one half).
func rangeOf(vs []float64) Range {
	if len(vs) == 0 {
		return Range{}
	}
	r := Range{Min: vs[0], Max: vs[0]}
	var sum float64
	for _, v := range vs {
		if v < r.Min {
			r.Min = v
		}
		if v > r.Max {
			r.Max = v
		}
		sum += v
	}
	r.Mean = sum / float64(len(vs))
	return r
}
