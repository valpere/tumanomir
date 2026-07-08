# Requirements: tumanomir

Specification-precision measurement tool for AI-driven software projects.
Productization of the "Source of the Unknown" methodology (see
`docs/investigation/history.md` for provenance).

This document is written in tumanomir's own traceable markup
(`[REQ-*] -> [FUN-*]`, `@schema`) — the tool must be able to measure its
own specification (dogfooding). The bracket distinguishes a definition
from a reference: a bracketed `[REQ-*]` tag at the start of a numbered
item *defines* that requirement, and `metrics.KDrift` (`internal/metrics/
kdrift.go`) counts every such occurrence — literally, by regex, with no
positional awareness — so a bracketed ID written anywhere else in this
document (including in an explanatory sentence like this one) would
register as a second, untraced "requirement." A bare mention elsewhere in
prose (e.g. "see REQ-CFG-01" below, with no brackets) is a
cross-reference, not a definition, and is deliberately left unbracketed
for exactly that reason.

---

## 1. Data model

@schema Thresholds {
  k_drift_max: Float @constraint(default: 0.20, range: [0,1]),
  d_const_min: Float @constraint(default: 0.35, range: [0,1]),
  d_pair_max:  Float @constraint(default: 0.30, range: [0,1])
}

@schema InstrumentConfig {
  backend: Enum["ollama"],
  model: String,
  temperature: Float @constraint(default: 1.0),
  samples: Int @constraint(default: 10, min: 2),
  think: Bool @constraint(default: false),
  num_ctx: Int @constraint(rule: "must exceed prompt token count"),
  num_predict: Int @constraint(rule: "must exceed natural output length"),
  sim_threshold: Float @constraint(default: 0.95, range: [0,1], rule: "the measure command's --sim-threshold flag defaults to 0.95 (a proposed hypothesis matching the article's experiment, not a calibrated constant); dispersion.Analyze itself takes SimThreshold as a required caller-supplied parameter with no internal default"),
  prompt: String @constraint(rule: "named, versioned package-level constant, not an inline literal — instrument-relative config, must be reproducible from the report"),
  prompt_version: String @constraint(rule: "identifies which named prompt constant prompt's value came from, e.g. \"PromptV1\" — lets the report print the version without a hardcoded literal at the print site")
}

@schema Report {
  check: CheckResult,
  measure: MeasureResult?,
  verdict: Enum["ok","warn","block","skipped"],
  exit_code: Int @constraint(in: [0,1,2])
}

---

## 2. Functional requirements

### 2.1 Deterministic layer (`check` command)

1. [REQ-CHK-01] The tool must compute K_drift (fraction of requirements
   lacking a trace edge) over markdown specs using explicit markup only:
   a requirement is a `[REQ-*]` tag; a trace edge is `-> [FUN-*]`,
   `-> [LOG-*]` or `-> [PHY-*]` appearing before the next `[REQ-*]` tag.
   -> [FUN-CHK-01] metrics.KDrift(doc []byte) KDriftResult
   When a spec (or aggregated corpus) has zero `[REQ-*]` tags, `check`
   must render K_drift as an explicit "no requirements found" signal
   (verdict `skipped`), not a numeric `0.00` pass — the two are not the
   same measurement and must not be visually indistinguishable.

2. [REQ-CHK-02] K_drift output must list the identifiers of hanging
   requirements, not only the ratio — the metric must be actionable.
   -> [FUN-CHK-02] KDriftResult.HangingIDs []string

3. [REQ-CHK-03] The tool must compute D_const (lexical constraint density)
   as markers/(markers+prose tokens), where markers are the substrings
   `@schema`, `@constraint`, `[REQ-`, `-> [FUN-`, `-> [LOG-`, `-> [PHY-`.
   Markers and prose tokens are disjoint by construction: prose tokens
   are whitespace-separated tokens of the document with every matched
   marker byte-span excluded first, so a marker's bytes are never also
   counted as part of a prose token. Consequently a document consisting
   entirely of markers (no other prose) must yield D_const = 1.0 exactly
   — not a value capped below 1.0 by marker text leaking into the prose
   count (a bug fixed in `dconst.go`, see its own history for detail).
   -> [FUN-CHK-03] metrics.DConst(doc []byte) DConstResult

