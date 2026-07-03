package metrics

import (
	"bytes"

	"github.com/valpere/tumanomir/internal"
)

// constraintMarkers are lexical signals of machine-readable engineering
// facts in spec markup. A cheap proxy for the graph-based D_const: it
// counts markers, not typed edges — first line of defense before graph
// construction, not a replacement for it.
var constraintMarkers = [][]byte{
	[]byte("@schema"),
	[]byte("@constraint"),
	[]byte("[REQ-"),
	[]byte("-> [FUN-"),
	[]byte("-> [LOG-"),
	[]byte("-> [PHY-"),
}

// DConst computes lexical constraint density: markers / (markers + prose
// tokens). Prose tokens are whitespace-separated words outside markers.
func DConst(doc []byte) internal.DConstResult {
	var res internal.DConstResult

	for i := 0; i < len(doc); {
		matched := 0
		for _, mk := range constraintMarkers {
			if len(doc)-i >= len(mk) && bytes.Equal(doc[i:i+len(mk)], mk) {
				matched = len(mk)
				break
			}
		}
		if matched > 0 {
			res.ConstraintMarkers++
			i += matched
			continue
		}
		i++
	}

	// Words counted independently of marker scan: a marker inside a line
	// still leaves surrounding prose as prose.
	res.ProseTokens = len(bytes.Fields(doc))

	if total := res.ConstraintMarkers + res.ProseTokens; total > 0 {
		res.Value = float64(res.ConstraintMarkers) / float64(total)
	}
	return res
}
