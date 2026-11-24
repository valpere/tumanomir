package dispersion

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
	"github.com/valpere/tumanomir/internal/report"
)

const declsFixtureDir = "testdata/dreet-dojo-req-uc-32cc10f"

func srcs(ss ...string) [][]byte {
	out := make([][]byte, len(ss))
	for i, s := range ss {
		out[i] = []byte(s)
	}
	return out
}

// find returns every slot of kind whose label equals name (case-insensitive).
func find(ds []internal.DeclAgreement, kind, name string) []internal.DeclAgreement {
	var out []internal.DeclAgreement
	for _, d := range ds {
		if d.Kind == kind && strings.EqualFold(d.Name, name) {
			out = append(out, d)
		}
	}
	return out
}

func mustOne(t *testing.T, ds []internal.DeclAgreement, kind, name string) internal.DeclAgreement {
	t.Helper()
	got := find(ds, kind, name)
	if len(got) != 1 {
		t.Fatalf("want exactly one %s %q slot, got %d: %+v", kind, name, len(got), ds)
	}
	return got[0]
}

func variantNames(d internal.DeclAgreement) []string {
	var out []string
	for _, v := range d.NameVariants {
		out = append(out, v.Name)
	}
	sort.Strings(out)
	return out
}

func TestDeclsExactMatchUnanimous(t *testing.T) {
	a := "package a\ntype S struct{ X int }\nfunc F(a int) error { return nil }\n"
	ds := Decls(srcs(a, a))
	for _, d := range ds {
		if d.Present != 2 || d.N != 2 || len(d.Variants) != 1 || len(d.NameVariants) != 0 {
			t.Fatalf("expected unanimous slot, got %+v", d)
		}
	}
	if len(ds) != 3 { // type S, field X, func F
		t.Fatalf("want 3 slots, got %+v", ds)
	}
}

func TestDeclsRenameByMembers(t *testing.T) {
	a := "package a\ntype Store interface{ A(); B(); C() }\n"
	b := "package a\ntype SRSStore interface{ A(); B(); D() }\n" // 2 shared / 4 union = 0.5
	ds := Decls(srcs(a, b))
	s := mustOne(t, ds, "type", "Store")
	if s.Present != 2 || !reflect.DeepEqual(variantNames(s), []string{"SRSStore", "Store"}) {
		t.Fatalf("got %+v", s)
	}
}

func TestDeclsRenameByEnumValues(t *testing.T) {
	a := "package a\ntype Format string\nconst (\n\tA Format = \"x\"\n\tB Format = \"y\"\n)\n"
	b := "package a\ntype CardFormat string\nconst (\n\tC CardFormat = \"x\"\n\tD CardFormat = \"y\"\n)\n"
	ds := Decls(srcs(a, b))
	s := mustOne(t, ds, "type", "Format")
	if s.Present != 2 || len(s.NameVariants) != 2 {
		t.Fatalf("got %+v", s)
	}
	if len(s.Variants) != 1 {
		t.Fatalf("same value set must be one shape, got %+v", s.Variants)
	}
}

func TestDeclsEnumValueSetsDoNotCrossMatchMembers(t *testing.T) {
	a := "package a\ntype E string\nconst (\n\tA E = \"x\"\n\tB E = \"y\"\n)\n"
	b := "package a\ntype S struct{ x, y int }\n"
	if got := len(find(Decls(srcs(a, b)), "type", "E")) + len(find(Decls(srcs(a, b)), "type", "S")); got != 2 {
		t.Fatalf("enum and struct must stay separate slots")
	}
}

func TestDeclsNoRenameOnEmptyOrSingleSharedElement(t *testing.T) {
	cases := map[string][2]string{
		"empty":  {"package a\ntype A struct{}\n", "package a\ntype B struct{}\n"},
		"1shard": {"package a\ntype A struct{ X int }\n", "package a\ntype B struct{ X int }\n"},
	}
	for name, c := range cases {
		ds := Decls(srcs(c[0], c[1]))
		if len(find(ds, "type", "A")) != 1 || len(find(ds, "type", "B")) != 1 {
			t.Fatalf("%s: A and B must stay separate slots, got %+v", name, ds)
		}
		if mustOne(t, ds, "type", "A").Present != 1 {
			t.Fatalf("%s: got %+v", name, ds)
		}
	}
}

