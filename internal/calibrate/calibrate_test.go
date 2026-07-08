package calibrate

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSpec writes a trivial spec file under dir and returns its path —
// content doesn't matter for these tests since Analyze operates on
// already-recomputed AnalyzedRow values, not on the raw spec bytes; only
// LoadCorpus's "spec_path must be openable" check needs a real file.
func writeSpec(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("[REQ-X-01] x\n-> [FUN-X-01] y\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return path
}

// rowsWithDPair builds n AnalyzedRows whose D_pair/Outcome values are
// dpair[i]/outcome[i] — K_drift/D_const are left at their zero value
// since these tests exercise Analyze's pure correlation math, not the
// metrics themselves.
func rowsWithDPair(dpair, outcome []float64) []AnalyzedRow {
	rows := make([]AnalyzedRow, len(dpair))
	for i := range dpair {
		rows[i] = AnalyzedRow{Row: Row{DPair: dpair[i], Outcome: outcome[i]}}
	}
	return rows
}

// --- Spearman correlation math (🏗️ Production) ---

// TestAnalyzeMonotonicNonlinearIsNearOne constructs a D_pair/Outcome
// relationship that is strictly increasing but nonlinear (Outcome =
// exp(0.8*D_pair)) — Spearman must report ~1.0 regardless of the
// nonlinearity, while a naive Pearson coefficient on the same raw values
// is noticeably lower. This is the test that proves the implementation is
// genuinely rank-based rather than accidentally still linear.
func TestAnalyzeMonotonicNonlinearIsNearOne(t *testing.T) {
	n := 10
	dpair := make([]float64, n)
	outcome := make([]float64, n)
	for i := 0; i < n; i++ {
		dpair[i] = float64(i+1) / float64(n)
		outcome[i] = math.Exp(float64(i+1) * 0.8)
	}

	result := Analyze(rowsWithDPair(dpair, outcome))
	got := metricByName(t, result, "D_pair").Correlation
	if math.Abs(got-1.0) > 0.01 {
		t.Fatalf("Spearman correlation = %v, want within 0.01 of 1.0", got)
	}

	pearsonRaw := pearson(dpair, outcome)
	if pearsonRaw >= 0.9 {
		t.Fatalf("Pearson on raw nonlinear values = %v, want clearly < 1.0 (proves Spearman isn't accidentally linear)", pearsonRaw)
	}
}

// TestAnalyzeUncorrelatedIsNearZero uses a fixed-seed random permutation
// (deterministic, no flakiness) to pair D_pair with an unrelated Outcome
// ordering — the coefficient must land close to 0.
func TestAnalyzeUncorrelatedIsNearZero(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	n := 30
	dpair := make([]float64, n)
	outcome := make([]float64, n)
	for i := 0; i < n; i++ {
		dpair[i] = float64(i + 1)
	}
	for i, p := range r.Perm(n) {
		outcome[i] = float64(p + 1)
	}

	result := Analyze(rowsWithDPair(dpair, outcome))
	got := metricByName(t, result, "D_pair").Correlation
	if math.Abs(got) > 0.5 {
		t.Fatalf("Spearman correlation = %v, want close to 0 for an uncorrelated fixture", got)
	}
}

// TestAnalyzeHeavyTiesBinaryOutcome exercises average-rank tie handling:
// Outcome is only 0 or 1 (five of each, so the outcome array alone is
// half-ties), while D_pair increases monotonically and perfectly
// separates the two outcome groups. Must not panic and must report a
// strong positive (directionally correct) coefficient.
func TestAnalyzeHeavyTiesBinaryOutcome(t *testing.T) {
	n := 10
	dpair := make([]float64, n)
	outcome := make([]float64, n)
	for i := 0; i < n; i++ {
		dpair[i] = float64(i + 1)
		if i >= n/2 {
			outcome[i] = 1
		}
	}

	result := Analyze(rowsWithDPair(dpair, outcome))
	got := metricByName(t, result, "D_pair").Correlation
	if got < 0.7 {
		t.Fatalf("Spearman correlation = %v, want a strong positive coefficient for a perfectly-separating binary outcome", got)
	}
}

// TestAnalyzeSmallSampleWarns confirms SmallSample is set (and the
// coefficients are still computed, not withheld) for a corpus under
// MinRowsForCalibration rows.
func TestAnalyzeSmallSampleWarns(t *testing.T) {
	result := Analyze(rowsWithDPair([]float64{0.1, 0.2, 0.3}, []float64{0.1, 0.5, 0.9}))
	if !result.SmallSample {
		t.Fatalf("want SmallSample=true for %d rows (< %d)", result.Rows, MinRowsForCalibration)
	}
	if len(result.Metrics) != 3 {
		t.Fatalf("want all 3 metrics still reported despite the small-sample warning, got %d", len(result.Metrics))
	}
}

// metricByName finds the named MetricCorrelation in result.Metrics,
// failing the test if absent — a small helper so the correlation tests
// above can assert on "D_pair" specifically without hardcoding index 2.
func metricByName(t *testing.T, result CalibrationResult, name string) MetricCorrelation {
	t.Helper()
	for _, m := range result.Metrics {
		if m.Name == name {
			return m
		}
	}
	t.Fatalf("metric %q not found in result.Metrics = %+v", name, result.Metrics)
	return MetricCorrelation{}
}

// --- LoadCorpus: malformed rows, zero-valid-rows, instrument mismatch (🏗️ Production) ---

func TestLoadCorpusMalformedRowsSkippedAndCounted(t *testing.T) {
	dir := t.TempDir()
	goodSpec := writeSpec(t, dir, "good.md")

	lines := []string{
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:m","d_pair":0.2,"outcome":0.3}`, goodSpec),
		`not valid json at all`,
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:m","d_pair":0.4,"outcome":0.5}`, filepath.Join(dir, "does-not-exist.md")),
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:m","d_pair":1.5,"outcome":0.5}`, goodSpec),
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:m","d_pair":0.5,"outcome":-0.1}`, goodSpec),
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:m","d_pair":0.6,"outcome":0.7}`, goodSpec),
	}
	corpusPath := writeCorpus(t, dir, lines)

	rows, skipped, err := LoadCorpus(corpusPath)
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if skipped != 4 {
		t.Fatalf("skipped = %d, want 4 (bad json, unreadable spec_path, d_pair>1, outcome<0)", skipped)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2 valid rows still processed", len(rows))
	}
}

func TestLoadCorpusZeroValidRows(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		`not valid json`,
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:m","d_pair":2.0,"outcome":0.5}`, filepath.Join(dir, "x.md")),
	}
	corpusPath := writeCorpus(t, dir, lines)

	rows, skipped, err := LoadCorpus(corpusPath)
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("len(rows) = %d, want 0", len(rows))
	}
	if skipped != 2 {
		t.Fatalf("skipped = %d, want 2", skipped)
	}
}

