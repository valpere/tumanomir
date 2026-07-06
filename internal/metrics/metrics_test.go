package metrics

import (
	"math"
	"testing"
)

const tracedSpec = `# Payments

@schema Transaction { id: UUID, amount: Decimal(10,2) @constraint(min: 0.01) }

1. [REQ-PAY-01] Validate incoming request against Transaction.
   -> [FUN-PAY-01] AcceptTransaction(tx Transaction) (Receipt, error)
2. [REQ-PAY-02] Log all operation results.
   -> [FUN-PAY-02] LogResult(txID, status, errorCode)
`

const hangingSpec = `# Payments

1. [REQ-PAY-01] Validate incoming request.
   -> [FUN-PAY-01] AcceptTransaction(tx)
2. [REQ-PAY-02] The system must be flexible and convenient.
3. [REQ-PAY-03] Log everything somehow.
`

func TestKDriftFullyTraced(t *testing.T) {
	res := KDrift([]byte(tracedSpec))
	if res.Requirements != 2 || res.Hanging != 0 || res.Value != 0 {
		t.Fatalf("want 2 reqs, 0 hanging; got %+v", res)
	}
}

func TestKDriftHanging(t *testing.T) {
	res := KDrift([]byte(hangingSpec))
	if res.Requirements != 3 || res.Hanging != 2 {
		t.Fatalf("want 3 reqs, 2 hanging; got %+v", res)
	}
	if math.Abs(res.Value-2.0/3.0) > 1e-9 {
		t.Fatalf("want K_drift 0.667, got %f", res.Value)
	}
	if len(res.HangingIDs) != 2 || res.HangingIDs[0] != "REQ-PAY-02" || res.HangingIDs[1] != "REQ-PAY-03" {
		t.Fatalf("wrong hanging IDs: %v", res.HangingIDs)
	}
}

func TestKDriftNoRequirements(t *testing.T) {
	res := KDrift([]byte("just prose, no markup at all"))
	if res.Requirements != 0 || res.Value != 0 {
		t.Fatalf("empty spec must yield zero result, got %+v", res)
	}
}

// The following guard the hand-written scanner (issue #66) against
// behavioral drift from the regexp it replaced: `\[(REQ-[A-Za-z0-9_-]+)\]`
// for tags, `->\s*\[(?:FUN|LOG|PHY)-[A-Za-z0-9_-]+\]` for edges.

func TestKDriftEdgeMarkerZeroWhitespace(t *testing.T) {
	// ->\s* allows zero whitespace between the arrow and the bracket.
	res := KDrift([]byte("[REQ-A-01] x\n->[FUN-A-01] y\n"))
	if res.Requirements != 1 || res.Hanging != 0 {
		t.Fatalf("want 1 req, 0 hanging (no space is still a valid edge); got %+v", res)
	}
}

func TestKDriftEdgeMarkerMultiWhitespace(t *testing.T) {
	// ->\s* is greedy over any run of \s = [\t\n\f\r ], not just single spaces.
	res := KDrift([]byte("[REQ-A-01] x\n->  \t\n [FUN-A-01] y\n"))
	if res.Requirements != 1 || res.Hanging != 0 {
		t.Fatalf("want 1 req, 0 hanging (mixed whitespace run is still valid); got %+v", res)
	}
}

func TestKDriftEdgeMarkerLogAndPhy(t *testing.T) {
	res := KDrift([]byte("[REQ-A-01] x\n-> [LOG-A-01] y\n[REQ-A-02] z\n-> [PHY-A-02] w\n"))
	if res.Requirements != 2 || res.Hanging != 0 {
		t.Fatalf("want 2 reqs, 0 hanging (LOG-/PHY- are valid edge kinds too); got %+v", res)
	}
}

func TestKDriftMalformedTagsNotCounted(t *testing.T) {
	tests := []struct {
		name string
		doc  string
	}{
		{"empty id", "[REQ-] x\n"},
		{"unterminated", "[REQ-A-01 x\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := KDrift([]byte(tt.doc))
			if res.Requirements != 0 {
				t.Fatalf("want 0 requirements for malformed tag %q, got %+v", tt.doc, res)
			}
		})
	}
}

func TestKDriftBackToBackTags(t *testing.T) {
	// No content between a tag's closing ']' and the next tag's '[' —
	// exercises the scanner's continuation point precisely.
	res := KDrift([]byte("[REQ-A-01][REQ-A-02]\n-> [FUN-A-02]\n"))
	if res.Requirements != 2 || res.Hanging != 1 || len(res.HangingIDs) != 1 || res.HangingIDs[0] != "REQ-A-01" {
		t.Fatalf("want 2 reqs, 1 hanging (REQ-A-01); got %+v", res)
	}
}

// TestKDriftMalformedTagThenValidTag guards findReqTag's continuation
// loop (fix-review, glm-5.1:cloud) — the only new branch the
// all-malformed and all-well-formed fixtures above don't exercise
// together: a rejected [REQ-] occurrence must not stop the scan; a
// well-formed tag afterward must still be found and correctly traced.
func TestKDriftMalformedTagThenValidTag(t *testing.T) {
	res := KDrift([]byte("[REQ-] [REQ-A-01]\n-> [FUN-A-01]\n"))
	if res.Requirements != 1 || res.Hanging != 0 {
		t.Fatalf("want 1 req (malformed [REQ-] skipped), 0 hanging; got %+v", res)
	}
}