4. [REQ-CHK-04] `check` must accept a single file or a directory
   (recursively, `*.md` only) and aggregate per-file results. Directory
   walks must skip dot-prefixed and `_`-prefixed subdirectories (e.g.
   `.git`, `.claude`, `_sanity`) — running `check` at a project root must
   not silently pull in tooling/scratch/archival markdown that isn't the
   spec under test. An explicitly-passed single file path is never
   filtered by this rule, even if dot-prefixed — the exclusion applies
   only to directory walks.
   -> [FUN-CHK-04] spec.Load(path string) ([]Spec, error)

5. [REQ-CHK-05] The deterministic layer must not invoke any LLM or
   network call; it must be usable as a git pre-commit hook.
   -> [LOG-CHK-05] packages internal/metrics, internal/spec have no
      network imports (enforced by test)

6. [REQ-CHK-06] D_const is a lexical proxy, not a ground-truth measure of
   specification precision — it must never produce `VerdictBlock` (exit
   code 1, see REQ-OUT-02) regardless of its value relative to
   `d_const_min`. When D_const falls below threshold the tool must report
   `VerdictWarn` at most; only K_drift's verdict may block. This is
   advisory-only by design, not an oversight.
   -> [LOG-CHK-06] cmd/tumanomir/main.go `aggregate`: `dcVerdict` is
      assigned only `VerdictOK` or `VerdictWarn`, never `VerdictBlock`;
      the exit-code-1 branch in `runCheck` checks `KDVerdict` exclusively.

### 2.2 Stochastic layer (`measure` command)

7. [REQ-MSR-01] The tool must measure D_pair = 1 − mean pairwise
   structural AST similarity over N generated Go artifacts from one spec.
   -> [FUN-MSR-01] dispersion.Analyze(sources [][]byte, simThreshold float64) DispersionResult

   "Mean pairwise structural AST similarity" is defined operationally,
   not left to inference: each generated source is parsed with `go/ast`
   into a bag-of-features vector keyed by structural tokens — type and
   struct/interface declarations, field names with their type strings,
   interface methods, func declarations (with receiver type folded into
   the key) and their signatures, and top-level const/value names.
   Cosine similarity is computed between every pair of feature vectors
   and averaged over all N(N-1)/2 pairs to give the mean similarity that
   D_pair is one minus. The authoritative implementation — including the
   exact feature-key format — is `internal/dispersion/astfeat.go`; this
   requirement points to it rather than duplicating it, so the two must
   not drift.

8. [REQ-MSR-02] Cluster entropy H (Shannon, over single-linkage clusters
   at a configurable similarity threshold) and its normalized form
   H_norm = H/log2(N) must be reported as ordinal signals, never as the
   primary gate metric.
   -> [FUN-MSR-02] DispersionResult{H, HNorm, Clusters, SimThresh}

9. [REQ-MSR-03] Generation must go through a pluggable instrument
   interface; v0.1 ships one backend: Ollama chat API.
   -> [FUN-MSR-03] instrument.Generator interface; instrument.Ollama

10. [REQ-MSR-04] The full instrument configuration (backend, model,
    temperature, N, think mode, num_ctx, num_predict, sim threshold,
    prompt) must be fixed per run and printed in the report —
    measurements are instrument-relative and meaningless without it. The
    prompt is a versioned, named constant (not an inline literal) so a
    reader can reproduce the measurement from the report alone.
    -> [FUN-MSR-04] InstrumentConfig serialized into Report header

11. [REQ-MSR-05] Generations that fail Go parsing must be retried up to a
    bounded limit and the discard count must be reported as invalid rate;
    hiding invalid generations is forbidden.
    -> [FUN-MSR-05] measure loop: retry ≤ 2 per sample; DispersionResult.Discarded

    A discard rate above a documented threshold must be flagged prominently
    in the `measure` command's report output, not merely included in the
    numeric summary buried among other stats. The proposed threshold is 40%,
    matching the retry budget of ≤2 per sample — stated here as a hypothesis,
    not a calibrated constant, the same treatment given to the 0.20/0.35/0.30
    thresholds elsewhere in this document (see REQ-CFG-01). This is a warning
    signal only, not a new gate: it implies no exit-code change, consistent
    with D_pair/H_norm staying ordinal/advisory in v0.1.

