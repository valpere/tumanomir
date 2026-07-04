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

## Build

Prerequisite: Go >= 1.26.

```bash
make build     # -> bin/tumanomir
```

Sample run against tumanomir's own dogfooded spec (`make dogfood` runs the
same command):

```bash
bin/tumanomir check docs/requirements.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.20, 0/18 requirements untraced)
  D_const:  0.07  [warn]   (threshold 0.35, 63 markers / 871 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)
```

## Usage

```bash
tumanomir check docs/                 # deterministic, instant
```

The stochastic `measure` command (`D_pair`, `H_norm`) is specified in
`docs/requirements.md` §2.2 (REQ-MSR-01..06) and requires a running Ollama
instance:

```bash
tumanomir measure docs/spec.md \
  --instrument ollama:qwen3-coder:30b \
  -n 10 --temp 1.0 --sim-threshold 0.95 \
  --num-ctx 8192 --num-predict 2048
```

`--instrument` (backend:model), `--num-ctx` and `--num-predict` are
required — `num-ctx` must have headroom for both the prompt and
`num-predict`, or the run is rejected before any generation is attempted
(silent truncation would be a measurement-integrity bug). `-n`/`--samples`
must be `>= 2` to compute a pairwise similarity. See `tumanomir --help`
for the full flag list.

Exit codes: `0` gates pass · `1` gate failed · `2` error.

All stochastic measurements are **instrument-relative**: results are
reported together with the full instrument configuration and are not
comparable across models without recalibration. Default thresholds are
uncalibrated hypotheses — tune them on your own spec corpus.

## Status

v0.1 in development. See `docs/requirements.md` (written in tumanomir's
own traceable markup — we eat our own dog food).

- `check` (deterministic layer: `K_drift`, `D_const`) is **implemented**.
- `measure` (stochastic layer: `D_pair`, `H_norm`) is **implemented**
  end-to-end against Ollama (`docs/requirements.md` §2.2, REQ-MSR-01..06).

## Limitations

`D_pair` measures generation spread at a **fixed instrument** (model +
prompt + temperature + N). By itself, it cannot separate how much of that
spread comes from spec ambiguity versus inherent model stochasticity/noise.
Without a calibration baseline across multiple instruments/models (the
roadmap `calibrate` command), a single `D_pair` number should be read as
"spread under this specific instrument," not as an instrument-independent
measure of "how foggy the spec truly is."

The "copy-floor" reference point (feeding the model already-complete,
unambiguous Go type definitions produced `H=0` in the source experiment) is
instrument-dependent, not a universal floor guaranteed by the metric — a
different or weaker model might not reach zero entropy even on a fully
unambiguous spec.

## License

[Apache License 2.0](LICENSE).
