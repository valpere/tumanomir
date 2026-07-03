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
