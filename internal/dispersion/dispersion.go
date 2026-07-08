package dispersion

import (
	"math"

	"github.com/valpere/tumanomir/internal"
)

// Analyze computes D_pair, mean similarity and cluster entropy from N
// generated Go sources. Sources that fail to parse are rejected by the
// caller before Analyze (the generation loop retries them and counts
// discards); passing an unparseable source here yields ok=false pairs
// with zero similarity, which would skew the result — don't.
//
// Algorithm, in order: (1) extract each source's AST feature vector,
// dropping any that don't parse; (2) if fewer than 2 valid sources remain,
// there is no pair to compare — return early with just N populated;
// (3) compute the full pairwise cosine-similarity matrix; (4) MeanSim/DPair
// come directly off that matrix, independent of simThreshold; (5) DPairCILow/
// DPairCIHigh bootstrap a 95% CI around DPair (REQ-MSR-07), advisory only —
// same status as H/HNorm below, never gated; (6) H/HNorm additionally
// cluster the matrix at simThreshold and take entropy over cluster sizes —
// this is the only place simThreshold affects the result.
func Analyze(sources [][]byte, simThreshold float64) internal.DispersionResult {
	res := internal.DispersionResult{SimThresh: simThreshold}

	feats := make([]map[string]float64, 0, len(sources))
	for _, src := range sources {
		f, ok := features(src)
		if !ok {
			continue // defensive; the generation loop should have filtered
		}
		feats = append(feats, f)
	}
	n := len(feats)
	res.N = n
	if n < 2 {
		// Fewer than 2 valid samples means zero pairs to compare — MeanSim,
		// DPair, H, HNorm all stay at their zero value rather than being
		// computed from a degenerate (empty or single-element) matrix,
		// which would otherwise divide by zero (pairs==0) or take
		// log2(1)==0 as HNorm's denominator.
		return res
	}

	// sims is the full symmetric N×N pairwise similarity matrix (diagonal
	// left at its zero value — a sample's similarity to itself is never
	// read by anything downstream). Built once and reused by both the
	// MeanSim/DPair sum below and singleLinkage's clustering.
	sims := make([][]float64, n)
	for i := range sims {
		sims[i] = make([]float64, n)
	}
	var sum float64
	var pairs int
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			s := cosine(feats[i], feats[j])
			sims[i][j], sims[j][i] = s, s
			sum += s
			pairs++
		}
	}

	res.MeanSim = sum / float64(pairs)
	res.DPair = 1 - res.MeanSim
	res.DPairCILow, res.DPairCIHigh = bootstrapDPairCI(feats, bootstrapCIB, bootstrapCISeed)
	res.H, res.Clusters = entropy(singleLinkage(sims, simThreshold))
	// HNorm normalizes H by the maximum possible entropy for N samples
	// (all N in separate singleton clusters, i.e. log2(N) bits) so H
	// values are comparable across runs with different N — raw H alone
	// saturates at log2(N) and so can't be compared directly between e.g.
	// an N=10 and an N=20 run.
	res.HNorm = res.H / math.Log2(float64(n))
	return res
}

// ValidGo reports whether src parses as a Go file with at least one
// extractable feature. Used by the generation loop to count invalid
// samples before retrying (REQ-MSR-05) — a thin public wrapper so callers
// outside this package never need the feature map itself, just the
// pass/fail signal.
func ValidGo(src []byte) bool {
	_, ok := features(src)
	return ok
}