func TestLoadCorpusInstrumentMismatchAborts(t *testing.T) {
	dir := t.TempDir()
	specA := writeSpec(t, dir, "a.md")
	specB := writeSpec(t, dir, "b.md")
	lines := []string{
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:qwen3-coder:30b","d_pair":0.2,"outcome":0.3}`, specA),
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:other-model","d_pair":0.4,"outcome":0.5}`, specB),
	}
	corpusPath := writeCorpus(t, dir, lines)

	_, _, err := LoadCorpus(corpusPath)
	if err == nil {
		t.Fatal("want an error for a corpus mixing two distinct instrument values, got nil")
	}
	if !strings.Contains(err.Error(), "ollama:qwen3-coder:30b") || !strings.Contains(err.Error(), "ollama:other-model") {
		t.Fatalf("error must name both the expected and mismatching instrument, got: %v", err)
	}
}

// TestLoadCorpusEmptyInstrumentSkipped guards REQ-CAL-02's "required"
// contract: an omitted/empty instrument field must not silently become the
// corpus baseline (an empty string is not a meaningful instrument
// identifier) — it's skipped and counted like any other malformed row,
// same as a missing d_pair/outcome would be.
func TestLoadCorpusEmptyInstrumentSkipped(t *testing.T) {
	dir := t.TempDir()
	goodSpec := writeSpec(t, dir, "good.md")
	lines := []string{
		fmt.Sprintf(`{"spec_path":%q,"instrument":"","d_pair":0.2,"outcome":0.3}`, goodSpec),
		fmt.Sprintf(`{"spec_path":%q,"d_pair":0.4,"outcome":0.5}`, goodSpec), // instrument field entirely absent
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:m","d_pair":0.6,"outcome":0.7}`, goodSpec),
	}
	corpusPath := writeCorpus(t, dir, lines)

	rows, skipped, err := LoadCorpus(corpusPath)
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if skipped != 2 {
		t.Fatalf("skipped = %d, want 2 (empty instrument, missing instrument)", skipped)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1 valid row still processed", len(rows))
	}
}

// writeCorpus writes lines (one JSONL row per line, already-serialized)
// to a corpus.jsonl file under dir and returns its path.
func writeCorpus(t *testing.T, dir string, lines []string) string {
	t.Helper()
	path := filepath.Join(dir, "corpus.jsonl")
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write corpus: %v", err)
	}
	return path
}

// --- BuildAnalyzedRows: recomputes K_drift/D_const from the pinned snapshot ---

func TestBuildAnalyzedRowsRecomputesMetrics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	content := "[REQ-X-01] traced\n-> [FUN-X-01] Do()\n[REQ-X-02] not traced\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	rows := []Row{{SpecPath: path, Instrument: "ollama:m", DPair: 0.3, Outcome: 0.6}}
	analyzed, err := BuildAnalyzedRows(rows)
	if err != nil {
		t.Fatalf("BuildAnalyzedRows: %v", err)
	}
	if len(analyzed) != 1 {
		t.Fatalf("len(analyzed) = %d, want 1", len(analyzed))
	}
	if analyzed[0].KDrift.Requirements != 2 || analyzed[0].KDrift.Hanging != 1 {
		t.Fatalf("KDrift = %+v, want 2 requirements/1 hanging (recomputed fresh from the snapshot)", analyzed[0].KDrift)
	}
	if analyzed[0].DPair != 0.3 {
		t.Fatalf("DPair = %v, want 0.3 (never recomputed, carried through from the corpus row)", analyzed[0].DPair)
	}
}

func TestBuildAnalyzedRowsUnreadableSpecPathErrors(t *testing.T) {
	rows := []Row{{SpecPath: filepath.Join(t.TempDir(), "missing.md"), Instrument: "ollama:m", DPair: 0.1, Outcome: 0.1}}
	if _, err := BuildAnalyzedRows(rows); err == nil {
		t.Fatal("want an error when a row's spec_path can no longer be read, got nil")
	}
}
