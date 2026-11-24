package dispersion

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"strings"

	"github.com/valpere/tumanomir/internal"
)

// Decls reports, per declaration slot, how far the accepted samples agree
// (REQ-MSR-12). It is a purely diagnostic companion to Analyze: it adds no
// metric and is never gated. Slot matching is a heuristic (see matchTypes),
// not a measurement.
//
// Samples are processed in input order and filtered with the same validity
// check as Analyze (features), so the result's N always equals
// DispersionResult.N. Fewer than 2 valid samples yields nil.
func Decls(sources [][]byte) []internal.DeclAgreement {
	var samples []*sample
	for _, src := range sources {
		if _, ok := features(src); !ok {
			continue
		}
		// ponytail: second parse per sample (features() already parsed once),
		// kept to leave features() untouched; share the AST if this ever shows
		// up in a profile.
		if s := parseSample(src); s != nil {
			samples = append(samples, s)
		}
	}
	if len(samples) < 2 {
		return nil
	}
	slots := matchTypes(samples)
	return aggregate(samples, slots)
}

// sample is one parsed source plus the declarations Decls cares about.
type sample struct {
	fset  *token.FileSet
	types []*stype
	funcs []*ast.FuncDecl // free functions (no receiver)
}

// stype is one type declaration with everything slot matching needs.
type stype struct {
	name    string
	spec    *ast.TypeSpec
	methods []*ast.FuncDecl // receiver methods declared in this sample
	values  []string        // literals of consts explicitly typed as this type
	set     map[string]bool // signature set: member:<name> / value:<literal>
	slot    int
}

func parseSample(src []byte) *sample {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "gen.go", src, 0)
	if err != nil {
		return nil
	}
	s := &sample{fset: fset}
	byName := map[string]*stype{}
	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE {
			for _, sp := range g.Specs {
				ts := sp.(*ast.TypeSpec)
				t := &stype{name: ts.Name.Name, spec: ts, set: map[string]bool{}, slot: -1}
				if _, dup := byName[t.name]; !dup { // first declaration wins
					byName[t.name] = t
					s.types = append(s.types, t)
				}
			}
		}
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				s.funcs = append(s.funcs, d)
			} else if t := byName[recvName(d)]; t != nil {
				t.methods = append(t.methods, d)
			}
		case *ast.GenDecl:
			if d.Tok != token.CONST {
				continue
			}
			for _, sp := range d.Specs {
				vs := sp.(*ast.ValueSpec)
				id, ok := vs.Type.(*ast.Ident)
				if !ok || byName[id.Name] == nil {
					continue
				}
				owner := byName[id.Name]
				for _, v := range vs.Values {
					if bl, ok := v.(*ast.BasicLit); ok { // iota/implicit continuations add nothing
						owner.values = append(owner.values, bl.Value)
					}
				}
			}
		}
	}
	for _, t := range s.types {
		t.buildSet(s)
	}
	return s
}

// recvName is the receiver's base type name with `*` and type params stripped.
func recvName(d *ast.FuncDecl) string {
	e := d.Recv.List[0].Type
	for {
		switch x := e.(type) {
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.IndexListExpr:
			e = x.X
		case *ast.Ident:
			return x.Name
		default:
			return ""
		}
	}
}

// buildSet fills the type's signature set. member: and value: elements carry
// different prefixes, so struct/interface sets never cross-match enum sets.
func (t *stype) buildSet(s *sample) {
	switch u := t.spec.Type.(type) {
	case *ast.StructType:
		for _, f := range u.Fields.List {
			t.addFields(s, f)
		}
	case *ast.InterfaceType:
		for _, m := range u.Methods.List {
			t.addFields(s, m)
		}
	}
	for _, m := range t.methods {
		t.set["member:"+strings.ToLower(m.Name.Name)] = true
	}
	for _, v := range t.values {
		t.set["value:"+v] = true
	}
}

func (t *stype) addFields(s *sample, f *ast.Field) {
	if len(f.Names) == 0 {
		t.set["member:embed:"+strings.ToLower(s.print(f.Type))] = true
	}
	for _, n := range f.Names {
		t.set["member:"+strings.ToLower(n.Name)] = true
	}
}

func (s *sample) print(n ast.Node) string {
	var sb strings.Builder
	_ = printer.Fprint(&sb, s.fset, n)
	return sb.String()
}

// tslot is one cross-sample type slot.
type tslot struct {
	counts map[string]int  // spelling -> samples using it
	order  []string        // spellings, first seen first
	lower  map[string]bool // lowercased spellings
	set    map[string]bool // union of assigned signature sets
	filled int             // index of the last sample that filled this slot
}

