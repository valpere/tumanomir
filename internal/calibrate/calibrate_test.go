package calibrate

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
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

	rows, skipped, _, err := LoadCorpus(corpusPath)
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

	rows, skipped, _, err := LoadCorpus(corpusPath)
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

	_, _, _, err := LoadCorpus(corpusPath)
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

	rows, skipped, _, err := LoadCorpus(corpusPath)
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

// --- Part A: corpus accretion (issue #107) ---

// TestAppendRowCreatesNewFile: when the corpus file does not exist yet,
// AppendRow creates it and writes one row with the expected shape
// (spec_hash computed from specContent, instrument encoded as the full
// InstrumentConfig string, d_pair copied through, outcome nil/omitted,
// timestamp set).
func TestAppendRowCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corpus.jsonl")
	specContent := []byte("package x\n")
	specPath := writeSpec(t, dir, "spec.md")
	inst := "ollama:qwen3-coder:30b|temp=1.000000|n=10|think=false|ctx=8192|pred=2048|sim=0.950000"

	if err := AppendRow(path, RowToWrite{
		SpecContent: specContent,
		SpecPath:    specPath,
		Instrument:  inst,
		DPair:       0.27,
	}); err != nil {
		t.Fatalf("AppendRow: %v", err)
	}

	// AppendRow writes one row with nil Outcome (unlabeled). LoadCorpus
	// therefore reports it via the unlabeled counter, not the labeled
	// rows slice.
	if got := loadCorpusForTest(t, path); got != 1 {
		t.Fatalf("file has %d JSONL lines, want 1", got)
	}
	rows, skipped, unlabeled, err := LoadCorpus(path)
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("labeled rows = %d, want 0 (AppendRow writes unlabeled, not labeled)", len(rows))
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
	if unlabeled != 1 {
		t.Errorf("unlabeled = %d, want 1", unlabeled)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus file: %v", err)
	}
	if !bytes.Contains(data, []byte(`"d_pair":0.27`)) {
		t.Errorf("file does not contain d_pair:0.27: %s", data)
	}
	if !bytes.Contains(data, []byte(`"spec_path":`)) {
		t.Errorf("file does not contain spec_path: %s", data)
	}
}

// TestAppendRowDedup: calling AppendRow twice with the same specHash and
// instrument (regardless of whether the d_pair differs between calls)
// must produce exactly one row on disk — the (spec_hash, instrument)
// dedup rule. Only the FIRST call's d_pair should survive.
func TestAppendRowDedup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corpus.jsonl")
	specContent := []byte("package y\n")
	specPath := writeSpec(t, dir, "spec.md")
	inst := "ollama:qwen3-coder:30b|temp=1.000000|n=10|think=false|ctx=8192|pred=2048|sim=0.950000"

	for _, d := range []float64{0.10, 0.20, 0.30} {
		if err := AppendRow(path, RowToWrite{
			SpecContent: specContent,
			SpecPath:    specPath,
			Instrument:  inst,
			DPair:       d,
		}); err != nil {
			t.Fatalf("AppendRow(d=%f): %v", d, err)
		}
	}
	if got := loadCorpusForTest(t, path); got != 1 {
		t.Fatalf("dedup failed: file has %d JSONL lines, want 1 (same spec+instrument, only the first should be appended)", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus file: %v", err)
	}
	if !bytes.Contains(data, []byte(`"d_pair":0.1`)) {
		t.Errorf("file should retain the FIRST call's d_pair=0.1, got: %s", data)
	}
}

// TestAppendRowDedupBySpecHashOnly: the spec_hash is a *content* hash, so
// two different spec contents (even at the same instrument) are NOT
// dedup-collapsed — they are genuinely different rows.
func TestAppendRowDedupBySpecHashOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corpus.jsonl")
	specPathA := writeSpec(t, dir, "a.md")
	specPathB := writeSpec(t, dir, "b.md")
	inst := "ollama:qwen3-coder:30b|temp=1.000000|n=10|think=false|ctx=8192|pred=2048|sim=0.950000"

	if err := AppendRow(path, RowToWrite{SpecContent: []byte("package a\n"), SpecPath: specPathA, Instrument: inst, DPair: 0.1}); err != nil {
		t.Fatalf("AppendRow a: %v", err)
	}
	if err := AppendRow(path, RowToWrite{SpecContent: []byte("package b\n"), SpecPath: specPathB, Instrument: inst, DPair: 0.2}); err != nil {
		t.Fatalf("AppendRow b: %v", err)
	}
	if got := loadCorpusForTest(t, path); got != 2 {
		t.Fatalf("file has %d JSONL lines, want 2 (different spec contents must be distinct rows)", got)
	}
}

