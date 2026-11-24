package dispersion

import (
	"math"
	"testing"
)

// ciEpsilon tolerates IEEE 754 rounding noise around the mathematical [0,1]
// bound: cosine of two identical-content vectors is mathematically 1.0 but
// sqrt(na)*sqrt(nb) doesn't always round-trip bit for bit (e.g. na=23 ->
// sqrt(23)^2 = 22.999999999999996), so a stored similarity can sit a few
// ULPs either side of 1 and push D* just outside [0,1]. That is float
// noise, not a real out-of-range value.
const ciEpsilon = 1e-9

// TestBootstrapDPairCIBounds guards the basic contract every caller relies
// on: both bounds land in [0,1] and low never exceeds high, across a mix of
// N and feature-vector shapes (not the point estimate DPair itself — a
// percentile bootstrap CI is not guaranteed to bracket the point estimate,
// so that's deliberately not asserted here or anywhere else in this file).
func TestBootstrapDPairCIBounds(t *testing.T) {
	feats := []map[string]float64{
		{"type:a": 1, "field:a:x:int": 1},
		{"type:a": 1, "field:a:x:int": 1, "field:a:y:string": 1},
		{"type:b": 1, "func:helper:func()": 1},
		{"type:a": 1, "field:a:x:int": 2},
	}
	low, high := ciOf(feats)
	if low > high {
		t.Fatalf("CILow (%v) > CIHigh (%v), want low <= high", low, high)
	}
	if low < -ciEpsilon || low > 1+ciEpsilon {
		t.Fatalf("CILow = %v, want in [0,1] (tol %v)", low, ciEpsilon)
	}
	if high < -ciEpsilon || high > 1+ciEpsilon {
		t.Fatalf("CIHigh = %v, want in [0,1] (tol %v)", high, ciEpsilon)
	}
}

// TestBootstrapDPairCIAllIdentical guards the collapse case: when every
// feature vector is the exact same map object, every stored pairwise
// similarity is cosine(v,v) = 1.0, so D* is 0 for every resample and the CI
// collapses to exactly [0,0] — not approximately, since a single-key vector
// makes cosine(v,v) exact (no summation-order or sqrt-rounding wiggle room).
func TestBootstrapDPairCIAllIdentical(t *testing.T) {
	v := map[string]float64{"type:a": 5}
	feats := make([]map[string]float64, 6)
	for i := range feats {
		feats[i] = v
	}
	low, high := ciOf(feats)
	if low != 0 || high != 0 {
		t.Fatalf("CI = [%v, %v], want exactly [0, 0] for all-identical input", low, high)
	}
}

// TestBootstrapDPairCIDeterministic guards REQ-MSR-04's reproducibility
// invariant: the same seed against the same feature vectors must yield
// bit-identical bounds across calls — Analyze relies on this to stay a
// deterministic pure function.
func TestBootstrapDPairCIDeterministic(t *testing.T) {
	feats := []map[string]float64{
		{"type:a": 1, "field:a:x:int": 1},
		{"type:a": 1, "field:a:x:int": 1, "field:a:y:string": 1},
		{"type:b": 1, "func:helper:func()": 1},
	}
	low1, high1 := ciOf(feats)
	low2, high2 := ciOf(feats)
	if low1 != low2 || high1 != high2 {
		t.Fatalf("bootstrapDPairCI is not deterministic: run1=[%v,%v] run2=[%v,%v]", low1, high1, low2, high2)
	}
}

// TestBootstrapDPairCINEqualsTwo guards the smallest valid input Analyze
// ever passes through (its n>=2 guard). About half the resamples draw the
// same index twice and are redrawn; every kept one is the original pair, so
// the CI is exactly [D, D]. Must not panic or produce NaN.
func TestBootstrapDPairCINEqualsTwo(t *testing.T) {
	feats := []map[string]float64{
		{"type:a": 1, "field:a:x:int": 1},
		{"type:b": 1, "func:helper:func()": 1},
	}
	low, high := ciOf(feats)
	if math.IsNaN(low) || math.IsNaN(high) {
		t.Fatalf("CI = [%v, %v], want no NaN for N=2", low, high)
	}
	if low > high {
		t.Fatalf("CILow (%v) > CIHigh (%v) for N=2", low, high)
	}
	if low < -ciEpsilon || high > 1+ciEpsilon {
		t.Fatalf("CI = [%v, %v], want in [0,1] (tol %v) for N=2", low, high, ciEpsilon)
	}
	if _, d := simsOf(feats); low != d || high != d {
		t.Fatalf("CI = [%v, %v], want [D, D] = [%v, %v] for N=2", low, high, d, d)
	}
}

