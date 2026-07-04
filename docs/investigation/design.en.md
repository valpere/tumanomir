# tumanomir — v0.1 design

> English translation. Ukrainian original: [`design.md`](design.md) — kept
> as the source of truth; this file is translated for accessibility (see
> issue #21) and should stay in sync with it.

Specification-precision measurement tool for AI-driven projects.
Productization of the methodology from the article "Source of the Unknown"
(`docs/investigation/SourceOfTheUnknown.md`).

## Metrics

| Metric | Layer | What it measures | Instrument |
| --- | --- | --- | --- |
| `K_drift` | deterministic | requirements without a `[REQ-*] -> [FUN/LOG/PHY-*]` trace | markup linter, no LLM |
| `D_const` | deterministic | lexical density of constraints (markers vs. prose) | scanner, no LLM |
| `D_pair` | stochastic | 1 − mean pairwise AST similarity of N generations | LLM via Ollama |
| `H_norm` | stochastic | cluster entropy / log₂N — ordinal signal | same |

Methodological invariants (from the article, do not roll back):
- D_pair is the working metric; H_norm (= H / log₂N) is ordinal ("one
  cluster or many") and the one actually reported/gated on; raw H (bits)
  is computed internally but saturates at log₂N for small N.
- Metrics are instrument-relative: the configuration (model, prompt, temp,
  N, threshold) is fixed and reported.
- Invalid rate is reported, never hidden (retry with a discard counter).
- Thresholds are default hypotheses (0.2 / 0.35 / 0.30), calibrated by the
  user.
- For reasoning models — `think: false`; `num_ctx` must fit the
  specification.

## CLI UX

```
tumanomir check <file.md|dir>       # deterministic layer, instant, git-hook-ready
tumanomir measure <file.md> \
  --instrument ollama:qwen3-coder:30b -n 10 --temp 1.0   # stochastic layer
```

Output is human-readable in a TTY; exit code: 0 ok / 1 gate failed / 2 error.

## Architecture

```
cmd/tumanomir/          CLI (stdlib flag, subcommands)
internal/types.go       shared types (Report, Verdict, Thresholds)
internal/spec/          markdown specification loading
internal/metrics/       K_drift (linter), D_const (lexical scanner)
internal/dispersion/    AST features, cosine, single-linkage, entropy, D_pair
internal/instrument/    Generator interface + Ollama backend
```

Origin of the dispersion code: a port of `sanity/analyze/main.go` from the
article's experiment.

## Roadmap (not in v0.1 — YAGNI)

- `.tumanomir.yaml` config + `gate` command (CI mode)
- baseline calibration (`tumanomir calibrate`)
- bootstrap CI for D_pair
- RFLP graph (Neo4j) for full D_const; assisted K_drift mode (LLM parser)
- other instruments (OpenAI/Anthropic API), other projections (SQL DDL, OpenAPI)
