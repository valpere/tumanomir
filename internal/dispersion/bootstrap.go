package dispersion

import (
	"math/rand/v2"
	"sort"
)

// bootstrapCIB is the number of bootstrap resamples used to estimate
// D_pair's 95% confidence interval (REQ-MSR-07). PURPOSE: 2000 resamples
// gives stable 2.5th/97.5th percentile estimates at the N (roughly 10-50
// samples per measure run) tumanomir actually generates, without making
// `measure` noticeably slower — resampling is O(B*N^2) cosine calls, cheap
// next to the LLM generation it follows.
const bootstrapCIB = 2000

// bootstrapCISeed fixes the bootstrap RNG.
// WHY: Analyze must stay a deterministic pure function — re-running it on
// identical stored artifacts has to yield a bit-identical CI. The
// instrument already has one source of randomness (LLM sampling) that
// tumanomir measures; adding a second, unseeded one on top would silently
// break the instrument-relative reproducibility invariant (REQ-MSR-04).
const bootstrapCISeed = 42

// bootstrapDPairCI computes a 95% percentile bootstrap confidence interval
// for D_pair over feats, the N AST feature vectors Analyze already
// extracted. It resamples the feature vectors themselves (with
// replacement, b times) rather than indices into the precomputed sims
// matrix: a bootstrap resample can draw the same original sample twice,
// and sims's diagonal is left at its zero value (nothing downstream of
// Analyze was ever meant to read a sample's similarity to itself) — using
// it here would score that self-pair as 0 instead of the correct 1.0.
// Calling cosine() fresh on every pair, including same-index pairs, gets
// self-similarity right by construction.
func bootstrapDPairCI(feats []map[string]float64, b int, seed uint64) (low, high float64) {
	n := len(feats)
	rng := rand.New(rand.NewPCG(seed, seed))
	dpairs := make([]float64, b)
	idx := make([]int, n)
	for i := 0; i < b; i++ {
		for j := range idx {
			idx[j] = rng.IntN(n)
		}
		var sum float64
		var pairs int
		for a := 0; a < n; a++ {
			for c := a + 1; c < n; c++ {
				sum += cosine(feats[idx[a]], feats[idx[c]])
				pairs++
			}
		}
		dpairs[i] = 1 - sum/float64(pairs)
	}
	sort.Float64s(dpairs)
	return percentile(dpairs, 0.025), percentile(dpairs, 0.975)
}

// percentile returns the value at fraction p (0..1) of sorted, a slice
// already in ascending order. Uses nearest-rank scaled by len-1 rather than
// interpolating between adjacent ranks — simple and, at B=2000 resamples,
// indistinguishable in practice from an interpolated percentile for this
// CI's purpose (an advisory uncertainty band, not a value being gated on).
func percentile(sorted []float64, p float64) float64 {
	idx := int(p * float64(len(sorted)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
