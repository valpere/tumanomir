# tumanomir — architecture

> English translation. Ukrainian original: [`architecture.md`](architecture.md)
> — kept as the source of truth; this file is translated for accessibility
> (see issue #21) and should stay in sync with it.
>
> This file used to live as `docs/investigation/design.md` — under the
> "investigation" directory, which is meant for methodology provenance
> (`docs/investigation/history.md`, external reviews), not the tool's live
> architecture. Moved alongside `docs/requirements.md`: requirements are
> the what, architecture is the how, `investigation/` is the why and how
> it was validated.

Specification-precision measurement tool for AI-driven projects.
Productization of the methodology from the article "Source of the Unknown"
(`docs/investigation/SourceOfTheUnknown.md`).

The roadmap (what's not built yet, and in what order) lives separately in
[`roadmap.md`](roadmap.md). Tactical debt and small tasks live in
[GitHub issues](https://github.com/valpere/tumanomir/issues), not here.

## Metrics

| Metric | Layer | What it measures | Instrument |
| --- | --- | --- | --- |
| `K_drift` | deterministic | requirements without a `[REQ-*] -> [FUN/LOG/PHY-*]` trace | markup linter, no LLM |
| `D_const` | deterministic | lexical density of constraints (markers vs. prose) | scanner, no LLM |
| `D_pair` | stochastic | 1 − mean pairwise AST similarity of N generations | LLM via Ollama |
| `H_norm` | stochastic | cluster entropy / log₂N — ordinal signal | same |

Methodological invariants (from the article; don't roll back without
updating `docs/requirements.md` first):
- D_pair is the working metric and the only stochastic-layer gate; H_norm
  (= H / log₂N) is ordinal ("one cluster or many"), reported but never
  gated; raw H (bits) is also printed in the report but saturates at
  log₂N for small N. Alongside the point estimate, `measure`/`gate` print a
  95% bootstrap confidence interval for D_pair (2000 resamples of the AST
  feature vectors, N>=2, fixed seed — REQ-MSR-07) — also advisory; the gate
  still compares the point estimate.
- Metrics are instrument-relative: the full configuration (backend, model,
  temperature, N, think, num_ctx, num_predict, sim_threshold, prompt) is
  fixed and printed in every `measure` report (REQ-MSR-04).
- Invalid rate is reported, never hidden (retry ≤2 per sample, a discard
  counter, and a prominent warning above a 40% discard rate).
- Thresholds are default hypotheses (0.20 / 0.35 / 0.30), calibrated by the
  user; only K_drift and D_pair gate the exit code — D_const and H_norm
  are ordinal/advisory (REQ-CHK-06 for D_const, REQ-MSR-02 for H_norm).
- For reasoning models — `think: false`; `num_ctx` is checked against an
  estimated prompt size before any HTTP call (silent truncation is a
  measurement-integrity bug, not a warning).

## CLI UX

```
tumanomir check [flags] <file.md|dir>   # deterministic layer: K_drift, D_const
tumanomir measure [flags] <file.md>     # stochastic layer: D_pair, H_norm
tumanomir gate [flags] <file.md>        # CI mode: check + measure (if an
                                         # instrument resolves) in one pass,
                                         # one exit code
tumanomir version                       # print version and exit

# check, measure, and gate
--config  string  path to a .tumanomir.yaml config file (default: load
                   ./.tumanomir.yaml if present, cwd only, no upward
                   search; a named --config path must exist and parse)

# check (and gate)
--k-drift-max  float   gate: max fraction of untraced requirements (default 0.20)
--d-const-min  float   warn: min lexical constraint density (default 0.35)

# measure (and gate, once an instrument resolves)
--instrument     string  format backend:model (e.g. ollama:qwen3-coder:30b);
                          required for measure, optional for gate — an
                          unresolved instrument runs gate deterministic-only
-n, --samples    int     number of generations to sample, must be >=2 (default 10)
--temp           float   sampling temperature (default 1.0)
--sim-threshold  float   single-linkage clustering threshold, in [0,1] (default 0.95)
--num-ctx        int     required: context window; must exceed the prompt token count
--num-predict    int     required: max generated tokens; must exceed natural output length
--think          bool    enable reasoning-model think mode (default false)
--d-pair-max     float   gate: max 1 − mean pairwise AST similarity (default 0.30)
```

`gate` fails with exit code 2 if any measure-specific flag above is passed
explicitly while no instrument resolves (CLI flags or .tumanomir.yaml's
instrument: section) — a silently-downgraded gate run is the same class of
measurement-integrity bug as REQ-MSR-06 (REQ-GATE-02).

Output is human-readable in a TTY; exit code: 0 ok / 1 gate failed / 2 error.

## Package architecture

```
cmd/tumanomir/          CLI (stdlib flag, check/measure/gate/version subcommands)
internal/types.go       shared types (Verdict, Thresholds, InstrumentConfig,
                         KDriftResult, DConstResult, DispersionResult)
internal/config/        loads .tumanomir.yaml (REQ-CFG-02/03)
internal/spec/          markdown specification loading (file or directory)
internal/metrics/       K_drift (traceability linter), D_const (lexical scanner)
internal/dispersion/    AST features, cosine, single-linkage, entropy, D_pair
internal/instrument/    Generator interface, Ollama backend, PromptV1 + fence extractor
internal/report/        renders CheckResult/MeasureResult/Report into a TTY report (REQ-OUT-01)
```

`internal/instrument` is the only package allowed to make network calls
(`internal/nonetwork_test.go` runtime-verifies that `internal/metrics`,
`internal/spec`, and `internal/config` don't violate this — REQ-CHK-05).

Report rendering has been extracted into `internal/report`
(`RenderCheck`/`RenderMeasure`, issue #82): the package depends only on
`internal`, never on `internal/metrics`/`internal/spec` — `aggregate()`
(the per-file aggregation logic) stays in `cmd/tumanomir`; only the
`CheckResult` type it returns moved into `internal/report`. `gate` (issue
#87) adds `Report`/`RenderReport` on top — one `@schema Report` covering
both layers in a single pass; `RenderCheck`/`RenderMeasure` are unchanged
for standalone `check`/`measure`.

Origin of the dispersion code: a port of
`docs/investigation/_sanity/analyze/main.go` from the article's experiment.