func (sl *tslot) add(name string, set map[string]bool, sample int) {
	if sl.counts[name] == 0 {
		sl.order = append(sl.order, name)
	}
	sl.counts[name]++
	sl.lower[strings.ToLower(name)] = true
	for k := range set {
		sl.set[k] = true
	}
	sl.filled = sample
}

// label is the most frequent spelling.
func (sl *tslot) label() string {
	return mostFrequent(sl.order, sl.counts)
}

// mostFrequent returns the spelling with the highest count; ties go to the
// spelling seen first (order is first-seen order, so it must be non-empty).
func mostFrequent(order []string, counts map[string]int) string {
	best := order[0]
	for _, n := range order[1:] {
		if counts[n] > counts[best] {
			best = n
		}
	}
	return best
}

// matchTypes assigns every sample's types to cross-sample slots and records
// the slot index on each stype (two passes per sample):
//
//	(a) a type whose lowercased name equals any spelling already in a slot
//	    takes the first-created such slot this sample has not filled;
//	(b) every remaining type rename-matches the unfilled slot with the best
//	    Jaccard similarity of signature sets, requiring >= 0.5 and >= 2 shared
//	    elements (ties -> first-created slot; an empty set never matches);
//	    otherwise it opens a new slot.
//
// WHY the thresholds: one shared member or an empty set is too weak a signal
// to call two differently named types "the same declaration".
// PURPOSE: let naming noise (Store/SRSStore) fold into one slot so structural
// gaps stay visible. Heuristic diagnostic, not a measurement.
func matchTypes(samples []*sample) []*tslot {
	var slots []*tslot
	for si, s := range samples {
		var rest []*stype
		for _, t := range s.types {
			var hit *tslot
			for i, sl := range slots {
				if sl.lower[strings.ToLower(t.name)] && sl.filled != si {
					hit, t.slot = sl, i
					break
				}
			}
			if hit != nil {
				hit.add(t.name, t.set, si)
			} else {
				rest = append(rest, t)
			}
		}
		for _, t := range rest {
			best, bs, bu := -1, 0, 1
			for i, sl := range slots {
				if sl.filled == si {
					continue
				}
				shared := 0
				for k := range t.set {
					if sl.set[k] {
						shared++
					}
				}
				union := len(t.set) + len(sl.set) - shared
				if shared < 2 || 2*shared < union {
					continue
				}
				if best < 0 || shared*bu > bs*union { // strictly better; ties keep the earlier slot
					best, bs, bu = i, shared, union
				}
			}
			if best < 0 {
				slots = append(slots, &tslot{counts: map[string]int{}, lower: map[string]bool{}, set: map[string]bool{}, filled: -1})
				best = len(slots) - 1
			}
			t.slot = best
			slots[best].add(t.name, t.set, si)
		}
	}
	return slots
}

// acc accumulates one reported slot across samples.
type acc struct {
	kind, parent string
	counts       map[string]int
	order        []string
	shapes       map[string]int
	present      int
}

type accKey struct {
	typeSlot int // -1 for free funcs
	member   string
	isType   bool
}

func (a *acc) add(name, shape string) {
	if a.counts[name] == 0 {
		a.order = append(a.order, name)
	}
	a.counts[name]++
	a.shapes[shape]++
	a.present++
}