func TestDeclsStructReceiverMethodsMatchInterface(t *testing.T) {
	a := "package a\ntype Service interface{ Hint(c int) string; Diff(c int) string }\n"
	b := "package a\ntype SessionService struct{}\nfunc (s *SessionService) Hint(c int) string { return \"\" }\nfunc (s *SessionService) Diff(c int) string { return \"\" }\n"
	ds := Decls(srcs(a, b))
	s := mustOne(t, ds, "type", "Service")
	if s.Present != 2 {
		t.Fatalf("got %+v", s)
	}
	h := mustOne(t, ds, "method", "Hint")
	if h.Present != 2 || len(h.Variants) != 1 || h.Parent != "Service" {
		t.Fatalf("receiver method must share the interface method slot, got %+v", h)
	}
	if len(find(ds, "func", "Hint")) != 0 {
		t.Fatalf("receiver methods are not free funcs: %+v", ds)
	}
}

func TestDeclsAmbiguousCandidateAssignedToFirstSlot(t *testing.T) {
	a := "package a\ntype P interface{ A(); B(); C() }\ntype Q interface{ A(); B(); D() }\n"
	b := "package a\ntype R interface{ A(); B(); C(); D() }\n" // 3/4 with both
	for i := 0; i < 20; i++ {
		ds := Decls(srcs(a, b))
		p := mustOne(t, ds, "type", "P")
		q := mustOne(t, ds, "type", "Q")
		if p.Present != 2 || q.Present != 1 || !reflect.DeepEqual(variantNames(p), []string{"P", "R"}) {
			t.Fatalf("tie must go to first-created slot, got P=%+v Q=%+v", p, q)
		}
	}
}

func TestDeclsGroupedParamsAndParamNamesDoNotSplitShapes(t *testing.T) {
	a := "package a\nfunc F(a, b string, c int) (x, y error) { return }\n"
	b := "package a\nfunc F(p string, q string, r int) (error, error) { return }\n"
	f := mustOne(t, Decls(srcs(a, b)), "func", "F")
	if len(f.Variants) != 1 || f.Variants[0].Shape != "(string, string, int) (error, error)" {
		t.Fatalf("got %+v", f.Variants)
	}
}

func TestDeclsLocalTypeNamesRewrittenToSlotLabel(t *testing.T) {
	a := "package a\ntype Res struct{ A int; B int }\nfunc F(r *Res) []Res { return nil }\n"
	b := "package a\ntype Result struct{ A int; B int }\nfunc F(r *Result) []Result { return nil }\n"
	f := mustOne(t, Decls(srcs(a, b)), "func", "F")
	if len(f.Variants) != 1 || f.Variants[0].Shape != "(*Res) []Res" {
		t.Fatalf("got %+v", f.Variants)
	}
}

func TestDeclsMemberCaseFolding(t *testing.T) {
	a := "package a\ntype S struct{}\nfunc (s S) GiveUp() {}\n"
	b := "package a\ntype S struct{}\nfunc (s S) Giveup() {}\n"
	m := mustOne(t, Decls(srcs(a, b)), "method", "GiveUp")
	if m.Present != 2 || len(m.NameVariants) != 2 || m.NameVariants[0].Name != "GiveUp" {
		t.Fatalf("got %+v", m)
	}
}

func TestDeclsOrderingAndDeterminism(t *testing.T) {
	a := "package a\nfunc F(a int)\nfunc G()\n"
	b := "package a\nfunc F(a string)\nfunc G()\nfunc H()\n"
	c := "package a\nfunc F(a int)\nfunc G()\n"
	first := Decls(srcs(a, b, c))
	for i := 0; i < 20; i++ {
		if got := Decls(srcs(a, b, c)); !reflect.DeepEqual(got, first) {
			t.Fatalf("non-deterministic output:\n%+v\n%+v", first, got)
		}
	}
	// agreement asc: H (1/3), F (2/3), G (3/3)
	var names []string
	for _, d := range first {
		names = append(names, d.Name)
	}
	if !reflect.DeepEqual(names, []string{"H", "F", "G"}) {
		t.Fatalf("order = %v", names)
	}
	if first[1].Variants[0].Count != 2 || first[1].Variants[1].Count != 1 {
		t.Fatalf("variants must be count desc: %+v", first[1].Variants)
	}
}

func TestDeclsFewerThanTwoValidIsNil(t *testing.T) {
	ok := "package a\nfunc F()\n"
	for name, in := range map[string][][]byte{
		"none":        nil,
		"one":         srcs(ok),
		"one+invalid": srcs(ok, "package a\nfunc F( {"),
	} {
		if got := Decls(in); got != nil {
			t.Fatalf("%s: want nil, got %+v", name, got)
		}
	}
}

func loadDeclsFixture(t *testing.T) ([][]byte, []byte) {
	t.Helper()
	valid := loadGoFixtures(t, declsFixtureDir)
	invalid, err := os.ReadFile(filepath.Join(declsFixtureDir, "05.go.invalid"))
	if err != nil {
		t.Fatal(err)
	}
	return valid, invalid
}