12. [REQ-MSR-06] For reasoning-capable models the instrument must set
    think=false; requests must set num_ctx and num_predict explicitly.
    Silent truncation of the input spec is a measurement-integrity bug.
    Truncation of the *output* must also be detected and surfaced: the
    backend's own done_reason ("stop" vs. "length") is a direct signal,
    stronger than inferring truncation from EvalCount == NumPredict alone.
    -> [FUN-MSR-06] instrument.Ollama request builder; prompt-size check
       against num_ctx before the run; Generation.DoneReason surfaced from
       the backend response and flagged in measure's report

13. [REQ-MSR-07] D_pair's point estimate must be reported alongside a 95%
    bootstrap confidence interval, so the report is honest about the
    sampling noise visible across independent instruments at the same N.
    The bootstrap is defined operationally: resample the N AST feature
    vectors with replacement B=2000 times; for each resample, compute
    mean pairwise cosine similarity over the resampled multiset (not the
    precomputed similarity matrix — a resample can draw the same original
    sample twice, and the matrix's diagonal is never populated) and take
    1 − that mean; the CI bounds are the 2.5th and 97.5th percentiles of
    the resulting 2000 D_pair values. This is purely additive: it does not
    change D_pair's definition, and DPairVerdict still gates on the point
    estimate alone — the CI is advisory, like H/H_norm (REQ-MSR-02).
    -> [FUN-MSR-07] dispersion.Analyze populates DispersionResult{DPairCILow, DPairCIHigh}

    B, the fixed RNG seed, and the 95% CI level are compile-time
    constants, not configurable via flag or config file. If any of them
    ever becomes configurable, it must enter InstrumentConfig and the
    printed report, or it would silently break the instrument-relative
    reproducibility invariant (REQ-MSR-04) — a report could no longer be
    reproduced from its own printed configuration alone.

### 2.3 Output and gating

14. [REQ-OUT-01] Human-readable TTY output: one line per gated metric
    with value, verdict (ok/warn/block/skipped) and the threshold it was
    judged against. Ordinal signals (H, H_norm) are printed without a
    verdict/threshold column, since they never gate (REQ-MSR-02).
    -> [FUN-OUT-01] internal/report.RenderCheck(w io.Writer, r CheckResult,
       th Thresholds) error and internal/report.RenderMeasure(w io.Writer,
       r MeasureResult, th Thresholds) error, called from cmd/tumanomir's
       runCheck/runMeasureImpl; internal/report.RenderReport(w io.Writer, r
       Report, th Thresholds) error, called from runGate, over the unified
       @schema Report shape above (REQ-GATE-01). check and measure keep
       rendering their own structurally different content standalone —
       RenderReport is additive, not a replacement.

15. [REQ-OUT-02] Exit codes: 0 = all gates pass, 1 = at least one gate
    failed, 2 = execution error. CI-composable by construction.
    -> [FUN-OUT-02] Report.exit_code

16. [REQ-OUT-03] `check`, `measure`, and `gate` must accept a `--format
    text|json` flag (default `text`, current TTY behavior unchanged). In
    JSON mode, the command emits exactly one compact JSON object to
    stdout (`json.NewEncoder(os.Stdout).Encode`), nothing else — no
    additional text before or after it. Field names and nesting are
    code-authoritative via the Go structs' `json` tags (CheckResult,
    MeasureResult, Report, and the value types they embed) rather than
    re-derived here in markup — the same treatment REQ-MSR-01 already
    gives `internal/dispersion/astfeat.go`, so this requirement points to
    the code rather than duplicating its shape, and the two must not
    drift. `check`/`measure`'s JSON carries no in-band exit_code field —
    the process exit status (REQ-OUT-02) is the only signal for those two
    commands; `gate`'s JSON does carry one, via Report.exit_code
    (REQ-GATE-03). Any `--format` value other than `text`/`json` is a
    usage error (stderr, exit 2). Error paths (validation/execution
    failures) are unaffected — always plain text on stderr regardless of
    `--format`.
    -> [FUN-OUT-03] cmd/tumanomir's checkJSON/measureJSON/gateJSON
       wrapper types (pairing each command's result with the
       internal.Thresholds it was gated against) and the --format
       branches in runCheck/runMeasureImpl/runGateImpl

