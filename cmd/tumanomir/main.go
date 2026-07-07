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
  tumanomir gate [flags] <file.md>        CI mode: check + measure (if an
                                           instrument resolves) in one pass,
                                           one unified exit code
  tumanomir version

Flags for check, measure, and gate:
  --config  string  path to a .tumanomir.yaml config file (default: load
                     ./.tumanomir.yaml if present, cwd only, no upward
                     search; a named --config path must exist and parse)

Flags for check (and gate):
  --k-drift-max  float   gate: max fraction of untraced requirements (default 0.20)
  --d-const-min  float   warn: min lexical constraint density (default 0.35)

Flags for measure (and gate, once an instrument resolves):
  --instrument     string  format backend:model (e.g. ollama:qwen3-coder:30b);
                            required for measure, optional for gate — an
                            unresolved instrument runs gate deterministic-only
  -n, --samples    int     number of generations to sample, must be >=2 (default 10)
  --temp           float   sampling temperature (default 1.0)
  --sim-threshold  float   single-linkage clustering threshold, in [0,1] (default 0.95)
  --num-ctx        int     required: context window; must exceed the prompt token count
  --num-predict    int     required: max generated tokens; must exceed natural output length
  --think          bool    enable reasoning-model think mode (default false)
  --d-pair-max     float   gate: max 1-minus-mean-pairwise-AST-similarity (default 0.30)

gate fails with exit code 2 if any measure-specific flag above is passed
explicitly while no instrument resolves (CLI flags or .tumanomir.yaml's
instrument: section) — a silently-downgraded gate run is a
measurement-integrity bug, not a convenience (REQ-GATE-02).

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
	case "gate":
		return runGate(args[1:])
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