// TestKDriftNonMatchingArrowThenValidEdge guards hasEdgeMarker's
// retry-continuation (kdrift.go line 99): a "->" that doesn't lead to a
// valid FUN-/LOG-/PHY- marker must not stop the scan — a genuine edge
// marker later in the same block must still be found. Without this
// branch working, a correctly-traced requirement would be misreported
// as hanging, corrupting K_drift, a gate metric (issue #71).
func TestKDriftNonMatchingArrowThenValidEdge(t *testing.T) {
	res := KDrift([]byte("[REQ-A-01] see below -> nothing here\n-> [FUN-A-01]\n"))
	if res.Requirements != 1 || res.Hanging != 0 {
		t.Fatalf("want 1 req, 0 hanging (non-matching arrow skipped, later valid edge found); got %+v", res)
	}
}

func TestDConstMarkersRaiseDensity(t *testing.T) {
	fog := DConst([]byte("the system should flexibly accept transactions from various providers"))
	sharp := DConst([]byte(tracedSpec))
	if fog.ConstraintMarkers != 0 {
		t.Fatalf("fog spec has no markers, got %d", fog.ConstraintMarkers)
	}
	if sharp.ConstraintMarkers == 0 || sharp.Value <= fog.Value {
		t.Fatalf("sharp spec must be denser: fog=%f sharp=%f", fog.Value, sharp.Value)
	}
}

func TestDConstEmptyDoc(t *testing.T) {
	if res := DConst(nil); res.Value != 0 {
		t.Fatalf("empty doc density must be 0, got %f", res.Value)
	}
}

// TestDConstPureMarkersYieldsOne guards REQ-CHK-03's disjoint marker/prose
// invariant: a document made ONLY of marker substrings must have zero
// leftover prose tokens, so density is exactly 1.0 — not 0.5, which is
// what len(bytes.Fields(doc)) over the un-blanked original produces (every
// marker byte-span re-appears as a whitespace-separated "prose" token).
func TestDConstPureMarkersYieldsOne(t *testing.T) {
	pure := "@schema @constraint [REQ- -> [FUN- -> [LOG- -> [PHY-"
	res := DConst([]byte(pure))
	if res.ConstraintMarkers != 6 {
		t.Fatalf("want 6 markers (one per constraintMarkers entry), got %+v", res)
	}
	if res.ProseTokens != 0 {
		t.Fatalf("pure-marker doc must have zero prose tokens, got %+v", res)
	}
	if res.Value != 1.0 {
		t.Fatalf("pure-marker doc must have density exactly 1.0, got %+v", res)
	}
}

// TestDConstMixedFixtureExactDensity is a regression guard against the
// marker/prose double-counting bug (#54): 25 hand-counted prose words that
// share no bytes with any marker, plus 6 hand-counted markers each isolated
// on its own line. Under the old code (res.ProseTokens = len(bytes.Fields(doc))
// over the un-blanked original) each marker line contributes 1-2 extra
// "prose" tokens too (@schema, @constraint, [REQ- each count as one token;
// "-> [FUN-", "-> [LOG-", "-> [PHY-" each split into two whitespace tokens
// "->" and the bracket piece), inflating the denominator from 25 to 34 and
// the density from 6/31=0.193548... down to 6/40=0.15. This test asserts
// the corrected 6/31 value, which the old code could not produce.
func TestDConstMixedFixtureExactDensity(t *testing.T) {
	mixed := `The team wrote a long document about payments and settlement flows using careful precise language throughout the specification for clarity and completeness across every corner.

@schema
@constraint
[REQ-
-> [FUN-
-> [LOG-
-> [PHY-
`
	res := DConst([]byte(mixed))
	if res.ConstraintMarkers != 6 {
		t.Fatalf("want 6 markers, got %+v", res)
	}
	if res.ProseTokens != 25 {
		t.Fatalf("want 25 prose tokens (hand-counted words sharing no bytes with any marker), got %+v", res)
	}
	want := 6.0 / 31.0
	if res.Value != want {
		t.Fatalf("want density %v (6/31), got %+v", want, res)
	}
}

// TestDConstMultiTokenMarkerBlanksBothPieces guards the byte-span-blanking
// approach against a naive per-whitespace-token exclusion: the marker
// "-> [FUN-" spans two whitespace-separated pieces ("->" and "[FUN-CHK-01]").
// Blanking only the matched marker bytes must leave exactly the true
// leftover ("CHK-01]") as a single prose token — not zero (both pieces
// dropped whole) and not two (neither piece recognized as marker-bearing).
func TestDConstMultiTokenMarkerBlanksBothPieces(t *testing.T) {
	res := DConst([]byte("-> [FUN-CHK-01]"))
	if res.ConstraintMarkers != 1 {
		t.Fatalf("want 1 marker, got %+v", res)
	}
	if res.ProseTokens != 1 {
		t.Fatalf("want exactly 1 leftover prose token (\"CHK-01]\"), got %+v", res)
	}
}