17. [REQ-CFG-01] Thresholds are overridable via CLI flags; defaults are
    the article's hypothesis values (0.20 / 0.35 / 0.30) and must be
    documented as uncalibrated starting points.
    -> [FUN-CFG-01] internal.DefaultThresholds(); flag wiring in cmd

    **v0.1 divergence from the article's protocol, declared per
    REQ-NFR-03 rather than left silent:** the article treats a ΔH
    calibration baseline (measuring against a zero-ambiguity reference
    spec, not an absolute value) as an obligatory step, and warns that
    quality-gate thresholds must be set at a working temperature
    (~0.3–0.5), never at the stress-test temperature (~1.0) used to
    probe the model's interpretive range — "numbers from the stress test
    are not substituted into gates." v0.1 does neither: `measure`'s
    default temperature is 1.0, and D_pair is gated as an absolute value
    against 0.30, not as a delta against a calibrated baseline measured
    at working temperature. This is a deliberate, temporary
    simplification pending the `calibrate` roadmap item, not a silent
    methodology change — anyone tuning `--d-pair-max` or `--temp` should
    read the resulting number as a stress-test-regime absolute, not the
    article's calibrated, working-temperature delta.

### 2.4 Configuration file (.tumanomir.yaml)

18. [REQ-CFG-02] `check`/`measure` (and later `gate`) must accept an
    optional `.tumanomir.yaml` config file so thresholds and instrument
    settings don't have to be repeated as CLI flags on every invocation.
    An explicit `--config <path>` is authoritative: the named file must
    exist and parse, or the command exits 2. Otherwise `./.tumanomir.yaml`
    (current working directory only, no upward directory search) is
    loaded if present and silently skipped if absent. The file's schema
    mirrors `@schema Thresholds`/`@schema InstrumentConfig` above, minus
    `prompt` (deliberately non-configurable — REQ-MSR-04's reproducibility
    invariant would be undermined by letting the prompt vary per project).
    -> [FUN-CFG-02] internal/config.Config, internal/config.Load(path string)
       (internal/config.Config, error)

19. [REQ-CFG-03] Precedence is CLI flag > config file > built-in default.
    Each subcommand's config file is resolved before its `flag.FlagSet` is
    built, and the resolved value seeds each flag's own default — so
    `flag.Parse`'s ordinary override behavior gives CLI-flag-wins for
    free, with no post-parse `fs.Visit` reconciliation needed.
    -> [FUN-CFG-03] internal/config.Config.ApplyThresholds(th
       *internal.Thresholds), internal/config.Config.InstrumentOr(def
       internal.InstrumentConfig) internal.InstrumentConfig; called from
       cmd/tumanomir's runCheck/runMeasureImpl before their fs.*Var
       registrations

### 2.5 Gate command (CI mode)

