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

// features extracts a bag-of-features vector from Go source. ok is false
// when the source does not parse — the caller must treat such samples as
// invalid (they are signal, not garbage: see the article on invalid rate).
func features(src []byte) (map[string]float64, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "gen.go", src, 0)
	if err != nil {
		return nil, false
	}

	feat := map[string]float64{}
	add := func(k string) { feat[strings.ToLower(k)]++ }
	exprStr := func(e ast.Expr) string {
		var sb strings.Builder
		_ = printer.Fprint(&sb, fset, e)
		return strings.ToLower(sb.String())
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.TypeSpec:
			add("type:" + d.Name.Name)
			switch t := d.Type.(type) {
			case *ast.StructType:
				add("kind:" + d.Name.Name + ":struct")
				for _, fld := range t.Fields.List {
					ts := exprStr(fld.Type)
					if len(fld.Names) == 0 {
						add("embed:" + d.Name.Name + ":" + ts)
					}
					for _, nm := range fld.Names {
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
				add("kind:" + d.Name.Name + ":" + exprStr(d.Type))
			}
		case *ast.FuncDecl:
			recv := ""
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv = exprStr(d.Recv.List[0].Type) + "."
			}
			add("func:" + recv + d.Name.Name + ":" + exprStr(d.Type))
		case *ast.ValueSpec:
			for _, nm := range d.Names {
				add("const:" + nm.Name)
			}
		}
		return true
	})
	return feat, len(feat) > 0
}

func cosine(a, b map[string]float64) float64 {
	var dot, na, nb float64
	for k, va := range a {
		na += va * va
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	for _, vb := range b {
		nb += vb * vb
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
