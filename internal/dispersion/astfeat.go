// Package dispersion implements the stochastic measurement layer: it turns
// N generated Go artifacts into AST feature vectors, computes pairwise
// cosine similarity, clusters them and derives D_pair and cluster entropy.
//
// Ported from the article's sanity-check analyzer
// (source_of_the_unknown/sanity/analyze). The feature vector is a bag of
// structural tokens — this measures *structural* similarity of generated
// artifacts, not semantic equivalence.
package dispersion

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"math"
	"strings"
)

// features extracts a bag-of-features vector from Go source: a map from a
// structural key (e.g. "field:point:x:int") to how many times that exact
// structure occurs. Two generations are "similar" (per cosine, below) to
// the extent their bags share keys with similar counts — this is why the
// keys encode *shape* (type name, field name, field type, method
// signature) rather than any semantic meaning of the code.
//
// ok is false when the source does not parse — the caller must treat such
// samples as invalid (they are signal, not garbage: see the article on
// invalid rate, and REQ-MSR-05's discard-rate reporting in
// internal/instrument). A source that parses but declares nothing
// (feat stays empty) also reports ok=false, via the len(feat) > 0 check —
// there is nothing to compare it against otherwise.
func features(src []byte) (map[string]float64, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "gen.go", src, 0)
	if err != nil {
		return nil, false
	}

	feat := map[string]float64{}
	// add records one occurrence of key k, lowercased so that e.g. two
	// generations differing only in identifier casing ("UserID" vs
	// "userId") still count as the same structural feature — casing is a
	// style choice an LLM makes per-generation, not a structural
	// difference D_pair should be sensitive to.
	add := func(k string) { feat[strings.ToLower(k)]++ }
	// exprStr renders a type expression (a field's type, a function's
	// signature, etc.) back to source text via go/printer, so e.g. a
	// field typed `map[string]int` becomes the literal string
	// "map[string]int" for use inside a feature key.
	exprStr := func(e ast.Expr) string {
		var sb strings.Builder
		_ = printer.Fprint(&sb, fset, e)
		return strings.ToLower(sb.String())
	}

	// Walk every node in the file, recording one feature key per
	// structurally-meaningful declaration. Each case below targets exactly
	// one Go declaration kind; unhandled node kinds (statements inside
	// function bodies, expressions, etc.) are deliberately not
	// featurized — this is a *declaration-shape* similarity measure, not a
	// full-body structural diff.
	ast.Inspect(f, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.TypeSpec:
			// Every named type contributes a "type:<Name>" key regardless
			// of its underlying kind, plus a kind-specific breakdown below.
			add("type:" + d.Name.Name)
			switch t := d.Type.(type) {
			case *ast.StructType:
				add("kind:" + d.Name.Name + ":struct")
				for _, fld := range t.Fields.List {
					ts := exprStr(fld.Type)
					if len(fld.Names) == 0 {
						// An embedded (anonymous) field — fld.Names is
						// empty because Go doesn't require a name for
						// embedding. Key on the struct + the embedded
						// type itself, since that's the only identifying
						// information available (there is no field name).
						add("embed:" + d.Name.Name + ":" + ts)
					}
					for _, nm := range fld.Names {
						// A regular named field: struct + field name +
						// field type, e.g. "field:point:x:int".
						add("field:" + d.Name.Name + ":" + nm.Name + ":" + ts)
					}
				}
			case *ast.InterfaceType:
				add("kind:" + d.Name.Name + ":interface")
				for _, m := range t.Methods.List {
					for _, nm := range m.Names {
						add("method:" + d.Name.Name + ":" + nm.Name + ":" + exprStr(m.Type))
					}
				}
			default:
				// A named type over anything else (an alias, a defined
				// primitive like `type ID string`, a slice/map type,
				// etc.) — record its underlying type expression as-is,
				// since there's no field/method breakdown to recurse into.
				add("kind:" + d.Name.Name + ":" + exprStr(d.Type))
			}
		case *ast.FuncDecl:
			// A function or method declaration. recv captures the
			// receiver type for methods (e.g. "point.") so a method and a
			// free function with the same name/signature don't collide in
			// the feature map — they're structurally different.
			recv := ""
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv = exprStr(d.Recv.List[0].Type) + "."
			}
			add("func:" + recv + d.Name.Name + ":" + exprStr(d.Type))
		case *ast.ValueSpec:
			// A top-level const/var declaration's name(s) — only the name
			// is featurized, not the assigned value or type, since the
			// value is data (potentially arbitrary), not shape.
			for _, nm := range d.Names {
				add("const:" + nm.Name)
			}
		}
		return true
	})
	return feat, len(feat) > 0
}

// cosine computes the cosine similarity between two sparse feature vectors
// a and b (as produced by features), treating each map as a vector over the
// union of both maps' keys with an implicit 0 for any key absent from one
// side. Result is in [0,1] for these non-negative count vectors: 1.0 means
// identical feature bags, 0.0 means no shared features at all.
func cosine(a, b map[string]float64) float64 {
	var dot, na, nb float64
	// Single pass over a computes both its squared norm (na) and the dot
	// product's contribution from keys a and b share — no need for a
	// second pass over a's keys once b is known.
	for k, va := range a {
		na += va * va
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	// b's squared norm needs its own pass regardless, since b may have
	// keys a doesn't (contributing to nb but not to dot).
	for _, vb := range b {
		nb += vb * vb
	}
	// A zero-vector (na or nb == 0, i.e. an empty feature map) would
	// divide by zero; not reachable via Analyze in practice (features
	// returns ok=false — filtered out — whenever its map would be empty),
	// but cosine is called directly enough elsewhere (tests) that the
	// guard is worth keeping explicit rather than relying on that
	// invariant holding at every call site.
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
