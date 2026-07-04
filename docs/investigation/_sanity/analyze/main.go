// Command analyze computes H_spec (generation dispersion) over N generated
// Go files per specification: AST feature vectors -> pairwise cosine
// similarity -> single-linkage clustering at a threshold -> Shannon entropy
// over cluster sizes.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type sample struct {
	name     string
	features map[string]float64
	valid    bool
}

func exprString(fset *token.FileSet, e ast.Expr) string {
	var sb strings.Builder
	_ = printer.Fprint(&sb, fset, e)
	return strings.ToLower(sb.String())
}

func extractFeatures(path string) (map[string]float64, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, false
	}
	feat := map[string]float64{}
	add := func(k string) { feat[strings.ToLower(k)]++ }

	ast.Inspect(f, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.TypeSpec:
			add("type:" + d.Name.Name)
			switch t := d.Type.(type) {
			case *ast.StructType:
				add("kind:" + d.Name.Name + ":struct")
				for _, fld := range t.Fields.List {
					ts := exprString(fset, fld.Type)
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
						add("method:" + d.Name.Name + ":" + nm.Name + ":" + exprString(fset, m.Type))
					}
				}
			default:
				add("kind:" + d.Name.Name + ":" + exprString(fset, d.Type))
			}
		case *ast.FuncDecl:
			recv := ""
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv = exprString(fset, d.Recv.List[0].Type) + "."
			}
			add("func:" + recv + d.Name.Name + ":" + exprString(fset, d.Type))
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

// singleLinkage clusters indices: i,j joined when sim >= threshold.
func singleLinkage(sims [][]float64, valid []bool, threshold float64) []int {
	n := len(sims)
	cluster := make([]int, n)
	for i := range cluster {
		cluster[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if cluster[x] != x {
			cluster[x] = find(cluster[x])
		}
		return cluster[x]
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			// invalid samples never join anything: each is its own cluster
			if valid[i] && valid[j] && sims[i][j] >= threshold {
				cluster[find(i)] = find(j)
			}
		}
	}
	for i := range cluster {
		cluster[i] = find(i)
	}
	return cluster
}

func entropy(cluster []int) (float64, int) {
	counts := map[int]int{}
	for _, c := range cluster {
		counts[c]++
	}
	n := float64(len(cluster))
	var h float64
	for _, c := range counts {
		p := float64(c) / n
		h -= p * math.Log2(p)
	}
	return h, len(counts)
}

func analyzeSpec(dir string, thresholds []float64) error {
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	var samples []sample
	for _, p := range paths {
		feat, ok := extractFeatures(p)
		samples = append(samples, sample{name: filepath.Base(p), features: feat, valid: ok})
	}
	n := len(samples)
	if n == 0 {
		return fmt.Errorf("no .go files in %s", dir)
	}

	sims := make([][]float64, n)
	valid := make([]bool, n)
	invalid := 0
	for i := range samples {
		sims[i] = make([]float64, n)
		valid[i] = samples[i].valid
		if !samples[i].valid {
			invalid++
		}
	}
	var sumSim float64
	var pairs int
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			s := 0.0
			if valid[i] && valid[j] {
				s = cosine(samples[i].features, samples[j].features)
			}
			sims[i][j], sims[j][i] = s, s
			sumSim += s
			pairs++
		}
	}

	fmt.Printf("%-14s n=%d invalid=%d meanPairwiseSim=%.3f", filepath.Base(dir), n, invalid, sumSim/float64(pairs))
	for _, th := range thresholds {
		h, k := entropy(singleLinkage(sims, valid, th))
		fmt.Printf("  H@%.2f=%.3f (k=%d)", th, h, k)
	}
	fmt.Println()
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: analyze <out-dir> [spec...]")
		os.Exit(1)
	}
	root := os.Args[1]
	specs := os.Args[2:]
	if len(specs) == 0 {
		entries, _ := os.ReadDir(root)
		for _, e := range entries {
			if e.IsDir() {
				specs = append(specs, e.Name())
			}
		}
		sort.Strings(specs)
	}
	thresholds := []float64{0.95, 0.80}
	fmt.Printf("H_max for n=10: %.3f bits\n", math.Log2(10))
	for _, s := range specs {
		if err := analyzeSpec(filepath.Join(root, s), thresholds); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", s, err)
		}
	}
}
