package calibrate

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/valpere/tumanomir/internal"
	"github.com/valpere/tumanomir/internal/metrics"
	"github.com/valpere/tumanomir/internal/spec"
)

// corpusRow is the on-disk JSONL shape (REQ-CAL-01): one row per
// historical spec, spec_path pinned to an immutable snapshot. Outcome
// is *float64 so a missing/null field unmarshals to nil rather than 0.0
// (a real correctness bug — see LoadCorpus for the full mechanism).
// SpecHash is an optional field that auto-appended rows (Part A corpus
// accretion) use as a measure-side dedup key; hand-built rows from
// before this issue may omit it — LoadCorpus ignores it for backward
// compat.
type corpusRow struct {
	SpecPath   string   `json:"spec_path"`
	Instrument string   `json:"instrument"`
	DPair      float64  `json:"d_pair"`
	Outcome    *float64 `json:"outcome,omitempty"`
	SpecHash   *string  `json:"spec_hash,omitempty"`
	TS         string   `json:"ts,omitempty"`
}

// LoadCorpus reads a JSONL corpus file, one Row per line, and returns the
// parsed rows plus two distinct counts: skipped (rows rejected for
// objective reasons — bad JSON, missing required fields, out-of-range
// values, unreadable spec_path) and unlabeled (rows that parse cleanly
// but have outcome absent/null — these are valid rows, just not yet
// scored; Part A's auto-appended rows look like this).
//
// A line that fails to parse as JSON, has an empty/missing instrument
// (REQ-CAL-02 calls it required — an empty string is not a meaningful
// identifier and must not silently become the corpus baseline), whose
// spec_path can't be opened, or whose d_pair falls outside [0,1] is
// skipped and counted — never aborting the whole run (REQ-CAL-04).
//
// The first valid (non-skipped) row's Instrument becomes the corpus
// baseline; any later valid row naming a different Instrument aborts the
// load immediately with an error naming both values. This is a hard
// abort, not a per-row skip: mixing instruments would produce an
// authoritative-looking but methodologically meaningless correlation,
// since D_pair values measured under different instrument configurations
// aren't comparable (REQ-MSR-04's instrument-relative invariant,
// REQ-CAL-02).
//
// Outcome validation: if Outcome is present (non-nil), it must be in
// [0,1]. Crucially, a nil Outcome is NOT malformed — it's an unlabeled
// row (a distinct bucket from skipped) and is counted in the unlabeled
// return value, never in skipped. The naive `float64` zero-value
// behavior (where `null` unmarshals to 0.0 and passes the range check)
// would corrupt Spearman with fabricated "perfect" outcomes — the
// *float64 design is the only correct fix.
func LoadCorpus(path string) (rows []Row, skipped, unlabeled int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer func() { _ = f.Close() }()

	var baseline string
	haveBaseline := false

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
		if raw.Instrument == "" || raw.DPair < 0 || raw.DPair > 1 {
			skipped++
			continue
		}
		if raw.Outcome != nil && (*raw.Outcome < 0 || *raw.Outcome > 1) {
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
			return nil, 0, 0, fmt.Errorf("corpus mixes instruments %q and %q — all rows in one run must share the same instrument (REQ-MSR-04)", baseline, raw.Instrument)
		}

		// Build Row field-by-field rather than via `Row(raw)` type
		// conversion: the *float64 Outcome field is the only difference
		// from Row, and the staticcheck S1016 (ignore) comment from the
		// previous shared-shape trick no longer applies.
		row := Row{
			SpecPath:   raw.SpecPath,
			Instrument: raw.Instrument,
			DPair:      raw.DPair,
		}
		if raw.Outcome != nil {
			row.Outcome = *raw.Outcome
			rows = append(rows, row)
		} else {
			// Unlabeled: a valid row that just isn't scored yet
			// (Part A auto-appends these). Counted separately from
			// skipped so a user can see "N labeled, M unlabeled" in
			// calibrate's printed output. NOT in rows[] — Analyze
			// only sees labeled rows.
			unlabeled++
		}
	}
	if err := sc.Err(); err != nil {
		return nil, 0, 0, err
	}
	return rows, skipped, unlabeled, nil
}