// TestAnalyzePopulatesDPairCI guards the wiring between Analyze and
// bootstrapDPairCI: DispersionResult's CI fields must actually be set from
// real feature vectors when N>=2, and must stay at their zero value when
// N<2 (the same threshold that already guards MeanSim/DPair/H/HNorm).
func TestAnalyzePopulatesDPairCI(t *testing.T) {
	a := []byte(`package sample

type Foo struct {
	X int
}
`)
	b := []byte(`package sample

type Bar struct {
	Y string
	Z int
}

func Helper() {}
`)
	res := Analyze([][]byte{a, b}, 0.95)
	if res.N != 2 {
		t.Fatalf("N = %d, want 2; result=%+v", res.N, res)
	}
	if res.DPairCILow > res.DPairCIHigh {
		t.Fatalf("DPairCILow (%v) > DPairCIHigh (%v); result=%+v", res.DPairCILow, res.DPairCIHigh, res)
	}
	if res.DPairCILow < -ciEpsilon || res.DPairCIHigh > 1+ciEpsilon {
		t.Fatalf("CI out of [0,1] (tol %v): result=%+v", ciEpsilon, res)
	}

	res1 := Analyze([][]byte{a}, 0.95)
	if res1.DPairCILow != 0 || res1.DPairCIHigh != 0 {
		t.Fatalf("want CI to stay at zero value for N<2, got [%v, %v]; result=%+v", res1.DPairCILow, res1.DPairCIHigh, res1)
	}
}

// simsOf builds the pairwise similarity matrix Analyze hands the bootstrap
// (diagonal left zero) and returns it with the point-estimate D_pair.
func simsOf(feats []map[string]float64) (sims [][]float64, dpair float64) {
	sims, meanSim := pairwiseSims(feats)
	return sims, 1 - meanSim
}

// ciOf runs the production CI over feature vectors with the production
// constants.
func ciOf(feats []map[string]float64) (low, high float64) {
	sims, dpair := simsOf(feats)
	return bootstrapDPairCI(sims, dpair, bootstrapCIB, bootstrapCISeed)
}

// bootstrapDStars returns the bootstrap D* distribution for the sources.
func bootstrapDStars(t *testing.T, sources [][]byte) []float64 {
	t.Helper()
	feats := make([]map[string]float64, 0, len(sources))
	for _, src := range sources {
		f, ok := features(src)
		if !ok {
			t.Fatal("unparseable source")
		}
		feats = append(feats, f)
	}
	sims, _ := simsOf(feats)
	return bootstrapDPairs(sims, bootstrapCIB, bootstrapCISeed)
}

// TestBootstrapCenteredOnDPair is the REQ-MSR-07 regression: self-pairs
// (a resample drawing one original sample twice) must not be scored as
// similarity 1, or the bootstrap centers on (1-1/N)*D_pair and the CI is
// shifted low by ~1/N. On the N=9 fixture the old gap was ~0.028.
func TestBootstrapCenteredOnDPair(t *testing.T) {
	valid, _ := loadDeclsFixture(t)
	dstars := bootstrapDStars(t, valid)
	var sum float64
	for _, d := range dstars {
		sum += d
	}
	mean := sum / float64(len(dstars))
	dpair := Analyze(valid, 0.95).DPair
	if gap := math.Abs(mean - dpair); gap >= 0.01 {
		t.Fatalf("|mean(D*) - D_pair| = %.4f (mean %.4f, D_pair %.4f), want < 0.01", gap, mean, dpair)
	}
}

// TestBootstrapDPairCITerminationCap guards the redraw cap and fallback:
// with N=1 every resample is degenerate (no distinct-index pair), so the
// loop must stop after 100*b attempts, collect nothing, and the CI must
// fall back to (D, D) instead of hanging or returning NaN.
func TestBootstrapDPairCITerminationCap(t *testing.T) {
	sims := [][]float64{{0}}
	if got := bootstrapDPairs(sims, 50, bootstrapCISeed); len(got) != 0 {
		t.Fatalf("bootstrapDPairs on N=1 = %v, want none collected", got)
	}
	low, high := bootstrapDPairCI(sims, 0.37, 50, bootstrapCISeed)
	if low != 0.37 || high != 0.37 {
		t.Fatalf("CI = [%v, %v], want fallback [0.37, 0.37]", low, high)
	}
}