// TestLoadCorpusUnlabeledRowsCountedSeparately: a corpus mixing
// labeled, unlabeled (no outcome), and genuinely malformed rows must
// report three distinct counts and never let the unlabeled row's
// `outcome: null` unmarshal to 0.0 and corrupt the analyzed rows.
func TestLoadCorpusUnlabeledRowsCountedSeparately(t *testing.T) {
	dir := t.TempDir()
	goodSpec := writeSpec(t, dir, "good.md")
	inst := "ollama:qwen3-coder:30b|temp=1.0|n=10|think=false|ctx=8192|pred=2048|sim=0.95"

	lines := []string{
		// labeled, valid
		fmt.Sprintf(`{"spec_path":%q,"instrument":%q,"d_pair":0.2,"outcome":0.3}`, goodSpec, inst),
		// unlabeled: outcome field absent entirely
		fmt.Sprintf(`{"spec_path":%q,"instrument":%q,"d_pair":0.4}`, goodSpec, inst),
		// unlabeled: outcome field explicitly null
		fmt.Sprintf(`{"spec_path":%q,"instrument":%q,"d_pair":0.5,"outcome":null}`, goodSpec, inst),
		// malformed: d_pair out of range
		fmt.Sprintf(`{"spec_path":%q,"instrument":%q,"d_pair":1.5,"outcome":0.5}`, goodSpec, inst),
		// malformed: bad JSON
		`not valid json`,
		// labeled, valid — different DPair to confirm ranking signal
		fmt.Sprintf(`{"spec_path":%q,"instrument":%q,"d_pair":0.6,"outcome":0.7}`, goodSpec, inst),
	}
	corpusPath := writeCorpus(t, dir, lines)

	rows, skipped, unlabeled, err := LoadCorpus(corpusPath)
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if skipped != 2 {
		t.Errorf("skipped = %d, want 2 (d_pair>1, bad json)", skipped)
	}
	if unlabeled != 2 {
		t.Errorf("unlabeled = %d, want 2 (absent outcome, null outcome)", unlabeled)
	}
	if len(rows) != 2 {
		t.Errorf("len(rows) = %d, want 2 labeled rows", len(rows))
	}
	// The CRITICAL invariant: no unlabeled row is in rows[] — Analyze
	// must never see a fabricated 0.0 outcome from a null field.
	for i, r := range rows {
		// D_pair is preserved as the labeled row's value (not 0.0).
		if r.Outcome != 0.3 && r.Outcome != 0.7 {
			t.Errorf("rows[%d].Outcome = %f, expected 0.3 or 0.7 (labeled)", i, r.Outcome)
		}
	}
}

// TestLoadCorpusToleratesMissingSpecHashField: a hand-built old-format
// row (no spec_hash field) must still load — backward compat with the
// pre-#107 schema.
func TestLoadCorpusToleratesMissingSpecHashField(t *testing.T) {
	dir := t.TempDir()
	goodSpec := writeSpec(t, dir, "spec.md")
	// Note: no spec_hash field at all.
	lines := []string{
		fmt.Sprintf(`{"spec_path":%q,"instrument":"ollama:m","d_pair":0.2,"outcome":0.3}`, goodSpec),
	}
	corpusPath := writeCorpus(t, dir, lines)

	rows, _, _, err := LoadCorpus(corpusPath)
	if err != nil {
		t.Fatalf("LoadCorpus: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("len(rows) = %d, want 1 (old-format row should still load)", len(rows))
	}
}

// TestInstrumentConfigStringDistinguishes: two configs that differ in any
// single field must produce different strings — dedup correctness.
func TestInstrumentConfigStringDistinguishes(t *testing.T) {
	base := func(mutate func(*internal.InstrumentConfig)) string {
		cfg := internal.InstrumentConfig{
			Backend: "ollama", Model: "m", Temperature: 1.0, Samples: 10,
			Think: false, NumCtx: 8192, NumPredict: 2048, SimThreshold: 0.95,
		}
		if mutate != nil {
			mutate(&cfg)
		}
		return InstrumentConfigString(cfg)
	}
	a := base(nil)
	b := base(nil)
	if a != b {
		t.Errorf("identical configs produced different strings: %q vs %q", a, b)
	}
	// Each field that affects methodology should distinguish.
	checks := []struct {
		name string
		x, y string
	}{
		{"temperature", base(nil), base(func(c *internal.InstrumentConfig) { c.Temperature = 0.5 })},
		{"samples", base(nil), base(func(c *internal.InstrumentConfig) { c.Samples = 20 })},
		{"think", base(nil), base(func(c *internal.InstrumentConfig) { c.Think = true })},
		{"numCtx", base(nil), base(func(c *internal.InstrumentConfig) { c.NumCtx = 4096 })},
		{"numPredict", base(nil), base(func(c *internal.InstrumentConfig) { c.NumPredict = 1024 })},
		{"simThreshold", base(nil), base(func(c *internal.InstrumentConfig) { c.SimThreshold = 0.80 })},
	}
	for _, c := range checks {
		if c.x == c.y {
			t.Errorf("%s: two configs differing only in this field produced the same string %q", c.name, c.x)
		}
	}
}

// TestHashSpecStable: same content → same hash (caller relies on this for
// dedup correctness).
func TestHashSpecStable(t *testing.T) {
	a := HashSpec([]byte("package x\ntype T struct{}"))
	b := HashSpec([]byte("package x\ntype T struct{}"))
	if a != b {
		t.Errorf("same content produced different hashes: %q vs %q", a, b)
	}
	if a == HashSpec([]byte("package y\n")) {
		t.Errorf("different content produced same hash %q", a)
	}
}

// loadCorpusForTestForAppend is the helper for the AppendRow tests:
// the appended rows have nil Outcome (they're unlabeled), so LoadCorpus
// returns them via the unlabeled counter, NOT via the labeled `rows`
// slice. We surface the raw JSONL line count from the file itself, which
// is what the AppendRow tests actually care about (the row was written).
func loadCorpusForTest(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus file: %v", err)
	}
	return bytes.Count(data, []byte("\n"))
}
