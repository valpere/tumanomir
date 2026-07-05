package metrics

import (
	"fmt"
	"strings"
	"testing"
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
