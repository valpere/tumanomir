// Package metrics implements the deterministic measurement layer:
// traceability (K_drift) and lexical constraint density (D_const).
// No LLM in the measurement loop.
package metrics

import (
	"regexp"

	"github.com/valpere/tumanomir/internal"
)

var (
	reqRe  = regexp.MustCompile(`\[(REQ-[A-Za-z0-9_-]+)\]`)
	edgeRe = regexp.MustCompile(`->\s*\[(?:FUN|LOG|PHY)-[A-Za-z0-9_-]+\]`)
)

// KDrift computes the relational drift coefficient in strict mode: a
// requirement is traced when at least one edge marker `-> [FUN-*]`
// (or LOG-/PHY-) appears between its [REQ-*] tag and the next one.
// Deterministic by construction — a pure scan over explicit markup.
func KDrift(doc []byte) internal.KDriftResult {
	var res internal.KDriftResult

	reqLocs := reqRe.FindAllSubmatchIndex(doc, -1)
	res.Requirements = len(reqLocs)
	if res.Requirements == 0 {
		return res
	}

	for i, loc := range reqLocs {
		blockEnd := len(doc)
		if i+1 < len(reqLocs) {
			blockEnd = reqLocs[i+1][0]
		}
		block := doc[loc[1]:blockEnd]
		if !edgeRe.Match(block) {
			res.Hanging++
			res.HangingIDs = append(res.HangingIDs, string(doc[loc[2]:loc[3]]))
		}
	}
	res.Value = float64(res.Hanging) / float64(res.Requirements)
	return res
}
