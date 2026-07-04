# tumanomir

**Туманомір** — a specification-precision measurement tool for AI-driven
software projects. It digitizes the "fog" of your specs before you let
AI agents turn that fog into architecture.

Reference implementation of the *Source of the Unknown* methodology:
specifications consumed by AI agents are not sources of truth but
distributions over possible implementations — so measure the spread.

## Metrics

| Metric | Layer | What it measures |
| --- | --- | --- |
| `K_drift` | deterministic | requirements without `[REQ-*] -> [FUN-*]` trace edges (trace-markup coverage, not implementation correctness) |
| `D_const` | deterministic | lexical density of machine-readable constraints (lexical proxy — rewards markup density, not constraint quality; advisory-only, never blocks) |
| `D_pair` | stochastic (LLM) | 1 − mean pairwise AST similarity of N generations |
| `H_norm` | stochastic (LLM) | cluster entropy / log₂N — ordinal signal only |

The deterministic layer needs no LLM and runs as a git hook.
The stochastic layer generates N Go artifacts from your spec via a fixed
instrument (Ollama) and measures how far apart they land: the wider the
spread, the foggier the spec.

## Usage

```bash
tumanomir check docs/                 # deterministic, instant
```

The stochastic `measure` command (`D_pair`, `H_norm`) is specified in
`docs/requirements.md` §2.2 (REQ-MSR-01..06) but not yet implemented; its
planned invocation looks like:

```bash
tumanomir measure docs/spec.md \
  --instrument ollama:qwen3-coder:30b -n 10  # specified, not yet implemented
```

Exit codes: `0` gates pass · `1` gate failed · `2` error.

All stochastic measurements are **instrument-relative**: results are
reported together with the full instrument configuration and are not
comparable across models without recalibration. Default thresholds are
uncalibrated hypotheses — tune them on your own spec corpus.

## Status

v0.1 in development. See `docs/requirements.md` (written in tumanomir's
own traceable markup — we eat our own dog food).

- `check` (deterministic layer: `K_drift`, `D_const`) is **implemented**.
- `measure` (stochastic layer: `D_pair`, `H_norm`) is **specified**
  (`docs/requirements.md` §2.2, REQ-MSR-01..06) but **not yet implemented**.

## License

[Apache License 2.0](LICENSE).
