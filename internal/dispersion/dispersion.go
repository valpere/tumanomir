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
		return res
	}

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
	res.H, res.Clusters = entropy(singleLinkage(sims, simThreshold))
	res.HNorm = res.H / math.Log2(float64(n))
	return res
}

// ValidGo reports whether src parses as a Go file with at least one
// extractable feature. Used by the generation loop to count invalid
// samples before retrying.
func ValidGo(src []byte) bool {
	_, ok := features(src)
	return ok
}