// RowToWrite is the public input shape for AppendRow: the caller's
// already-recomputed DPair value (and the spec content it was measured
// from) plus the spec_path for human inspection. Path is informational
// only — the dedup key is (SpecHash, Instrument).
type RowToWrite struct {
	SpecContent []byte
	SpecPath    string
	Instrument  string
	DPair       float64
}

// AppendRow writes one row to the corpus file at path, with Outcome left
// nil (an "unlabeled" row — the auto-accretion model is that outcome is
// filled in later by a separate process, not at measure time). Dedup
// rule: if a row with the same (SpecHash, Instrument) already exists in
// the file, the append is skipped — re-running measure on an unchanged
// spec under the same instrument is idempotent with respect to the
// corpus.
//
// Schema-ownership: this writer lives in internal/calibrate (same
// package as LoadCorpus) so the on-disk row schema has exactly one
// source of truth. The caller (cmd/tumanomir/runMeasureImpl) does not
// construct the JSON row itself — it calls this function.
//
// SpecHash is sha256 of the spec content (so the spec_hash pins to the
// exact content that produced this DPair, even if spec_path later
// changes on disk).
//
// Network-free: this function uses only os/os.Create/io, no net/*.
// internal/nonetwork_test.go's guard set is unaffected.
func AppendRow(path string, r RowToWrite) error {
	hash := hashSpec(r.SpecContent)

	if existing, err := os.Open(path); err == nil {
		defer func() { _ = existing.Close() }()
		sc := bufio.NewScanner(existing)
		sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
		for sc.Scan() {
			var existingRow corpusRow
			if err := json.Unmarshal(sc.Bytes(), &existingRow); err != nil {
				continue
			}
			if existingRow.SpecHash != nil && *existingRow.SpecHash == hash &&
				existingRow.Instrument == r.Instrument {
				return nil // already there, idempotent
			}
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s for append: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	row := corpusRow{
		SpecPath:   r.SpecPath,
		Instrument: r.Instrument,
		DPair:      r.DPair,
		Outcome:    nil, // explicitly nil — unlabeled
		SpecHash:   &hash,
		TS:         time.Now().UTC().Format(time.RFC3339Nano),
	}
	enc := json.NewEncoder(f)
	if err := enc.Encode(row); err != nil {
		return fmt.Errorf("encode corpus row: %w", err)
	}
	return nil
}

// hashSpec returns the sha256 hex digest of specContent — AppendRow's
// dedup key component. Unexported: only AppendRow needs it, so
// cmd/tumanomir never has to know the hash algorithm.
func hashSpec(specContent []byte) string {
	hash := sha256.Sum256(specContent)
	return hex.EncodeToString(hash[:])
}

// InstrumentConfigString renders a full internal.InstrumentConfig as a
// stable dedup-key string. Must encode every config field (temperature,
// N, think, num_ctx, num_predict, sim_threshold — not just
// backend:model) so two measurements at different settings never
// silently collide under the dedup key and never let calibrate treat
// methodologically non-comparable DPair values as comparable
// (REQ-MSR-04). Prompt/PromptVersion are deliberately excluded — they
// are not independently configurable (REQ-MSR-04), so including them
// would add noise without adding discriminating power.
func InstrumentConfigString(cfg internal.InstrumentConfig) string {
	var b strings.Builder
	b.WriteString(cfg.Backend)
	b.WriteString(":")
	b.WriteString(cfg.Model)
	b.WriteString("|temp=")
	b.WriteString(strconv.FormatFloat(cfg.Temperature, 'f', -1, 64))
	b.WriteString("|n=")
	b.WriteString(strconv.Itoa(cfg.Samples))
	b.WriteString("|think=")
	b.WriteString(strconv.FormatBool(cfg.Think))
	b.WriteString("|ctx=")
	b.WriteString(strconv.Itoa(cfg.NumCtx))
	b.WriteString("|pred=")
	b.WriteString(strconv.Itoa(cfg.NumPredict))
	b.WriteString("|sim=")
	b.WriteString(strconv.FormatFloat(cfg.SimThreshold, 'f', -1, 64))
	return b.String()
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
