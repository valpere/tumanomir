// Command tumanomir measures specification precision for AI-driven
// software projects. See docs/requirements.md — this tool is specified
// in its own markup.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/valpere/tumanomir/internal"
	"github.com/valpere/tumanomir/internal/dispersion"
	"github.com/valpere/tumanomir/internal/instrument"
	"github.com/valpere/tumanomir/internal/metrics"
	"github.com/valpere/tumanomir/internal/spec"
)

// version is printed by the `version` subcommand and `-h`/`--help`/`help`.
// Bump alongside any user-visible CLI or behavior change.
const version = "0.1.0-dev"

// usage is both the `-h`/`--help`/`help` output and the no-arguments error
// message — a single source of truth for the CLI's documented surface so
// the two presentations (help vs. error) can never drift apart.
const usage = `tumanomir — specification-precision measurement for AI projects

Usage:
  tumanomir check [flags] <file.md|dir>   deterministic layer: K_drift, D_const
  tumanomir measure [flags] <file.md>     stochastic layer: D_pair, H_norm
  tumanomir version

Flags for check:
  --k-drift-max  float   gate: max fraction of untraced requirements (default 0.20)
  --d-const-min  float   warn: min lexical constraint density (default 0.35)

Flags for measure:
  --instrument     string  required, format backend:model (e.g. ollama:qwen3-coder:30b)
  -n, --samples    int     number of generations to sample, must be >=2 (default 10)
  --temp           float   sampling temperature (default 1.0)
  --sim-threshold  float   single-linkage clustering threshold, in [0,1] (default 0.95)
  --num-ctx        int     required: context window; must exceed the prompt token count
  --num-predict    int     required: max generated tokens; must exceed natural output length
  --think          bool    enable reasoning-model think mode (default false)
  --d-pair-max     float   gate: max 1-minus-mean-pairwise-AST-similarity (default 0.30)

Default thresholds are uncalibrated hypotheses from the methodology
article; tune them on your own spec corpus.

Exit codes: 0 gates pass · 1 gate failed · 2 error.
`

func main() {
	os.Exit(dispatch(os.Args[1:]))
}

