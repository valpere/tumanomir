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
