package metrics

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// buildBenchmarkCorpus returns a synthetic spec of at least targetBytes,
// built from repeating fully-traced requirement blocks — REQ-NFR-01's "1
// MB spec corpus" target, with realistic prose density (not a bare-tag
// worst case) so D_const sees a genuine markers-vs-prose ratio.
func buildBenchmarkCorpus(targetBytes int) []byte {
	var b strings.Builder
	for i := 0; b.Len() < targetBytes; i++ {
		fmt.Fprintf(&b, `%d. [REQ-BENCH-%d] The system must handle this synthetic
   requirement for benchmarking purposes, with enough surrounding prose
   to be realistic rather than a bare tag, mimicking real specification
   writing style and giving D_const a genuine markers-vs-prose ratio.
   -> [FUN-BENCH-%d] doSomething(x int) error

`, i+1, i, i)
	}
	return []byte(b.String())
}

// BenchmarkKDrift1MB verifies REQ-NFR-01: check on a 1 MB spec corpus
// must complete in under 100ms.
func BenchmarkKDrift1MB(b *testing.B) {
	doc := buildBenchmarkCorpus(1 << 20) // 1 MiB
	b.ReportAllocs()
	b.SetBytes(int64(len(doc)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		KDrift(doc)
	}
}

// BenchmarkDConst1MB is DConst's half of REQ-NFR-01's 1 MB / 100ms target.
func BenchmarkDConst1MB(b *testing.B) {
	doc := buildBenchmarkCorpus(1 << 20)
	b.ReportAllocs()
	b.SetBytes(int64(len(doc)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DConst(doc)
	}
}

// BenchmarkCheck1MB runs both metrics REQ-NFR-01 actually gates — K_drift
// and D_const — back to back per iteration, mirroring what a single-file
// `tumanomir check` invocation does. Reported separately from
// BenchmarkKDrift1MB/BenchmarkDConst1MB (fix-review, glm-5.1:cloud) so the
// 100ms budget is checked end-to-end, not inferred by manually summing
// two isolated benchmark numbers.
func BenchmarkCheck1MB(b *testing.B) {
	doc := buildBenchmarkCorpus(1 << 20)
	b.ReportAllocs()
	b.SetBytes(int64(len(doc)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		KDrift(doc)
		DConst(doc)
	}
}

// TestCheckPerformanceBudget turns REQ-NFR-01's "verified" claim into an
// enforced CI gate rather than a number a human has to eyeball in
// `go test -bench` output (issue #67): a `check` pass on a 1MB corpus
// must stay well within the 100ms target. The ceiling is deliberately
// generous — ~30x the ~17ms observed on typical dev hardware (see
// BenchmarkCheck1MB/REQ-NFR-01) — to absorb
// slower/noisier CI runners without becoming flaky, while still catching
// a genuine multi-fold regression (e.g. an accidental return to O(n^2)
// scanning, or K_drift's allocation-flat rewrite from #66 regressing).
func TestCheckPerformanceBudget(t *testing.T) {
	const ceiling = 500 * time.Millisecond
	doc := buildBenchmarkCorpus(1 << 20)

	start := time.Now()
	KDrift(doc)
	DConst(doc)
	elapsed := time.Since(start)

	if elapsed > ceiling {
		t.Fatalf("check (KDrift+DConst) on a 1MB corpus took %v, want under %v — REQ-NFR-01's 100ms target with a generous CI-noise margin", elapsed, ceiling)
	}
}

// TestKDriftAllocationBudget guards #66's allocation-flat rewrite:
// KDrift's allocation count on a 1MB/~3400-requirement corpus must stay
// low, not silently regress back toward the old regexp-based
// implementation's ~1:1-with-requirement-count scaling (was 3260
// allocs/op; the rewrite brought it to ~14). The ceiling is well above
// the current baseline (headroom for minor Go-runtime-version drift in
// slice growth behavior) but far below what per-match scaling on ~3400
// requirements would produce, so only a real structural regression trips
// it.
func TestKDriftAllocationBudget(t *testing.T) {
	const ceiling = 200
	doc := buildBenchmarkCorpus(1 << 20)

	allocs := testing.AllocsPerRun(10, func() {
		KDrift(doc)
	})
	if allocs > ceiling {
		t.Fatalf("KDrift on a 1MB corpus allocated %.0f times, want under %d (allocation-flat baseline is ~14; per-match scaling on ~3400 requirements would be in the thousands)", allocs, ceiling)
	}
}

// TestDConstAllocationBudget guards D_const's allocation-flat baseline
// (2 allocs/op: the blanked-doc copy from #54's marker/prose disjoint-set
// fix, plus bytes.Fields' result) against a future accidental regression,
// e.g. a per-marker allocation reintroduced into the scan loop.
func TestDConstAllocationBudget(t *testing.T) {
	const wantAllocs = 2
	doc := buildBenchmarkCorpus(1 << 20)

	allocs := testing.AllocsPerRun(10, func() {
		DConst(doc)
	})
	if allocs != wantAllocs {
		t.Fatalf("DConst on a 1MB corpus allocated %.0f times, want exactly %d (allocation-flat baseline)", allocs, wantAllocs)
	}
}