20. [REQ-GATE-01] `gate` must run the deterministic layer (K_drift,
    D_const) and, when an instrument is configured, the stochastic
    layer (D_pair, H_norm) in one process invocation over one spec
    file, producing one unified Report (@schema Report) and one exit
    code — the CI-composable entry point REQ-OUT-02 already specifies
    the exit-code contract for.
    -> [FUN-GATE-01] cmd/tumanomir's runGate; internal/report.Report,
       internal/report.RenderReport(w io.Writer, r Report, th
       internal.Thresholds) error

    `gate` takes exactly one `<file.md>` argument, never a directory —
    the same restriction `measure` already enforces (see
    runMeasureImpl's directory check), extended to `gate` uniformly
    regardless of which mode it runs in.

21. [REQ-GATE-02] `gate` must run in deterministic-only mode —
    Report.measure left null — when no instrument is resolvable from
    CLI flags or `.tumanomir.yaml`'s `instrument:` section. If any
    measure-specific CLI flag (`--samples`/`-n`, `--temp`,
    `--sim-threshold`, `--num-ctx`, `--num-predict`, `--think`,
    `--d-pair-max`) is explicitly passed while no instrument resolves,
    `gate` must fail with exit code 2 rather than silently discarding
    it — a silently-downgraded gate run is the same class of
    measurement-integrity bug REQ-MSR-06 already treats as a bug, not a
    warning.
    -> [FUN-GATE-02] cmd/tumanomir's runGate instrument-resolution and
       contradiction-check logic (fs.Visit over measure-specific flags)

22. [REQ-GATE-03] `gate`'s Report.verdict/exit_code must combine
    KDVerdict, DCVerdict, and (when the stochastic layer ran)
    DPairVerdict by worst-case precedence block > warn > skipped > ok
    over that full set. exit_code is 1 if and only if KDVerdict ==
    block or DPairVerdict == block — DCVerdict and H/H_norm (never
    Verdict-bearing) must never independently produce exit_code == 1,
    consistent with REQ-CHK-06/REQ-MSR-02. exit_code == 2 is reserved
    for execution errors that never reach a rendered Report.
    -> [FUN-GATE-03] cmd/tumanomir's gateVerdict(kd, dc internal.Verdict,
       dpair *internal.Verdict) (internal.Verdict, int)

### 2.6 Calibrate command

23. [REQ-CAL-01] `calibrate` must accept a single JSONL corpus file, one
    row per historical spec: `{"spec_path": "...", "instrument":
    "ollama:qwen3-coder:30b", "d_pair": 0.27, "outcome": 0.8}`. `spec_path`
    must point to the immutable spec version that produced the paired
    `d_pair`/`outcome` — a snapshot, not a live/mutable working file —
    since `calibrate` recomputes K_drift/D_const fresh from it (DRY: the
    spec is the single source of truth for its own deterministic metrics)
    but never recomputes `d_pair` itself (that would mean re-running an
    LLM instrument, which `calibrate` must never do — see REQ-CAL-05).
    `outcome` is a caller-defined continuous float (higher = worse
    downstream pain); this document does not prescribe how
    rework/compile-failures/iteration-count/token-budget combine into it.
    -> [FUN-CAL-01] calibrate.Row, calibrate.LoadCorpus(path string)
       (rows []Row, skipped int, err error), calibrate.AnalyzedRow,
       calibrate.BuildAnalyzedRows(rows []Row) ([]AnalyzedRow, error)

24. [REQ-CAL-02] `instrument` is a required opaque identifier for the
    `InstrumentConfig` that produced a row's `d_pair`. All rows in one
    `calibrate` run must share the same `instrument` value — mixing
    instruments would produce an authoritative-looking but
    methodologically meaningless correlation, since D_pair values
    measured under different instrument configurations aren't
    comparable, violating the instrument-relative reporting invariant
    (REQ-MSR-04). A second, distinct `instrument` value anywhere in the
    corpus is a hard abort naming both the expected and the mismatching
    value — never a per-row skip like the malformed-row cases in
    REQ-CAL-04.
    -> [FUN-CAL-02] calibrate.LoadCorpus's baseline/mismatch check (the
       first valid row's Instrument becomes the baseline; any later
       valid row naming a different Instrument returns an error
       immediately)

25. [REQ-CAL-03] For each of K_drift.Value, D_const.Value, and D_pair,
    `calibrate` must compute the Spearman rank correlation (not Pearson —
    `outcome`'s arbitrary, caller-defined scale means only a monotonic
    relationship is meaningful to test, and Spearman degrades correctly
    to the binary "clear vs. fog" case via average-rank tie handling)
    against `outcome`, printing the coefficient even when weak or near
    zero — a metric that doesn't predict outcome here is itself the
    finding. `calibrate` must also print a median-split summary: each
    metric's min/mean/max within the below-median-outcome half and the
    above-median-outcome half. `calibrate` must never auto-select a
    single threshold number and must never write to `.tumanomir.yaml` —
    thresholds stay a human decision (REQ-NFR-03).
    -> [FUN-CAL-03] calibrate.Analyze(rows []AnalyzedRow)
       CalibrationResult (Spearman via rank-transform + Pearson on the
       ranks, per calibrate.go's rank/pearson/spearman helpers; median
       split via medianSplit); cmd/tumanomir's renderCalibration prints
       the result with no threshold recommendation and no config write