// dispatch routes a top-level command (check/measure/version/help/unknown)
// to its handler and returns the process exit code, mirroring the
// runCheck/runMeasure separation-from-main pattern so the routing itself is
// testable without a subprocess (issue #74).
func dispatch(args []string) int {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	switch args[0] {
	case "check":
		return runCheck(args[1:])
	case "measure":
		return runMeasure(args[1:])
	case "version":
		fmt.Println("tumanomir", version)
		return 0
	case "-h", "--help", "help":
		fmt.Print(usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// runCheck implements the `check` subcommand: the deterministic layer
// (K_drift, D_const) — zero network, zero LLM. Parses flags, loads the
// spec(s) via spec.Load, delegates to aggregate for the pure metric
// computation, then prints the report. Returns the process exit code
// (0 pass, 1 gate failed, 2 error) rather than calling os.Exit directly,
// so it's directly testable (see main_test.go's TestRunCheck* tests).
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
	fmt.Printf("  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)\n")

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
	KD        internal.KDriftResult // deterministic traceability metric, aggregated across all specs
	DC        internal.DConstResult // deterministic lexical constraint-density metric, aggregated
	KDVerdict internal.Verdict      // gates the exit code (VerdictBlock -> exit 1)
	DCVerdict internal.Verdict      // advisory only (VerdictWarn), never gates the exit code
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

// discardWarnThreshold is REQ-MSR-05's hypothesis discard-rate threshold
// above which the measure report must flag the run as potentially
// unreliable. Stated here as a hypothesis, not a calibrated constant, the
// same treatment given to the 0.20/0.35/0.30 thresholds in
// internal.DefaultThresholds.
const discardWarnThreshold = 0.40

// maxAttemptsPerSample is 1 initial attempt + 2 retries, per REQ-MSR-05.
const maxAttemptsPerSample = 3

// promptEstimateDivergenceFactor flags a generation whose actual
// PromptEvalCount exceeds the pre-flight byte/3 estimate by more than
// this multiple — a signal the estimate under-counted (e.g. non-ASCII
// input), not a calibrated constant. See issue #57.
const promptEstimateDivergenceFactor = 1.5

// measureResult holds the stochastic layer's aggregated metric values,
// discard-rate warning state, and gate verdict.
//
// TODO(REQ-OUT-01): move to internal/report once that package exists
type measureResult struct {
	// Dispersion is the raw D_pair/H/H_norm computation from
	// dispersion.Analyze over the run's surviving valid samples.
	Dispersion internal.DispersionResult
	// Config is the instrument configuration this run measured under —
	// printed verbatim in the report per REQ-MSR-04's instrument-relative
	// reporting requirement.
	Config internal.InstrumentConfig
	// DPairVerdict gates the exit code (VerdictBlock -> exit 1); may also
	// be VerdictSkipped if too many discards left fewer than 2 valid
	// samples to compare.
	DPairVerdict internal.Verdict
	DiscardRate  float64 // Discarded / (Discarded + N), 0 if no attempts made
	DiscardWarn  bool    // DiscardRate > 0.40 (REQ-MSR-05's hypothesis threshold)
	// Truncated is the count of accepted (valid) generations with
	// DoneReason == instrument.DoneReasonLength (REQ-MSR-06). It lives
	// here rather than on internal.DispersionResult because it's an
	// instrument/generation-loop concept (which backend, why a
	// generation stopped), not something dispersion.Analyze's pure
	// AST-similarity computation has any business knowing about.
	Truncated int
	// PromptUnderestimated is the count of generations (valid or not)
	// whose actual PromptEvalCount exceeded the pre-flight byte/3
	// estimate by more than promptEstimateDivergenceFactor — the
	// heuristic under-counts non-ASCII prompts, so this is a diagnostic
	// signal that the preflight's "errs toward refusing" guarantee may
	// not have held for this run (issue #57).
	PromptUnderestimated int
}

// runMeasure parses flags, validates the positional spec-file argument,
// constructs the real Ollama generator and delegates to
// runMeasureWithGenerator for the testable retry/discard/analyze logic,
// then prints the report.
func runMeasure(args []string) int {
	return runMeasureImpl(args, func(cfg internal.InstrumentConfig) instrument.Generator {
		return instrument.NewOllama(cfg)
	})
}

// runMeasureImpl is runMeasure's testable core: it takes the generator
// constructor as a parameter (mirroring runMeasureWithGenerator's own
// dependency-injection style) so tests can drive flag parsing, config
// construction, and the exit-code branch through a fake Generator without
// touching the network (issue #70).
func runMeasureImpl(args []string, newGen func(internal.InstrumentConfig) instrument.Generator) int {
	th := internal.DefaultThresholds()
	fs := flag.NewFlagSet("measure", flag.ExitOnError)

	var (
		instrumentFlag string
		samples        int
		temp           float64
		simThreshold   float64
		numCtx         int
		numPredict     int
		think          bool
	)
	fs.StringVar(&instrumentFlag, "instrument", "", "required, format backend:model (e.g. ollama:qwen3-coder:30b)")
	fs.IntVar(&samples, "n", 10, "number of generations to sample, must be >=2")
	fs.IntVar(&samples, "samples", 10, "alias for -n")
	fs.Float64Var(&temp, "temp", 1.0, "sampling temperature")
	fs.Float64Var(&simThreshold, "sim-threshold", 0.95, "single-linkage clustering threshold, in [0,1]")
	fs.IntVar(&numCtx, "num-ctx", 0, "required: context window; must exceed the prompt token count")
	fs.IntVar(&numPredict, "num-predict", 0, "required: max generated tokens; must exceed natural output length")
	fs.BoolVar(&think, "think", false, "enable reasoning-model think mode")
	fs.Float64Var(&th.DPairMax, "d-pair-max", th.DPairMax, "gate: max 1-minus-mean-pairwise-AST-similarity (hypothesis, not calibrated)")
	_ = fs.Parse(args)

	if instrumentFlag == "" {
		fmt.Fprintln(os.Stderr, "measure: --instrument is required, format backend:model (e.g. ollama:qwen3-coder:30b)")
		return 2
	}
	// Split on the first ':' only — model names may themselves contain ':'
	// (e.g. qwen3-coder:30b).
	colon := strings.Index(instrumentFlag, ":")
	if colon < 0 {
		fmt.Fprintln(os.Stderr, "measure: --instrument must be in backend:model format (e.g. ollama:qwen3-coder:30b)")
		return 2
	}
	backend, model := instrumentFlag[:colon], instrumentFlag[colon+1:]
	if backend == "" || model == "" {
		fmt.Fprintln(os.Stderr, "measure: --instrument backend and model must both be non-empty (format backend:model)")
		return 2
	}
	if backend != "ollama" {
		fmt.Fprintf(os.Stderr, "measure: unsupported backend %q; v0.1 supports only \"ollama\"\n", backend)
		return 2
	}

	if samples < 2 {
		fmt.Fprintln(os.Stderr, "measure: --samples (-n) must be >= 2 to compute pairwise similarity")
		return 2
	}
	if simThreshold < 0 || simThreshold > 1 {
		fmt.Fprintln(os.Stderr, "measure: --sim-threshold must be within [0,1]")
		return 2
	}
	if numCtx <= 0 {
		fmt.Fprintln(os.Stderr, "measure: --num-ctx is required (must exceed the prompt token count)")
		return 2
	}
	if numPredict <= 0 {
		fmt.Fprintln(os.Stderr, "measure: --num-predict is required (must exceed the natural output length)")
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "measure: exactly one <file.md> argument required")
		return 2
	}
	path := fs.Arg(0)
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "measure:", err)
		return 2
	}
	if info.IsDir() {
		fmt.Fprintf(os.Stderr, "measure: %s is a directory; measure takes a single spec file (directory aggregation is not methodologically meaningful for dispersion measurement in v0.1)\n", path)
		return 2
	}
	specContent, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "measure:", err)
		return 2
	}

	cfg := internal.InstrumentConfig{
		Backend:       backend,
		Model:         model,
		Temperature:   temp,
		Samples:       samples,
		Think:         think,
		NumCtx:        numCtx,
		NumPredict:    numPredict,
		SimThreshold:  simThreshold,
		Prompt:        instrument.PromptV1,
		PromptVersion: instrument.PromptVersion,
	}

	// v0.1 ships only "ollama"; already validated above. Future backends
	// would switch on cfg.Backend here.
	gen := newGen(cfg)

	mr, err := runMeasureWithGenerator(gen, cfg, specContent, samples, th)
	if err != nil {
		fmt.Fprintln(os.Stderr, "measure:", err)
		return 2
	}

	printMeasureResult(mr, th)

	if mr.DPairVerdict == internal.VerdictBlock {
		fmt.Println("\nexit code: 1 (gate failed)")
		return 1
	}
	return 0
}

