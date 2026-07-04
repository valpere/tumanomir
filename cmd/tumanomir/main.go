// Command tumanomir measures specification precision for AI-driven
// software projects. See docs/requirements.md — this tool is specified
// in its own markup.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/valpere/tumanomir/internal"
	"github.com/valpere/tumanomir/internal/metrics"
	"github.com/valpere/tumanomir/internal/spec"
)

const version = "0.1.0-dev"

const usage = `tumanomir — specification-precision measurement for AI projects

Usage:
  tumanomir check [flags] <file.md|dir>   deterministic layer: K_drift, D_const
  tumanomir measure                       not yet implemented — v0.1 roadmap
  tumanomir version

Flags for check:
  --k-drift-max  float   gate: max fraction of untraced requirements (default 0.20)
  --d-const-min  float   warn: min lexical constraint density (default 0.35)

Default thresholds are uncalibrated hypotheses from the methodology
article; tune them on your own spec corpus.

Exit codes: 0 gates pass · 1 gate failed · 2 error.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "version":
		fmt.Println("tumanomir", version)
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func runCheck(args []string) int {
	th := internal.DefaultThresholds()
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.Float64Var(&th.KDriftMax, "k-drift-max", th.KDriftMax, "max fraction of untraced requirements")
	fs.Float64Var(&th.DConstMin, "d-const-min", th.DConstMin, "min lexical constraint density")
	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "check: exactly one <file.md|dir> argument required")
		return 2
	}

	specs, err := spec.Load(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "check:", err)
		return 2
	}

	cr := aggregate(specs, th)

	if cr.KDVerdict == internal.VerdictSkipped {
		fmt.Printf("  K_drift:  —     [n/a]%s(no [REQ-*] tags found)\n", pad(cr.KDVerdict))
	} else {
		fmt.Printf("  K_drift:  %.2f  [%s]%s(threshold %.2f, %d/%d requirements untraced)\n",
			cr.KD.Value, cr.KDVerdict, pad(cr.KDVerdict), th.KDriftMax, cr.KD.Hanging, cr.KD.Requirements)
	}
	fmt.Printf("  D_const:  %.2f  [%s]%s(threshold %.2f, %d markers / %d prose tokens)\n",
		cr.DC.Value, cr.DCVerdict, pad(cr.DCVerdict), th.DConstMin, cr.DC.ConstraintMarkers, cr.DC.ProseTokens)
	fmt.Printf("  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument; not yet implemented — v0.1 roadmap)\n")

	for _, id := range cr.KD.HangingIDs {
		fmt.Printf("    hanging: %s\n", id)
	}

	if cr.KDVerdict == internal.VerdictBlock {
		fmt.Println("\nexit code: 1 (gate failed)")
		return 1
	}
	return 0
}

// checkResult holds the deterministic layer's aggregated metric values and
// their gate verdicts, computed by aggregate.
//
// TODO(REQ-OUT-01): move to internal/report once that package exists
type checkResult struct {
	KD        internal.KDriftResult
	DC        internal.DConstResult
	KDVerdict internal.Verdict
	DCVerdict internal.Verdict
}

// aggregate combines K_drift and D_const across specs so a multi-file
// corpus is judged as one source of truth, then computes gate verdicts
// against th. Per-file hanging requirement IDs are prefixed with their
// source path for actionable output.
func aggregate(specs []spec.Spec, th internal.Thresholds) checkResult {
	var kd internal.KDriftResult
	var dc internal.DConstResult
	for _, s := range specs {
		k := metrics.KDrift(s.Content)
		kd.Requirements += k.Requirements
		kd.Hanging += k.Hanging
		for _, id := range k.HangingIDs {
			kd.HangingIDs = append(kd.HangingIDs, s.Path+": "+id)
		}
		d := metrics.DConst(s.Content)
		dc.ConstraintMarkers += d.ConstraintMarkers
		dc.ProseTokens += d.ProseTokens
	}
	if kd.Requirements > 0 {
		kd.Value = float64(kd.Hanging) / float64(kd.Requirements)
	}
	if total := dc.ConstraintMarkers + dc.ProseTokens; total > 0 {
		dc.Value = float64(dc.ConstraintMarkers) / float64(total)
	}

	kdVerdict := internal.VerdictOK
	if kd.Value > th.KDriftMax {
		kdVerdict = internal.VerdictBlock
	}
	if kd.Requirements == 0 {
		// No [REQ-*] tags at all is a distinct signal from a genuine
		// fully-traced pass (0.00 with N>0) — render it explicitly
		// rather than let it masquerade as "K_drift: 0.00 [ok]".
		kdVerdict = internal.VerdictSkipped
	}
	dcVerdict := internal.VerdictOK
	if dc.Value < th.DConstMin {
		dcVerdict = internal.VerdictWarn // lexical proxy: advisory, not a gate
	}

	return checkResult{KD: kd, DC: dc, KDVerdict: kdVerdict, DCVerdict: dcVerdict}
}

// pad aligns verdict columns for ok/warn/block widths.
func pad(v internal.Verdict) string {
	switch v {
	case internal.VerdictOK:
		return "     "
	case internal.VerdictWarn:
		return "   "
	case internal.VerdictSkipped:
		return "    "
	default:
		return "  "
	}
}
