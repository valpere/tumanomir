// Package metrics implements the deterministic measurement layer:
// traceability (K_drift) and lexical constraint density (D_const).
// No LLM in the measurement loop.
package metrics

import (
	"bytes"

	"github.com/valpere/tumanomir/internal"
)

var (
	reqTagPrefix = []byte("[REQ-")
	edgeArrow    = []byte("->")
	edgePrefixes = [][]byte{[]byte("[FUN-"), []byte("[LOG-"), []byte("[PHY-")}
)

// isIDChar reports whether b can appear inside a REQ-*/FUN-*/LOG-*/PHY-*
// identifier tail, matching the character class [A-Za-z0-9_-].
func isIDChar(b byte) bool {
	return b == '_' || b == '-' ||
		(b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// isSpace reports whether b is an ASCII whitespace byte, matching RE2's
// \s class: tab, newline, form feed, carriage return, space.
func isSpace(b byte) bool {
	switch b {
	case '\t', '\n', '\f', '\r', ' ':
		return true
	}
	return false
}

// findReqTag scans doc from offset i for the next well-formed [REQ-*]
// tag — equivalent to regexp `\[(REQ-[A-Za-z0-9_-]+)\]` but without a
// per-match heap allocation (issue #66). Returns the tag's byte span
// [start,end) (brackets included) and the identifier's span
// [idStart,idEnd) (brackets excluded); found is false once no further
// tag exists at or after i.
func findReqTag(doc []byte, i int) (start, end, idStart, idEnd int, found bool) {
	for {
		rel := bytes.Index(doc[i:], reqTagPrefix)
		if rel < 0 {
			return 0, 0, 0, 0, false
		}
		p := i + rel
		idStart = p + 1 // right after '[' — the reported ID includes "REQ-"
		afterPrefix := p + len(reqTagPrefix)
		j := afterPrefix
		for j < len(doc) && isIDChar(doc[j]) {
			j++
		}
		if j > afterPrefix && j < len(doc) && doc[j] == ']' {
			return p, j + 1, idStart, j, true
		}
		// Not a valid tag at this position (empty or unterminated ID) —
		// keep scanning for a later occurrence.
		i = p + 1
	}
}

// hasEdgeMarker reports whether block contains a trace edge — equivalent
// to regexp `->\s*\[(?:FUN|LOG|PHY)-[A-Za-z0-9_-]+\]`, without allocating.
func hasEdgeMarker(block []byte) bool {
	i := 0
	for {
		rel := bytes.Index(block[i:], edgeArrow)
		if rel < 0 {
			return false
		}
		p := i + rel
		j := p + len(edgeArrow)
		for j < len(block) && isSpace(block[j]) {
			j++
		}
		for _, prefix := range edgePrefixes {
			if j+len(prefix) > len(block) || !bytes.Equal(block[j:j+len(prefix)], prefix) {
				continue
			}
			k := j + len(prefix)
			for k < len(block) && isIDChar(block[k]) {
				k++
			}
			if k > j+len(prefix) && k < len(block) && block[k] == ']' {
				return true
			}
		}
		i = p + 1
	}
}

// reqTag is one located [REQ-*] occurrence, kept minimal (four ints) so
// the accumulating slice stays cheap to grow.
type reqTag struct {
	start, end     int // tag span, brackets included
	idStart, idEnd int // identifier span, brackets excluded
}

// KDrift computes the relational drift coefficient in strict mode: a
// requirement is traced when at least one edge marker `-> [FUN-*]`
// (or LOG-/PHY-) appears between its [REQ-*] tag and the next one.
// Deterministic by construction — a pure scan over explicit markup.
//
// Implemented as a hand-written byte scanner rather than
// regexp.FindAllSubmatchIndex (issue #66): the regexp package allocates
// one []int per match, so allocations scaled ~1:1 with requirement count
// (3260 allocs/op on a 1MB/~3400-requirement benchmark corpus, see
// BenchmarkKDrift1MB). This scanner's own allocations are limited to the
// tags slice's O(log n) growth-reallocations and the final HangingIDs
// result, which is proportional to the hanging count being reported, not
// the total requirement count scanned.
func KDrift(doc []byte) internal.KDriftResult {
	var res internal.KDriftResult

	var tags []reqTag
	for i := 0; i < len(doc); {
		start, end, idStart, idEnd, ok := findReqTag(doc, i)
		if !ok {
			break
		}
		tags = append(tags, reqTag{start: start, end: end, idStart: idStart, idEnd: idEnd})
		i = end
	}
	res.Requirements = len(tags)
	if res.Requirements == 0 {
		return res
	}

	for i, t := range tags {
		blockEnd := len(doc)
		if i+1 < len(tags) {
			blockEnd = tags[i+1].start
		}
		if !hasEdgeMarker(doc[t.end:blockEnd]) {
			res.Hanging++
			res.HangingIDs = append(res.HangingIDs, string(doc[t.idStart:t.idEnd]))
		}
	}
	res.Value = float64(res.Hanging) / float64(res.Requirements)
	return res
}