// runMeasureWithGenerator runs the retry/discard generation loop against
// gen, then hands the surviving valid sources to dispersion.Analyze and
// computes the D_pair gate verdict. It is the pure-logic, testable
// counterpart to runMeasure (which does flag parsing, file I/O and
// printing), mirroring the aggregate/runCheck split.
//
// Error-signaling contract: a non-nil error means gen.Generate itself
// failed (network/HTTP/preflight failure) — a hard failure of the whole
// run, never a per-sample retry case, since a broken instrument would
// otherwise produce a misleading discard rate that looks like model
// non-determinism. The caller (runMeasure) is expected to print the error
// and exit 2. A nil error always comes with a fully populated
// measureResult, even when valid samples < 2 (handled via DPairVerdict ==
// internal.VerdictSkipped, not as an error).
func runMeasureWithGenerator(gen instrument.Generator, cfg internal.InstrumentConfig, specContent []byte, samples int, th internal.Thresholds) (measureResult, error) {
	prompt := instrument.BuildPrompt(specContent)

	promptEstimate := instrument.EstimatePromptTokens(prompt)

	var sources [][]byte
	discarded := 0
	truncated := 0
	underestimated := 0
	for slot := 0; slot < samples; slot++ {
		valid := false
		for attempt := 0; attempt < maxAttemptsPerSample; attempt++ {
			g, err := gen.Generate(context.Background(), prompt)
			if err != nil {
				return measureResult{}, fmt.Errorf("generation failed: %w", err)
			}
			// The byte/3 preflight estimate under-counts non-ASCII (e.g.
			// Cyrillic) prompts; cross-check against the backend's actual
			// count post-hoc, since a stdlib-only pre-flight fix isn't
			// possible without a real tokenizer (issue #57).
			if promptEstimate > 0 && float64(g.PromptEvalCount) > float64(promptEstimate)*promptEstimateDivergenceFactor {
				underestimated++
			}
			block, ok := instrument.ExtractGoBlock(g.Text)
			if ok && dispersion.ValidGo(block) {
				sources = append(sources, block)
				valid = true
				// A truncated generation can still parse as valid Go (e.g. a
				// complete top-level decl before the cut) and pass ValidGo —
				// that's exactly the measurement-integrity gap REQ-MSR-06's
				// done_reason signal closes: accept the sample (it IS valid
				// Go), but flag it so the report can warn that its AST may
				// not reflect the model's full intended output.
				if g.DoneReason == instrument.DoneReasonLength {
					truncated++
				}
				break
			}
			// invalid: retry while attempts remain for this slot
		}
		if !valid {
			// 3rd attempt still invalid: discard this slot and move on —
			// never pad the sample count back up with extra slots.
			discarded++
		}
	}

	result := dispersion.Analyze(sources, cfg.SimThreshold)
	result.Discarded = discarded

	dPairVerdict := internal.VerdictOK
	switch {
	case result.N < 2:
		// Too many discards left fewer than 2 valid samples to compare —
		// a distinct skipped state, never a misleading "D_pair: 0.00".
		dPairVerdict = internal.VerdictSkipped
	case result.DPair > th.DPairMax:
		dPairVerdict = internal.VerdictBlock
	}

	total := discarded + result.N
	discardRate := 0.0
	if total > 0 {
		discardRate = float64(discarded) / float64(total)
	}

	return measureResult{
		Dispersion:           result,
		Config:               cfg,
		DPairVerdict:         dPairVerdict,
		DiscardRate:          discardRate,
		DiscardWarn:          discardRate > discardWarnThreshold,
		Truncated:            truncated,
		PromptUnderestimated: underestimated,
	}, nil
}

