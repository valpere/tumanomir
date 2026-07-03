package dispersion

import "math"

// singleLinkage joins samples i,j into one cluster when sim >= threshold,
// via union-find. Known limitation (part of the instrument definition):
// chaining — A~B, B~C merges A and C even when A is far from C.
func singleLinkage(sims [][]float64, threshold float64) []int {
	n := len(sims)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if sims[i][j] >= threshold {
				parent[find(i)] = find(j)
			}
		}
	}
	for i := range parent {
		parent[i] = find(i)
	}
	return parent
}

// entropy returns Shannon entropy (bits) over cluster sizes and the
// cluster count.
func entropy(cluster []int) (float64, int) {
	counts := map[int]int{}
	for _, c := range cluster {
		counts[c]++
	}
	n := float64(len(cluster))
	var h float64
	for _, c := range counts {
		p := float64(c) / n
		h -= p * math.Log2(p)
	}
	return h, len(counts)
}
