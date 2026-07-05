package dispersion

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// Tolerances for the reference numbers in testdata/README.md: MeanSim/DPair
// are deterministic pure-math computations over fixed fixture inputs, so a
// tight tolerance is appropriate — a larger discrepancy is a real finding,
// not noise to paper over. H is reported to 2 decimals in the README, hence
// a slightly looser tolerance.
const (
	simTol = 0.005
	hTol   = 0.02
)

// loadGoFixtures reads all <n>.go fixtures from dir (sorted by filename for
// determinism) into [][]byte. It fails the test loudly, naming the offending
// file, if any fixture does not parse as valid Go — "all N fixtures parse"
// is itself part of what testdata/README.md's reference numbers assert.
func loadGoFixtures(t *testing.T, dir string) [][]byte {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	if len(matches) == 0 {
		t.Fatalf("no .go fixtures found in %s", dir)
	}
	sort.Strings(matches)

	sources := make([][]byte, 0, len(matches))
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		if !ValidGo(b) {
			t.Fatalf("fixture %s does not parse as valid Go (0 extractable AST features)", m)
		}
		sources = append(sources, b)
	}
	return sources
}

// TestAnalyzeInstrumentASharp reproduces the Instrument A ("sharp" spec,
// qwen3-coder:30b local) reference numbers from testdata/README.md:
// mean pairwise sim 0.730, D_pair 0.270, H@0.95 1.77 bits.
func TestAnalyzeInstrumentASharp(t *testing.T) {
	sources := loadGoFixtures(t, "testdata/out/sharp")
	if len(sources) != 10 {
		t.Fatalf("want 10 fixtures, got %d", len(sources))
	}

	res := Analyze(sources, 0.95)
	if res.N != 10 {
		t.Fatalf("want N=10 (all fixtures valid), got %+v", res)
	}
	if diff := math.Abs(res.MeanSim - 0.730); diff > simTol {
		t.Fatalf("MeanSim: want ~0.730 (tol %.3f), got %.4f; result=%+v", simTol, res.MeanSim, res)
	}
	if diff := math.Abs(res.DPair - 0.270); diff > simTol {
		t.Fatalf("DPair: want ~0.270 (tol %.3f), got %.4f; result=%+v", simTol, res.DPair, res)
	}
	if diff := math.Abs(res.H - 1.77); diff > hTol {
		t.Fatalf("H: want ~1.77 bits (tol %.2f), got %.4f; result=%+v", hTol, res.H, res)
	}
}

// TestAnalyzeInstrumentBSharp reproduces the Instrument B ("sharp" spec,
// kimi-k2.7-code:cloud, think=false) reference numbers from
// testdata/README.md: mean pairwise sim 0.682, D_pair 0.318, H@0.95 3.32
// bits, H@0.80 2.12 bits. The two thresholds exercise that simThreshold
// actually changes clustering (H), while MeanSim/DPair stay
// threshold-independent.
func TestAnalyzeInstrumentBSharp(t *testing.T) {
	sources := loadGoFixtures(t, "testdata/out-cloud/sharp")
	if len(sources) != 10 {
		t.Fatalf("want 10 fixtures, got %d", len(sources))
	}

	res95 := Analyze(sources, 0.95)
	if res95.N != 10 {
		t.Fatalf("want N=10 (all fixtures valid), got %+v", res95)
	}
	if diff := math.Abs(res95.MeanSim - 0.682); diff > simTol {
		t.Fatalf("MeanSim: want ~0.682 (tol %.3f), got %.4f; result=%+v", simTol, res95.MeanSim, res95)
	}
	if diff := math.Abs(res95.DPair - 0.318); diff > simTol {
		t.Fatalf("DPair: want ~0.318 (tol %.3f), got %.4f; result=%+v", simTol, res95.DPair, res95)
	}
	if diff := math.Abs(res95.H - 3.32); diff > hTol {
		t.Fatalf("H@0.95: want ~3.32 bits (tol %.2f), got %.4f; result=%+v", hTol, res95.H, res95)
	}

	res80 := Analyze(sources, 0.80)
	if res80.MeanSim != res95.MeanSim || res80.DPair != res95.DPair {
		t.Fatalf("MeanSim/DPair must be threshold-independent: @0.95=%+v @0.80=%+v", res95, res80)
	}
	if diff := math.Abs(res80.H - 2.12); diff > hTol {
		t.Fatalf("H@0.80: want ~2.12 bits (tol %.2f), got %.4f; result=%+v", hTol, res80.H, res80)
	}
}

// TestFeatures is a small, hand-countable sanity check of the AST feature
// extraction: a struct with two fields, an interface with one method, a
// const, a method and a plain function should each contribute exactly the
// feature keys features() is documented to produce.
func TestFeatures(t *testing.T) {
	src := []byte(`package sample

type Point struct {
	X int
	Y int
}

type Named interface {
	Get() string
}

const MaxCount = 10

func (p Point) Sum() int {
	return p.X + p.Y
}

func Helper() {}
`)
	got, ok := features(src)
	if !ok {
		t.Fatalf("expected valid Go source to yield features")
	}

	want := map[string]float64{
		"type:point":                     1,
		"kind:point:struct":              1,
		"field:point:x:int":              1,
		"field:point:y:int":              1,
		"type:named":                     1,
		"kind:named:interface":           1,
		"method:named:get:func() string": 1,
		"const:maxcount":                 1,
		"func:point.sum:func() int":      1,
		"func:helper:func()":             1,
	}
	if len(got) != len(want) {
		t.Fatalf("feature count: want %d, got %d; got=%+v", len(want), len(got), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("feature %q: want %v, got %v; got=%+v", k, v, got[k], got)
		}
	}
}
