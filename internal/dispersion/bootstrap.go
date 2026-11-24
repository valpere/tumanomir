package dispersion

import (
	"math/rand/v2"
	"sort"
)

// bootstrapCIB is the number of bootstrap resamples used to estimate
// D_pair's 95% confidence interval (REQ-MSR-07). PURPOSE: 2000 resamples
// gives stable 2.5th/97.5th percentile estimates at the N (roughly 10-50
// samples per measure run) tumanomir actually generates, without making
// `measure` noticeably slower — resampling is O(B*N^2) matrix lookups,
// cheap next to the LLM generation it follows.
const bootstrapCIB = 2000

// bootstrapCISeed fixes the bootstrap RNG.
// WHY: Analyze must stay a deterministic pure function — re-running it on
// identical stored artifacts has to yield a bit-identical CI. The
// instrument already has one source of randomness (LLM sampling) that
// tumanomir measures; adding a second, unseeded one on top would silently
// break the instrument-relative reproducibility invariant (REQ-MSR-04).
const bootstrapCISeed = 42

// bootstrapMaxAttemptsFactor caps total resample attempts at factor*b.
// WHY: a resample with no distinct-original-index pair is redrawn; for
// N=1 (or any degenerate input) every draw is degenerate, so without a cap
// the loop would never end. Real inputs (N>=2) reject with probability
// N^(1-N) <= 1/2, far below the cap.
const bootstrapMaxAttemptsFactor = 100

// bootstrapDPairCI computes a 95% percentile bootstrap confidence interval
// for D_pair (REQ-MSR-07) from the bootstrap D* values of sims, the N×N
// pairwise similarity matrix Analyze already built. dpair is the point
// estimate, returned as (dpair, dpair) when no resample was collected.
func bootstrapDPairCI(sims [][]float64, dpair float64, b int, seed uint64) (low, high float64) {
	dpairs := bootstrapDPairs(sims, b, seed)
	if len(dpairs) == 0 {
		return dpair, dpair
	}
	sort.Float64s(dpairs)
	return percentile(dpairs, 0.025), percentile(dpairs, 0.975)
}

// bootstrapDPairs returns up to b bootstrap D* values (unsorted). Each
// resample draws N original indices with replacement and averages the
// similarity only over pairs whose two ORIGINAL indices differ: scoring a
// self-pair as 1 would center the bootstrap on (1-1/N)*D_pair and bias the
// CI low by ~1/N. The matrix diagonal is never read. A resample with no
// distinct pair is redrawn, within bootstrapMaxAttemptsFactor*b attempts
// in total; the slice is shorter than b (possibly empty) if the cap hits.
func bootstrapDPairs(sims [][]float64, b int, seed uint64) []float64 {
	n := len(sims)
	rng := rand.New(rand.NewPCG(seed, seed))
	dpairs := make([]float64, 0, b)
	idx := make([]int, n)
	for attempt := 0; attempt < bootstrapMaxAttemptsFactor*b && len(dpairs) < b; attempt++ {
		for j := range idx {
			idx[j] = rng.IntN(n)
		}
		var sum float64
		var pairs int
		for a := 0; a < n; a++ {
			for c := a + 1; c < n; c++ {
				if idx[a] == idx[c] {
					continue
				}
				sum += sims[idx[a]][idx[c]]
				pairs++
			}
		}
		if pairs == 0 {
			continue
		}
		dpairs = append(dpairs, 1-sum/float64(pairs))
	}
	return dpairs
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
