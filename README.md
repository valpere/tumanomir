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
make install   # -> $GOBIN/tumanomir (or $GOPATH/bin), on $PATH
```

Sample run against tumanomir's own dogfooded spec (`make dogfood` runs the
same command):

```bash
bin/tumanomir check docs/requirements.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.20, 0/30 requirements untraced)
  D_const:  0.03  [warn]   (threshold 0.35, 95 markers / 3161 prose tokens)
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

`gate` runs both layers (or just the deterministic one, if no instrument
resolves) in a single pass for CI, and `calibrate` correlates
K_drift/D_const/D_pair against a labeled historical corpus of specs —
neither auto-sets a threshold. `label` is the sole writer of a corpus
row's outcome, resolved by `spec_hash` prefix — no other command ever
computes or guesses it. An optional `.tumanomir.yaml` config file
lets `check`/`measure`/`gate` read thresholds and instrument settings
instead of repeating them as flags every time (CLI flag still wins). Add
`--format json` to any of `check`/`measure`/`gate` for one compact JSON
object on stdout instead of the TTY report.

For worked examples of every command and flag, `.tumanomir.yaml`'s full
schema, and a troubleshooting table of real error messages, see
[`docs/user-guide.md`](docs/user-guide.md)
([`.en.md`](docs/user-guide.en.md)).

Exit codes: `0` gates pass · `1` gate failed · `2` error.

All stochastic measurements are **instrument-relative**: results are
reported together with the full instrument configuration and are not
comparable across models without recalibration. Default thresholds are
uncalibrated hypotheses — tune them on your own spec corpus.

## Status

Current release: [`v0.2.0`](https://github.com/valpere/tumanomir/releases/tag/v0.2.0)
(first tagged release: `v0.1.0`).
`check`, `measure`, `gate`, `calibrate`, and `label` are all **implemented**
and work end-to-end. See `docs/requirements.md` (written in tumanomir's own
traceable markup — we eat our own dog food), `docs/architecture.md` for how
it's built, `docs/user-guide.md` for how to use it, and `docs/roadmap.md`
for what's not built yet.

## Limitations

`D_pair` measures generation spread at a **fixed instrument** (model +
prompt + temperature + N). By itself, it cannot separate how much of that
spread comes from spec ambiguity versus inherent model stochasticity/noise.
`calibrate` (see the user's guide) correlates `D_pair` against a
caller-defined outcome on your own historical corpus, but a real
outcome-labeled corpus is still being accumulated (see `docs/roadmap.md`)
— until then, a single `D_pair` number should be read as "spread under
this specific instrument," not as an instrument-independent measure of
"how foggy the spec truly is."

The "copy-floor" reference point (feeding the model already-complete,
unambiguous Go type definitions produced `H=0` in the source experiment) is
instrument-dependent, not a universal floor guaranteed by the metric — a
different or weaker model might not reach zero entropy even on a fully
unambiguous spec.

**N=10 cannot separate nearby specs** (one instrument, one spec family; not a
general law). Two dreet-dojo `req-uc` revisions (`32cc10f`, `58504a8`) were
measured with `gemma4:31b-cloud` at temp 1.0. A field run reported a jump
from 0.19 to 0.25 between them. At larger N that jump disappears:

| revision | N=10 | N=20 | N=30 |
|---|---|---|---|
| `32cc10f` | 0.234 [0.183, 0.280] | 0.211 [0.186, 0.236] | 0.220 [0.194, 0.242] |
| `58504a8` | 0.211 [0.158, 0.264] | 0.235 [0.207, 0.263] | 0.216 [0.186, 0.245] |

At N=30 both revisions sit at about 0.22, and their 95% CIs overlap fully.
The CI is about 0.10–0.11 wide at N=10 and about 0.05–0.06 wide at both N=20
and N=30. The same `32cc10f` spec gave D_pair between 0.18 and 0.24 across
four N=10 runs: the dreet-dojo field run (0.19), the two #163 acceptance
runs (0.18 and 0.24) and this table's run (0.234). Only the table's run has
kept samples. A difference smaller than the CI width is therefore not
evidence that a spec changed. To compare revisions, use N ≥ 20 (see the
user's guide).

`measure --decls` (per-declaration agreement) is a heuristic diagnostic, not
a measurement: declarations are matched into slots by name and by
member/enum-value overlap (Jaccard >= 0.5, >= 2 shared elements), so a slot
split or merge is possible. Field renames (`WeakTags` / `WeakestTags`) are
not matched in v1 and appear as two partly-present slots.

## Contributing

Issues, bug reports, and pull requests are welcome — see [`CONTRIBUTING.md`](CONTRIBUTING.md).

## License

[Apache License 2.0](LICENSE).
