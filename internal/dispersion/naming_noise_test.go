package dispersion

import (
	"math"
	"testing"
)

// TestNamingNoiseDPair quantifies how much of D_pair's signal is naming-style
// variance versus real structural divergence (issue #106). Three fixtures
// (testdata/naming-noise/{1,2,3}.go) encode one single structural
// interpretation of a small spec — a struct with two fields (int, string),
// a function, and a const — varying ONLY identifier text across the three
// while holding field types, cardinality, and shape fixed. A second pair
// (testdata/naming-noise-reorder/{1,2}.go) holds everything fixed including
// identifiers, reordering only the struct field declarations — a control
// for the order-independence astfeat.go's map-keyed feature extraction
// already guarantees by construction.
//
// Measured reference numbers (observed once, then pinned — these fixtures
// and features()/cosine() are deterministic pure math with no randomness):
//
//	naming-noise  D_pair = 0.6667  MeanSim = 0.3333  N = 3
//	reorder       D_pair = 0.0000  MeanSim = 1.0000  N = 2
//
// The naming-noise D_pair of ~0.67 means two-thirds of the similarity
// space is consumed by identifier-choice alone when the structural
// interpretation is held fixed — a significant fraction of D_pair's
// signal on empty-body type skeletons (PromptV1's only generation target
// in v0.1) is naming-style noise, not structural divergence. This number
// is the input to a future Phase 2 go/no-go decision on whether to add
// an additive, non-gating D_pair_shape diagnostic (identifier-
// canonicalized feature keys); that decision is NOT made here — this
// test only measures and pins the baseline.
//
// Explicitly rejected alternatives (documented so they aren't re-suggested):
//   - Embedding-based synonym matching: reintroduces a non-deterministic
//     model dependency into a metric whose value is being a fixed,
//     reproducible, zero-additional-LLM instrument.
//   - Deterministic string edit-distance (e.g. Levenshtein): not a model
//     dependency, but matches morphology/typos not synonyms —
//     UserID vs ID vs Identifier have large edit distance despite being
//     exactly the naming variance measured here. Phase 2's drop-the-
//     identifier canonicalization strictly dominates it.
func TestNamingNoiseDPair(t *testing.T) {
	// --- naming-variance fixtures: same shape, different identifier text ---

	namingSources := loadGoFixtures(t, "testdata/naming-noise")
	namingRes := Analyze(namingSources, 0.95)

	// Pin the observed D_pair as a durable reference constant. These
	// fixtures and features()/cosine() are deterministic pure math —
	// "observe once, then pin" is correct, not premature (same pattern
	// as TestAnalyzeInstrumentASharp's existing reference-number asserts).
	//
	// D_pair = 2/3 exactly (3 variants share 2 of 6 feature keys each:
	// type:record and kind:record:struct; cosine = 2/6 = 1/3; D_pair =
	// 1 - 1/3 = 2/3 ≈ 0.6667).
	if diff := math.Abs(namingRes.DPair - 0.6667); diff > simTol {
		t.Fatalf("naming-noise D_pair = %.6f, want ~0.6667 (tol %.3f); result=%+v",
			namingRes.DPair, simTol, namingRes)
	}
	if diff := math.Abs(namingRes.MeanSim - 0.3333); diff > simTol {
		t.Fatalf("naming-noise MeanSim = %.6f, want ~0.3333 (tol %.3f); result=%+v",
			namingRes.MeanSim, simTol, namingRes)
	}

	// Assert the naming effect is real and non-trivial — D_pair must be
	// strictly greater than zero, confirming the naming variance produces
	// actual dissimilarity, not a fixture-construction artifact.
	if namingRes.DPair <= 0 {
		t.Fatalf("naming-noise D_pair = %.6f, must be strictly > 0 (naming effect is real)",
			namingRes.DPair)
	}

	// --- reorder control: same everything, field order reversed ---

	reorderSources := loadGoFixtures(t, "testdata/naming-noise-reorder")
	reorderRes := Analyze(reorderSources, 0.95)

	// Reorder D_pair must be at/near zero — confirms astfeat.go's
	// map-keyed feature extraction is order-independent (a regression
	// guard: if someone accidentally makes the feature extraction
	// position-dependent, this catches it).
	if diff := math.Abs(reorderRes.DPair - 0.0); diff > simTol {
		t.Fatalf("reorder D_pair = %.6f, want ~0.0000 (tol %.3f); result=%+v — "+
			"order-independence regression in astfeat.go?",
			reorderRes.DPair, simTol, reorderRes)
	}
	if diff := math.Abs(reorderRes.MeanSim - 1.0); diff > simTol {
		t.Fatalf("reorder MeanSim = %.6f, want ~1.0000 (tol %.3f); result=%+v",
			reorderRes.MeanSim, simTol, reorderRes)
	}

	// --- cross-check: naming noise strictly exceeds reorder control ---

	// The naming-variance D_pair must be strictly greater than the
	// reorder-control D_pair — confirming that identifier text change
	// produces more dissimilarity than a no-op reordering, i.e. the
	// naming effect is larger than zero (which the reorder already
	// demonstrated) and the fixtures are genuinely testing different
	// things.
	if namingRes.DPair <= reorderRes.DPair {
		t.Fatalf("naming D_pair (%.6f) must exceed reorder D_pair (%.6f) — "+
			"naming variance must produce more dissimilarity than a no-op reordering",
			namingRes.DPair, reorderRes.DPair)
	}
}