func aggregate(samples []*sample, slots []*tslot) []internal.DeclAgreement {
	labels := make([]string, len(slots))
	for i, sl := range slots {
		labels[i] = sl.label()
	}
	var accs []*acc
	byKey := map[accKey]*acc{}
	// seen drops a repeated declaration within one sample (e.g. a generated
	// duplicate func) so Present can never exceed N.
	var seen map[accKey]bool
	record := func(k accKey, kind, parent, name, shape string) {
		if seen[k] {
			return
		}
		seen[k] = true
		a := byKey[k]
		if a == nil {
			a = &acc{kind: kind, parent: parent, counts: map[string]int{}, shapes: map[string]int{}}
			byKey[k] = a
			accs = append(accs, a)
		}
		a.add(name, shape)
	}

	for _, s := range samples {
		seen = map[accKey]bool{}
		local := map[string]string{} // this sample's type name -> slot label
		for _, t := range s.types {
			local[t.name] = labels[t.slot]
		}
		// rewrite swaps local type names for slot labels in place. It is
		// applied exactly once per expression (so labels never chain), and
		// skips selector members and field/param names.
		var rewrite func(n ast.Node)
		rewrite = func(n ast.Node) {
			ast.Inspect(n, func(x ast.Node) bool {
				switch x := x.(type) {
				case *ast.SelectorExpr:
					rewrite(x.X)
					return false
				case *ast.Field:
					rewrite(x.Type)
					return false
				case *ast.Ident:
					if l, ok := local[x.Name]; ok {
						x.Name = l
					}
				}
				return true
			})
		}
		sig := func(ft *ast.FuncType) string {
			rewrite(ft)
			list := func(fl *ast.FieldList) []string {
				var out []string
				if fl == nil {
					return nil
				}
				for _, f := range fl.List {
					ts := s.print(f.Type)
					for i := max(1, len(f.Names)); i > 0; i-- {
						out = append(out, ts)
					}
				}
				return out
			}
			r := "(" + strings.Join(list(ft.Params), ", ") + ")"
			switch res := list(ft.Results); len(res) {
			case 0:
			case 1:
				r += " " + res[0]
			default:
				r += " (" + strings.Join(res, ", ") + ")"
			}
			return r
		}
		typeStr := func(e ast.Expr) string { rewrite(e); return s.print(e) }

		for _, t := range s.types {
			parent := labels[t.slot]
			shape := ""
			switch u := t.spec.Type.(type) {
			case *ast.StructType:
				shape = "struct"
				for _, f := range u.Fields.List {
					ts := typeStr(f.Type)
					if len(f.Names) == 0 {
						record(accKey{t.slot, "embed:" + strings.ToLower(ts), false}, "field", parent, ts, ts)
					}
					for _, n := range f.Names {
						record(accKey{t.slot, strings.ToLower(n.Name), false}, "field", parent, n.Name, ts)
					}
				}
			case *ast.InterfaceType:
				shape = "interface"
				for _, m := range u.Methods.List {
					ft, isFn := m.Type.(*ast.FuncType)
					if !isFn { // embedded interface
						ts := typeStr(m.Type)
						record(accKey{t.slot, "embed:" + strings.ToLower(ts), false}, "field", parent, ts, ts)
						continue
					}
					for _, n := range m.Names {
						record(accKey{t.slot, strings.ToLower(n.Name), false}, "method", parent, n.Name, sig(ft))
					}
				}
			default:
				shape = typeStr(t.spec.Type)
			}
			if len(t.values) > 0 {
				vals := append([]string(nil), t.values...)
				sort.Strings(vals)
				shape += "{" + strings.Join(vals, ",") + "}"
			}
			for _, m := range t.methods {
				record(accKey{t.slot, strings.ToLower(m.Name.Name), false}, "method", parent, m.Name.Name, sig(m.Type))
			}
			record(accKey{t.slot, "", true}, "type", "", t.name, shape)
		}
		for _, f := range s.funcs {
			record(accKey{-1, strings.ToLower(f.Name.Name), false}, "func", "", f.Name.Name, sig(f.Type))
		}
	}

	n := len(samples)
	out := make([]internal.DeclAgreement, 0, len(accs))
	for _, a := range accs {
		d := internal.DeclAgreement{Kind: a.kind, Parent: a.parent, Present: a.present, N: n}
		d.Name = mostFrequent(a.order, a.counts)
		if len(a.order) > 1 {
			for _, nm := range a.order {
				d.NameVariants = append(d.NameVariants, internal.NameCount{Name: nm, Count: a.counts[nm]})
			}
			sort.SliceStable(d.NameVariants, func(i, j int) bool {
				x, y := d.NameVariants[i], d.NameVariants[j]
				return x.Count > y.Count || x.Count == y.Count && x.Name < y.Name
			})
		}
		for sh, c := range a.shapes {
			d.Variants = append(d.Variants, internal.ShapeCount{Shape: sh, Count: c})
		}
		sort.Slice(d.Variants, func(i, j int) bool {
			x, y := d.Variants[i], d.Variants[j]
			return x.Count > y.Count || x.Count == y.Count && x.Shape < y.Shape
		})
		out = append(out, d)
	}
	// Total order, least agreed first: agreement (top variant count; N is
	// shared), then presence, then kind/parent/name; creation order (stable)
	// breaks any remaining tie.
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Variants[0].Count != b.Variants[0].Count {
			return a.Variants[0].Count < b.Variants[0].Count
		}
		if a.Present != b.Present {
			return a.Present < b.Present
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Parent != b.Parent {
			return a.Parent < b.Parent
		}
		return a.Name < b.Name
	})
	return out
}
