package dispersion

import "math"

// singleLinkage groups n samples into clusters via a union-find
// (disjoint-set) structure: samples i and j are merged into the same
// cluster whenever their pairwise similarity sims[i][j] meets or exceeds
// threshold. Returns a slice where result[i] is sample i's cluster
// representative (an arbitrary sample index acting as the cluster's ID —
// not itself meaningful, only useful for grouping via equality, see
// entropy below).
//
// "Single-linkage" means a merge only requires ONE sufficiently-similar
// pair, not that every pair within the resulting cluster meets threshold.
// Known limitation (part of the instrument definition, not a bug):
// chaining — A~B and B~C merges A, B, and C into one cluster even if A and
// C themselves are nowhere near threshold-similar to each other.
func singleLinkage(sims [][]float64, threshold float64) []int {
	n := len(sims)
	// parent[i] starts as i (n singleton clusters); union merges shrink
	// the number of distinct representatives reachable via find.
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	// find follows parent pointers to a cluster's representative,
	// path-compressing along the way (parent[x] = find(parent[x])) so
	// future find(x) calls resolve in ~O(1) rather than re-walking the
	// same chain.
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	// One pass over every unordered pair (i,j): union their clusters
	// whenever sims[i][j] clears threshold. Order doesn't matter for
	// correctness — find always resolves to the same representative
	// regardless of which merge direction was taken.
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if sims[i][j] >= threshold {
				parent[find(i)] = find(j)
			}
		}
	}
	// Final pass: resolve every entry to its ultimate representative, so
	// callers can group by simple equality on the returned slice without
	// needing to know about path compression.
	for i := range parent {
		parent[i] = find(i)
	}
	return parent
}

// entropy computes Shannon entropy (in bits) over the distribution of
// cluster sizes produced by singleLinkage, plus the distinct cluster
// count. High entropy (clusters roughly equal-sized, many of them) signals
// a wide spread of structurally distinct generations; low entropy (one
// dominant cluster) signals the instrument converges on one shape almost
// every time — this is the raw H that internal.DispersionResult.HNorm
// normalizes by log2(N) for cross-N comparability.
func entropy(cluster []int) (float64, int) {
	// counts[representative] = how many samples landed in that cluster.
	counts := map[int]int{}
	for _, c := range cluster {
		counts[c]++
	}
	n := float64(len(cluster))
	var h float64
	// Standard Shannon entropy: H = -sum(p_i * log2(p_i)) over each
	// cluster's proportion p_i = size_i/n. A singleton cluster (p_i small)
	// contributes little; an even split across many clusters maximizes H.
	for _, c := range counts {
		p := float64(c) / n
		h -= p * math.Log2(p)
	}
	return h, len(counts)
}
