package dispersion

import (
	"math"
	"testing"
)

// ciEpsilon tolerates IEEE 754 rounding noise around the mathematical [0,1]
// bound: cosine(v,v) is mathematically exactly 1.0 for any non-empty
// vector, but sqrt(na)*sqrt(na) doesn't always round-trip back to na bit
// for bit (e.g. na=23 -> sqrt(23)^2 = 22.999999999999996), so a bootstrap
// resample landing on a same-index self-pair can push DPair a few ULPs on
// either side of 0 or 1. That's float noise, not a real out-of-range
// value — %.2f-rounded report output can't even distinguish it from 0.00
// or 1.00.
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
	low, high := bootstrapDPairCI(feats, bootstrapCIB, bootstrapCISeed)
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
// feature vector is the exact same map object, every resample's mean
// pairwise similarity is 1.0 (cosine(v,v), including self-pairs drawn twice
// by the same resample) so DPair is 0 for all B resamples and the CI
// collapses to exactly [0,0] — not approximately, since a single-key vector
// makes cosine(v,v) exact (na/nb/dot all reduce to the same one term, no
// summation-order or sqrt-rounding wiggle room).
func TestBootstrapDPairCIAllIdentical(t *testing.T) {
	v := map[string]float64{"type:a": 5}
	feats := make([]map[string]float64, 6)
	for i := range feats {
		feats[i] = v
	}
	low, high := bootstrapDPairCI(feats, bootstrapCIB, bootstrapCISeed)
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
	low1, high1 := bootstrapDPairCI(feats, bootstrapCIB, bootstrapCISeed)
	low2, high2 := bootstrapDPairCI(feats, bootstrapCIB, bootstrapCISeed)
	if low1 != low2 || high1 != high2 {
		t.Fatalf("bootstrapDPairCI is not deterministic: run1=[%v,%v] run2=[%v,%v]", low1, high1, low2, high2)
	}
}

// TestBootstrapDPairCINEqualsTwo guards the smallest valid input Analyze
// ever passes through (its n>=2 guard) — only 3 distinct resample
// multisets are possible when drawing 2 indices with replacement from
// {0,1} ({0,0}, {0,1}/{1,0}, {1,1}), so this exercises the self-pair path
// on nearly every resample. Must not panic or produce NaN.
func TestBootstrapDPairCINEqualsTwo(t *testing.T) {
	feats := []map[string]float64{
		{"type:a": 1, "field:a:x:int": 1},
		{"type:b": 1, "func:helper:func()": 1},
	}
	low, high := bootstrapDPairCI(feats, bootstrapCIB, bootstrapCISeed)
	if math.IsNaN(low) || math.IsNaN(high) {
		t.Fatalf("CI = [%v, %v], want no NaN for N=2", low, high)
	}
	if low > high {
		t.Fatalf("CILow (%v) > CIHigh (%v) for N=2", low, high)
	}
	if low < -ciEpsilon || high > 1+ciEpsilon {
		t.Fatalf("CI = [%v, %v], want in [0,1] (tol %v) for N=2", low, high, ciEpsilon)
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