// printMeasureResult renders REQ-MSR-04's instrument config, the
// discard-rate and truncation warnings (if triggered), and the
// D_pair/H/H_norm lines. H and H_norm are always printed as
// ordinal/advisory signals — they never gate, per the methodological
// invariant in CLAUDE.md.
//
// The discard-rate warning (REQ-MSR-05) and the truncation warning
// (REQ-MSR-06) are printed as two separate lines rather than folded
// together: they flag two distinct failure modes — generations that never
// became valid Go at all (discarded, excluded from N) vs. generations that
// parsed as valid Go but were cut off by num_predict (accepted into N, but
// their AST may not reflect the model's full intended output). Merging the
// two would blur which failure mode a reader needs to act on.
func printMeasureResult(mr measureResult, th internal.Thresholds) {
	cfg := mr.Config

	if mr.DiscardWarn {
		fmt.Printf("⚠ discard rate: %.0f%% (%d/%d generations invalid) — exceeds the %.0f%% hypothesis threshold (REQ-MSR-05); results may be unreliable\n\n",
			mr.DiscardRate*100, mr.Dispersion.Discarded, mr.Dispersion.Discarded+mr.Dispersion.N, discardWarnThreshold*100)
	}

	if mr.Truncated > 0 {
		fmt.Printf("⚠ %d/%d accepted generations had done_reason=length (truncated by num_predict) — their AST may not reflect the model's full intended output; consider raising --num-predict\n\n",
			mr.Truncated, mr.Dispersion.N)
	}

	if mr.PromptUnderestimated > 0 {
		fmt.Printf("⚠ %d generation(s) had an actual prompt-token count over %.1fx the preflight estimate — the byte/3 heuristic under-counts non-ASCII (e.g. Cyrillic) prompts and may not have caught a real truncation risk; verify --num-ctx has enough headroom\n\n",
			mr.PromptUnderestimated, promptEstimateDivergenceFactor)
	}

	fmt.Println("Instrument config (REQ-MSR-04):")
	fmt.Printf("  backend:        %s\n", cfg.Backend)
	fmt.Printf("  model:          %s\n", cfg.Model)
	fmt.Printf("  temperature:    %.2f\n", cfg.Temperature)
	fmt.Printf("  samples (N):    %d\n", cfg.Samples)
	fmt.Printf("  think:          %t\n", cfg.Think)
	fmt.Printf("  num_ctx:        %d\n", cfg.NumCtx)
	fmt.Printf("  num_predict:    %d\n", cfg.NumPredict)
	fmt.Printf("  sim_threshold:  %.2f\n", cfg.SimThreshold)
	fmt.Printf("  prompt:         %s (%d bytes)\n\n", cfg.PromptVersion, len(cfg.Prompt))

	if mr.DPairVerdict == internal.VerdictSkipped {
		fmt.Printf("  D_pair:   —     [%s]%s(only %d valid sample(s); need >=2 to compute pairwise similarity)\n",
			internal.VerdictSkipped, pad(internal.VerdictSkipped), mr.Dispersion.N)
		fmt.Printf("  H:        —     [%s]%s(ordinal signal only, not gated)\n", internal.VerdictSkipped, pad(internal.VerdictSkipped))
		fmt.Printf("  H_norm:   —     [%s]%s(ordinal signal only, not gated)\n", internal.VerdictSkipped, pad(internal.VerdictSkipped))
		return
	}

	fmt.Printf("  D_pair:   %.2f  [%s]%s(threshold %.2f, mean sim %.2f, N=%d valid, %d discarded)\n",
		mr.Dispersion.DPair, mr.DPairVerdict, pad(mr.DPairVerdict), th.DPairMax, mr.Dispersion.MeanSim, mr.Dispersion.N, mr.Dispersion.Discarded)
	fmt.Printf("  H:        %.2f  bits (ordinal signal only, not gated)\n", mr.Dispersion.H)
	fmt.Printf("  H_norm:   %.2f  (ordinal signal only, not gated)\n", mr.Dispersion.HNorm)
}
