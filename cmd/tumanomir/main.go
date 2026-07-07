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
	"github.com/valpere/tumanomir/internal/config"
	"github.com/valpere/tumanomir/internal/dispersion"
	"github.com/valpere/tumanomir/internal/instrument"
	"github.com/valpere/tumanomir/internal/metrics"
	"github.com/valpere/tumanomir/internal/report"
	"github.com/valpere/tumanomir/internal/spec"
)

// defaultConfigPath is the cwd-only (no upward directory walk) location
// checked for an implicit config file when --config isn't given.
const defaultConfigPath = ".tumanomir.yaml"

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

Flags for check and measure:
  --config  string  path to a .tumanomir.yaml config file (default: load
                     ./.tumanomir.yaml if present, cwd only, no upward
                     search; a named --config path must exist and parse)

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

Flag precedence: CLI flag > .tumanomir.yaml > built-in default. See
.tumanomir.yaml's schema in docs/requirements.md (REQ-CFG-02/03).

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

// scanConfigFlag manually pre-scans args for an explicit --config value
// before the real FlagSet exists — needed because flag defaults must be
// resolved (from config.Load) *before* the fs.*Var calls that register
// them, and flag.Parse's own parsing happens too late for that. Supports
// both "--config value" and "--config=value" (and their single-dash
// spellings, since Go's flag package treats -x and --x identically).
// Unlike flag.Parse, it does not stop at the first non-flag token — a
// value belonging to some other, arity-unknown-to-this-scan flag (e.g.
// "--k-drift-max 0.5") must not be mistaken for the end of the flag
// section.
func scanConfigFlag(args []string) (path string, ok bool) {
	for i, a := range args {
		if a == "--" {
			break
		}
		if !strings.HasPrefix(a, "-") {
			continue
		}
		name := strings.TrimLeft(a, "-")
		if v, found := strings.CutPrefix(name, "config="); found {
			return v, true
		}
		if name == "config" && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// resolveConfig implements the --config discovery/precedence rule
// (REQ-CFG-02): an explicit --config path is authoritative and must
// exist/parse; otherwise ./.tumanomir.yaml (cwd only, no upward walk) is
// loaded if present and silently skipped if absent. On failure it prints
// an actionable, cmdName-prefixed message to stderr (matching this file's
// existing error-reporting convention) and returns ok=false, so the
// caller can return exit code 2 immediately.
func resolveConfig(args []string, cmdName string) (cfg config.Config, ok bool) {
	path, explicit := scanConfigFlag(args)
	if !explicit {
		path = defaultConfigPath
		if _, err := os.Stat(path); err != nil {
			return config.Config{}, true // no default config file: silently skip
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		// err (os.PathError or config.Load's own "parse %s: %w" wrap)
		// already names path, so it isn't repeated here.
		fmt.Fprintf(os.Stderr, "%s: config: %v\n", cmdName, err)
		return config.Config{}, false
	}
	return cfg, true
}

// runCheck implements the `check` subcommand: the deterministic layer
// (K_drift, D_const) — zero network, zero LLM. Parses flags, loads the
// spec(s) via spec.Load, delegates to aggregate for the pure metric
// computation, then prints the report. Returns the process exit code
// (0 pass, 1 gate failed, 2 error) rather than calling os.Exit directly,
// so it's directly testable (see main_test.go's TestRunCheck* tests).
func runCheck(args []string) int {
	cfg, ok := resolveConfig(args, "check")
	if !ok {
		return 2
	}

	th := internal.DefaultThresholds()
	cfg.ApplyThresholds(&th)
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	var configFlag string
	fs.StringVar(&configFlag, "config", "", "path to a .tumanomir.yaml config file")
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

	if err := report.RenderCheck(os.Stdout, cr, th); err != nil {
		fmt.Fprintln(os.Stderr, "check:", err)
		return 2
	}

	if cr.KDVerdict == internal.VerdictBlock {
		fmt.Println("\nexit code: 1 (gate failed)")
		return 1
	}
	return 0
}

// aggregate combines K_drift and D_const across specs so a multi-file
// corpus is judged as one source of truth, then computes gate verdicts
// against th. Per-file hanging requirement IDs are prefixed with their
// source path for actionable output.
func aggregate(specs []spec.Spec, th internal.Thresholds) report.CheckResult {
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

	return report.CheckResult{KD: kd, DC: dc, KDVerdict: kdVerdict, DCVerdict: dcVerdict}
}

// maxAttemptsPerSample is 1 initial attempt + 2 retries, per REQ-MSR-05.
const maxAttemptsPerSample = 3

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
	fileCfg, ok := resolveConfig(args, "measure")
	if !ok {
		return 2
	}

	th := internal.DefaultThresholds()
	fileCfg.ApplyThresholds(&th)

	// Seed the instrument flag defaults from the config file, layered over
	// v0.1's built-in defaults — mirrors runCheck's th/ApplyThresholds
	// seeding, but per-field rather than via a single struct pointer since
	// these flags aren't backed by one InstrumentConfig variable.
	seeded := fileCfg.InstrumentOr(internal.InstrumentConfig{Temperature: 1.0, Samples: 10, SimThreshold: 0.95})
	// The combined "backend:model" default is composed only when the
	// config actually set one of the two — leaving it "" (not ":")
	// otherwise, so the no-config-file behavior (and its "--instrument is
	// required" error) is unchanged.
	instrumentDefault := ""
	if seeded.Backend != "" || seeded.Model != "" {
		instrumentDefault = seeded.Backend + ":" + seeded.Model
	}

	fs := flag.NewFlagSet("measure", flag.ExitOnError)

	var (
		configFlag     string
		instrumentFlag string
		samples        int
		temp           float64
		simThreshold   float64
		numCtx         int
		numPredict     int
		think          bool
	)
	fs.StringVar(&configFlag, "config", "", "path to a .tumanomir.yaml config file")
	fs.StringVar(&instrumentFlag, "instrument", instrumentDefault, "required, format backend:model (e.g. ollama:qwen3-coder:30b)")
	fs.IntVar(&samples, "n", seeded.Samples, "number of generations to sample, must be >=2")
	fs.IntVar(&samples, "samples", seeded.Samples, "alias for -n")
	fs.Float64Var(&temp, "temp", seeded.Temperature, "sampling temperature")
	fs.Float64Var(&simThreshold, "sim-threshold", seeded.SimThreshold, "single-linkage clustering threshold, in [0,1]")
	fs.IntVar(&numCtx, "num-ctx", seeded.NumCtx, "required: context window; must exceed the prompt token count")
	fs.IntVar(&numPredict, "num-predict", seeded.NumPredict, "required: max generated tokens; must exceed natural output length")
	fs.BoolVar(&think, "think", seeded.Think, "enable reasoning-model think mode")
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

	if err := report.RenderMeasure(os.Stdout, mr, th); err != nil {
		fmt.Fprintln(os.Stderr, "measure:", err)
		return 2
	}

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
// report.MeasureResult, even when valid samples < 2 (handled via
// DPairVerdict == internal.VerdictSkipped, not as an error).
func runMeasureWithGenerator(gen instrument.Generator, cfg internal.InstrumentConfig, specContent []byte, samples int, th internal.Thresholds) (report.MeasureResult, error) {
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
				return report.MeasureResult{}, fmt.Errorf("generation failed: %w", err)
			}
			// The byte/3 preflight estimate under-counts non-ASCII (e.g.
			// Cyrillic) prompts; cross-check against the backend's actual
			// count post-hoc, since a stdlib-only pre-flight fix isn't
			// possible without a real tokenizer (issue #57).
			if promptEstimate > 0 && float64(g.PromptEvalCount) > float64(promptEstimate)*internal.PromptEstimateDivergenceFactor {
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

	return report.MeasureResult{
		Dispersion:           result,
		Config:               cfg,
		DPairVerdict:         dPairVerdict,
		DiscardRate:          discardRate,
		DiscardWarn:          discardRate > internal.DiscardWarnThreshold,
		Truncated:            truncated,
		PromptUnderestimated: underestimated,
	}, nil
}