// validateMeasureFlags validates the measure-specific flag set shared by
// `measure` (where an instrument is always required) and `gate` (where
// validateMeasureFlags is only called once gate has already established
// that an instrument resolved — see runGateImpl) — extracted so the two
// callers can't drift apart on --instrument's format rules or the
// samples/sim-threshold/num-ctx/num-predict constraints (issue #87).
// Prints an actionable cmdName-prefixed stderr message and returns
// ok=false on the first violation, mirroring resolveConfig's own
// error-reporting convention.
func validateMeasureFlags(cmdName, instrumentFlag string, samples int, simThreshold float64, numCtx, numPredict int) (backend, model string, ok bool) {
	if instrumentFlag == "" {
		fmt.Fprintf(os.Stderr, "%s: --instrument is required, format backend:model (e.g. ollama:qwen3-coder:30b)\n", cmdName)
		return "", "", false
	}
	// Split on the first ':' only — model names may themselves contain ':'
	// (e.g. qwen3-coder:30b).
	colon := strings.Index(instrumentFlag, ":")
	if colon < 0 {
		fmt.Fprintf(os.Stderr, "%s: --instrument must be in backend:model format (e.g. ollama:qwen3-coder:30b)\n", cmdName)
		return "", "", false
	}
	backend, model = instrumentFlag[:colon], instrumentFlag[colon+1:]
	if backend == "" || model == "" {
		fmt.Fprintf(os.Stderr, "%s: --instrument backend and model must both be non-empty (format backend:model)\n", cmdName)
		return "", "", false
	}
	if backend != "ollama" {
		fmt.Fprintf(os.Stderr, "%s: unsupported backend %q; v0.1 supports only \"ollama\"\n", cmdName, backend)
		return "", "", false
	}

	if samples < 2 {
		fmt.Fprintf(os.Stderr, "%s: --samples (-n) must be >= 2 to compute pairwise similarity\n", cmdName)
		return "", "", false
	}
	if simThreshold < 0 || simThreshold > 1 {
		fmt.Fprintf(os.Stderr, "%s: --sim-threshold must be within [0,1]\n", cmdName)
		return "", "", false
	}
	if numCtx <= 0 {
		fmt.Fprintf(os.Stderr, "%s: --num-ctx is required (must exceed the prompt token count)\n", cmdName)
		return "", "", false
	}
	if numPredict <= 0 {
		fmt.Fprintf(os.Stderr, "%s: --num-predict is required (must exceed the natural output length)\n", cmdName)
		return "", "", false
	}
	return backend, model, true
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
	// config set BOTH parts — a config with only one of the two leaves
	// this "" so the existing "--instrument is required" error still
	// fires, rather than producing a malformed default like "ollama:" or
	// ":my-model" that would bypass that clear error for a confusing
	// downstream backend:model-format parse failure instead (fix-review,
	// glm-5.1:cloud + deepseek-v4-flash:cloud, independently).
	instrumentDefault := ""
	if seeded.Backend != "" && seeded.Model != "" {
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

	backend, model, ok := validateMeasureFlags("measure", instrumentFlag, samples, simThreshold, numCtx, numPredict)
	if !ok {
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

// gateMeasureFlagNames lists the measure-specific CLI flags whose explicit
// presence on a `gate` invocation, combined with no instrument resolving
// (neither --instrument nor .tumanomir.yaml's instrument: section), signals
// a silently-downgraded gate run rather than an intentional
// deterministic-only invocation (REQ-GATE-02) — the same class of
// measurement-integrity bug REQ-MSR-06 already treats as a bug, not a
// warning. Scoped to CLI flags only (checked via fs.Visit, which reports
// only flags actually set on this invocation, not their defaults): a
// half-filled instrument: config section is tolerated silently, since a
// config file is persistent ambient state, not an explicit signal on this
// particular run the way a CLI flag is.
var gateMeasureFlagNames = map[string]bool{
	"n": true, "samples": true, "temp": true, "sim-threshold": true,
	"num-ctx": true, "num-predict": true, "think": true, "d-pair-max": true,
}

// runGate parses flags, validates the positional spec-file argument,
// constructs the real Ollama generator (used only if an instrument
// resolves) and delegates to runGateImpl, mirroring runMeasure's own
// split (issue #70's DI pattern).
func runGate(args []string) int {
	return runGateImpl(args, func(cfg internal.InstrumentConfig) instrument.Generator {
		return instrument.NewOllama(cfg)
	})
}

// runGateImpl is runGate's testable core (REQ-GATE-01/02/03): it runs the
// deterministic layer unconditionally, then — only if an instrument
// resolves from CLI flags or .tumanomir.yaml's instrument: section — the
// stochastic layer too, combining both into one report.Report/exit code.
// Mirrors runMeasureImpl's newGen dependency-injection split so tests can
// drive the full path through a fake Generator without touching the
// network.
func runGateImpl(args []string, newGen func(internal.InstrumentConfig) instrument.Generator) int {
	fileCfg, ok := resolveConfig(args, "gate")
	if !ok {
		return 2
	}

	th := internal.DefaultThresholds()
	fileCfg.ApplyThresholds(&th)

	// Same instrument-default seeding as runMeasureImpl (see its own
	// comment): a config with only one of backend/model leaves
	// instrumentDefault "", so gate correctly treats it as unresolved
	// rather than composing a malformed "ollama:" or ":my-model" default.
	seeded := fileCfg.InstrumentOr(internal.InstrumentConfig{Temperature: 1.0, Samples: 10, SimThreshold: 0.95})
	instrumentDefault := ""
	if seeded.Backend != "" && seeded.Model != "" {
		instrumentDefault = seeded.Backend + ":" + seeded.Model
	}

	fs := flag.NewFlagSet("gate", flag.ExitOnError)

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
	fs.Float64Var(&th.KDriftMax, "k-drift-max", th.KDriftMax, "max fraction of untraced requirements")
	fs.Float64Var(&th.DConstMin, "d-const-min", th.DConstMin, "min lexical constraint density")
	fs.StringVar(&instrumentFlag, "instrument", instrumentDefault, "format backend:model (e.g. ollama:qwen3-coder:30b); omit to run deterministic-only")
	fs.IntVar(&samples, "n", seeded.Samples, "number of generations to sample, must be >=2")
	fs.IntVar(&samples, "samples", seeded.Samples, "alias for -n")
	fs.Float64Var(&temp, "temp", seeded.Temperature, "sampling temperature")
	fs.Float64Var(&simThreshold, "sim-threshold", seeded.SimThreshold, "single-linkage clustering threshold, in [0,1]")
	fs.IntVar(&numCtx, "num-ctx", seeded.NumCtx, "context window; must exceed the prompt token count")
	fs.IntVar(&numPredict, "num-predict", seeded.NumPredict, "max generated tokens; must exceed natural output length")
	fs.BoolVar(&think, "think", seeded.Think, "enable reasoning-model think mode")
	fs.Float64Var(&th.DPairMax, "d-pair-max", th.DPairMax, "gate: max 1-minus-mean-pairwise-AST-similarity (hypothesis, not calibrated)")
	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "gate: exactly one <file.md> argument required")
		return 2
	}
	path := fs.Arg(0)
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		return 2
	}
	if info.IsDir() {
		fmt.Fprintf(os.Stderr, "gate: %s is a directory; gate takes a single spec file (directory aggregation is not methodologically meaningful for dispersion measurement in v0.1)\n", path)
		return 2
	}

	if instrumentFlag == "" {
		var contradicting string
		fs.Visit(func(f *flag.Flag) {
			if contradicting == "" && gateMeasureFlagNames[f.Name] {
				contradicting = f.Name
			}
		})
		if contradicting != "" {
			fmt.Fprintf(os.Stderr, "gate: --%s was passed but no instrument resolved (no --instrument and no .tumanomir.yaml instrument: section) — refusing to silently downgrade to deterministic-only (REQ-GATE-02)\n", contradicting)
			return 2
		}
	}

	specs, err := spec.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		return 2
	}
	cr := aggregate(specs, th)

	var mrPtr *report.MeasureResult
	if instrumentFlag != "" {
		backend, model, ok := validateMeasureFlags("gate", instrumentFlag, samples, simThreshold, numCtx, numPredict)
		if !ok {
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

		mr, err := runMeasureWithGenerator(gen, cfg, specs[0].Content, samples, th)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gate:", err)
			return 2
		}
		mrPtr = &mr
	}

	var dpair *internal.Verdict
	if mrPtr != nil {
		dpair = &mrPtr.DPairVerdict
	}
	verdict, exitCode := gateVerdict(cr.KDVerdict, cr.DCVerdict, dpair)

	rep := report.Report{Check: cr, Measure: mrPtr, Verdict: verdict, ExitCode: exitCode}
	if err := report.RenderReport(os.Stdout, rep, th); err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		return 2
	}

	return exitCode
}