26. [REQ-CAL-04] A corpus row that fails to parse, has an unreadable
    `spec_path`, or has `d_pair`/`outcome` outside `[0,1]` is skipped and
    counted — never silently dropped without a count, and never aborting
    the whole run (that treatment is reserved for REQ-CAL-02's
    instrument-mismatch case). `calibrate` must exit with code 2 only
    when zero valid rows remain after skipping. A corpus with fewer than
    5 valid rows must still produce full output, with a printed warning
    that the coefficients are not yet statistically meaningful — a
    warning, not a failure.
    -> [FUN-CAL-04] calibrate.LoadCorpus's per-row validation (skip +
       count); cmd/tumanomir's runCalibrate zero-valid-rows check (exit
       2); calibrate.MinRowsForCalibration,
       CalibrationResult.SmallSample

27. [REQ-CAL-05] `calibrate` must never invoke an LLM or make a network
    call: `d_pair` comes pre-computed from the corpus, K_drift/D_const
    recompute via the existing zero-network `internal/metrics` functions,
    and the correlation math is pure arithmetic. This is the same
    zero-network guarantee `check` already provides (REQ-CHK-05), so
    `calibrate` is safe to run in the same offline/git-hook contexts.
    -> [LOG-CAL-05] package internal/calibrate has no network imports
       (enforced by test, internal/nonetwork_test.go)

---

## 3. Non-functional requirements

28. [REQ-NFR-01] `check` on a 1 MB spec corpus must complete in under
    100 ms.
    -> [PHY-NFR-01] BenchmarkKDrift1MB, BenchmarkDConst1MB,
       BenchmarkCheck1MB in internal/metrics/benchmark_test.go. Verified
       end-to-end (both metrics run per iteration, mirroring a
       single-file `check` invocation, not inferred by summing isolated
       numbers): ~17ms on a 1MB synthetic corpus — comfortably within the
       100ms budget (K_drift ~0.25ms, D_const ~17ms individually).
       Both metrics are now allocation-flat: K_drift was rewritten
       (issue #66) from regexp.FindAllSubmatchIndex (which allocated one
       []int per match — 3260 allocs/op, ~1:1 with requirement count) to
       a hand-written byte scanner, dropping to 14 allocs/op independent
       of requirement count (a ~233x reduction) and ~9x faster in the
       same benchmark. D_const's blanking-based implementation (from
       #54) was already allocation-flat (2 allocs/op).

       "Verified" is enforced, not just measured once by hand (issue
       #67): TestCheckPerformanceBudget fails `go test ./...` if a
       1MB check exceeds 500ms (generous CI-noise margin over the
       observed ~17ms); TestKDriftAllocationBudget and
       TestDConstAllocationBudget fail if either metric's allocation
       count regresses off its allocation-flat baseline.

29. [REQ-NFR-02] Single static binary, Go ≥ 1.26, stdlib-only except
    gopkg.in/yaml.v3 — added specifically to parse .tumanomir.yaml
    (REQ-CFG-02) — no CLI framework. This is v0.1's documented trigger for
    lifting the "no YAML deps" constraint; it is not a general license for
    further dependencies without a similarly documented need.
    -> [PHY-NFR-02] go.mod with exactly one external require:
       gopkg.in/yaml.v3

30. [REQ-NFR-03] Methodology invariants must not be silently changed:
    D_pair is the working metric, H is ordinal; thresholds are
    hypotheses; instrument config is part of every result. Changes here
    require updating this document first.
    -> [LOG-NFR-03] CLAUDE.md "Methodology invariants" section

---

## 4. Out of scope for v0.1 (roadmap)

- graph-based D_const (RFLP/Neo4j), assisted K_drift (LLM parser)
- non-Ollama instruments, non-Go projections (SQL DDL, OpenAPI)

`calibrate` (REQ-CAL-01..05) and bootstrap CI for D_pair (REQ-MSR-07) both
shipped, so neither is listed here anymore — see docs/roadmap.md for
`calibrate`'s remaining open item (accumulating a real outcome-labeled
corpus, which is data collection, not tool-building).