func TestDeclsFixtureInvalidSampleDropped(t *testing.T) {
	valid, invalid := loadDeclsFixture(t)
	if len(valid) != 9 || ValidGo(invalid) {
		t.Fatalf("fixture must have 9 valid samples and an unparseable 05.go.invalid (got %d)", len(valid))
	}
	with := append(append([][]byte{}, valid[:4]...), invalid)
	with = append(with, valid[4:]...)
	if !reflect.DeepEqual(Decls(with), Decls(valid)) {
		t.Fatal("Decls must drop the unparseable sample")
	}
	if n := Decls(valid)[0].N; n != Analyze(with, 0.95).N || n != 9 {
		t.Fatalf("Decls N=%d must equal Analyze N", n)
	}
}

// AC3: the real structural gaps buried under naming noise.
func TestDeclsFixtureStructuralGaps(t *testing.T) {
	valid, _ := loadDeclsFixture(t)
	ds := Decls(valid)

	counts := func(d internal.DeclAgreement) []int {
		var c []int
		for _, v := range d.Variants {
			c = append(c, v.Count)
		}
		return c
	}
	for _, c := range []struct {
		name string
		want []int
	}{
		{"Hint", []int{7, 2}},
		{"RecordAttempt", []int{6, 3}},
	} {
		d := mustOne(t, ds, "method", c.name)
		if d.Present != 9 || !reflect.DeepEqual(counts(d), c.want) {
			t.Fatalf("%s: want 9/9 with variants %v, got %+v", c.name, c.want, d)
		}
	}
	if d := mustOne(t, ds, "method", "Diff"); d.Present != 9 || len(d.Variants) != 3 {
		t.Fatalf("Diff: want 9/9 with 3 shapes, got %+v", d)
	}
	if d := mustOne(t, ds, "method", "Next"); d.Present != 9 || len(d.Variants) != 1 {
		t.Fatalf("Next: want a single-shape 9/9 slot, got %+v", d)
	}
}

// AC4: naming noise folds into slots instead of splitting them.
func TestDeclsFixtureNameMatching(t *testing.T) {
	valid, _ := loadDeclsFixture(t)
	ds := Decls(valid)

	for _, pair := range [][2]string{
		{"Store", "SRSStore"}, {"Service", "SessionService"},
		{"CheckResult", "Result"}, {"CheckStatus", "Status"}, {"Format", "CardFormat"},
	} {
		var hit *internal.DeclAgreement
		for i, d := range ds {
			if d.Kind != "type" {
				continue
			}
			names := variantNames(d)
			has := func(n string) bool {
				for _, x := range names {
					if x == n {
						return true
					}
				}
				return false
			}
			if has(pair[0]) && has(pair[1]) {
				hit = &ds[i]
			}
		}
		if hit == nil {
			t.Errorf("%v must form one type slot with name variants", pair)
		}
	}

	g := mustOne(t, ds, "method", "GiveUp")
	if len(g.NameVariants) < 2 {
		t.Errorf("GiveUp/Giveup must be one member slot with name variants: %+v", g)
	}

	bareResult := regexp.MustCompile(`\bResult\b`)
	sawLabel := false
	for _, d := range ds {
		for _, v := range d.Variants {
			if bareResult.MatchString(v.Shape) {
				t.Errorf("local type Result must render as CheckResult: %s %s: %s", d.Kind, d.Name, v.Shape)
			}
			if strings.Contains(v.Shape, "CheckResult") {
				sawLabel = true
			}
		}
	}
	if !sawLabel {
		t.Error("expected CheckResult in some shape")
	}
}

// TestDeclsFixtureRender renders the fixture's --decls block exactly as
// `measure --decls` would (log it with -v; the PR body quotes it) and pins
// the fixture's own Analyze numbers for its README.
func TestDeclsFixtureRender(t *testing.T) {
	valid, _ := loadDeclsFixture(t)
	res := Analyze(valid, 0.95)
	// Reference numbers pinned in testdata/dreet-dojo-req-uc-32cc10f/README.md
	// (this fixture's own, not the measured run's reported 0.19).
	for name, c := range map[string][2]float64{
		"D_pair": {res.DPair, 0.2462}, "CI low": {res.DPairCILow, 0.1895}, "CI high": {res.DPairCIHigh, 0.2972},
		"H": {res.H, 3.1699}, "clusters": {float64(res.Clusters), 9},
	} {
		if math.Abs(c[0]-c[1]) > 5e-4 {
			t.Errorf("%s = %.4f, want %.4f", name, c[0], c[1])
		}
	}
	var sb strings.Builder
	if err := report.RenderMeasure(&sb, report.MeasureResult{Dispersion: res, Decls: Decls(valid)}, internal.DefaultThresholds()); err != nil {
		t.Fatal(err)
	}
	t.Log("\n" + sb.String())
}