// gateVerdict combines the deterministic layer's K_drift/D_const verdicts
// and (when the stochastic layer ran) D_pair's verdict into gate's headline
// Verdict and exit code (REQ-GATE-03).
//
// exit_code is 1 iff K_drift or D_pair blocked — D_const/H/H_norm never
// independently gate (REQ-CHK-06/REQ-MSR-02). The headline Verdict is the
// worst-case precedence block > warn > skipped > ok over the FULL set {kd,
// dc, dpair-if-present} — deliberately checked separately from exit_code so
// an (impossible, but not type-guarded) dc == Block still surfaces as
// headline Block (fail-loud) rather than silently downgrading to "ok",
// while exit_code stays governed solely by kd/dpair.
func gateVerdict(kd, dc internal.Verdict, dpair *internal.Verdict) (internal.Verdict, int) {
	exitCode := 0
	if kd == internal.VerdictBlock || (dpair != nil && *dpair == internal.VerdictBlock) {
		exitCode = 1
	}

	all := []internal.Verdict{kd, dc}
	if dpair != nil {
		all = append(all, *dpair)
	}
	for _, v := range all {
		if v == internal.VerdictBlock {
			return internal.VerdictBlock, exitCode
		}
	}
	for _, v := range all {
		if v == internal.VerdictWarn {
			return internal.VerdictWarn, exitCode
		}
	}
	for _, v := range all {
		if v == internal.VerdictSkipped {
			return internal.VerdictSkipped, exitCode
		}
	}
	return internal.VerdictOK, exitCode
}
