# tumanomir — user's guide

> English translation. Ukrainian original: [`user-guide.md`](user-guide.md)
> — kept as the source of truth; this file is translated for accessibility
> and should stay in sync with it.

A practical, example-driven guide to using `tumanomir` day to day. For the
elevator pitch and the methodology's philosophy, see [`README.md`](../README.md)
and the source article
[`docs/investigation/SourceOfTheUnknown.md`](investigation/SourceOfTheUnknown.md);
that argument is not re-made here.

## 1. What tumanomir is

`tumanomir` is a specification-precision measurement tool for projects
where an AI agent writes the implementation. It computes two independent
metrics: deterministic (`K_drift`, `D_const` — no network, no LLM) and
stochastic (`D_pair`, `H_norm` — generates N Go artifacts from your spec
via Ollama and measures how far apart they land). Four commands: `check`
(deterministic layer), `measure` (stochastic layer), `gate` (both layers
in one pass, for CI), and `calibrate` (correlating the metrics against a
labeled historical corpus). All four are already implemented.

## 2. Install & build

Prerequisite: Go >= 1.26.

```bash
git clone https://github.com/valpere/tumanomir.git
cd tumanomir
make build     # -> bin/tumanomir
```

Make targets a user needs (full list: `Makefile`):

```bash
make build     # go build -o bin/tumanomir ./cmd/tumanomir
make test      # go test ./...
make dogfood   # build + bin/tumanomir check docs/requirements.md
               # (the tool gates its own spec — a smoke test)
make ci        # build + vet + test + lint + dogfood, all together
```

The rest of this guide assumes you're at the repository root with
`bin/tumanomir` already built; substitute your own spec's path where it
matters.

## 3. Quick start

### 3.1. `check` — zero setup, zero network

```bash
bin/tumanomir check docs/requirements.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.20, 0/30 requirements untraced)
  D_const:  0.03  [warn]   (threshold 0.35, 95 markers / 3161 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)
```

This is real output from `make dogfood`'s smoke test: `tumanomir`
measuring its own specification. Run the same against your own spec
(file or directory, recursively `*.md`):

```bash
bin/tumanomir check path/to/your-specs/
```

No configuration or network call is needed — `check` is safe to run as a
git pre-commit hook (see §7.1).

### 3.2. `measure` — the stochastic layer, requires a running Ollama

```bash
bin/tumanomir measure \
  --instrument ollama:qwen3-coder:30b \
  -n 3 --temp 1.0 --sim-threshold 0.95 \
  --num-ctx 8192 --num-predict 2048 \
  docs/investigation/_sanity/specs/sharp.md
```

Real output (a live generation against `ollama:qwen3-coder:30b`, `-n 3`
for speed here; a typical run uses `-n 10`, `--samples`'s default):

```
Instrument config (REQ-MSR-04):
  backend:        ollama
  model:          qwen3-coder:30b
  temperature:    1.00
  samples (N):    3
  think:          false
  num_ctx:        8192
  num_predict:    2048
  sim_threshold:  0.95
  prompt:         PromptV1 (276 bytes)

  D_pair:   0.33  [block]  (95% CI [-0.00, 0.33]; threshold 0.30, mean sim 0.67, N=3 valid, 0 discarded)
  H:        1.58  bits (ordinal signal only, not gated)
  H_norm:   1.00  (ordinal signal only, not gated)

exit code: 1 (gate failed)
```

`--instrument`, `--num-ctx`, and `--num-predict` are required — details in
§4.2.

## 4. Command reference

For the full flag list with defaults, see the table in
[`docs/architecture.md`](architecture.en.md#cli-ux) ("CLI UX") — not
reproduced here, only explained through examples.

### 4.1. `check`

```bash
bin/tumanomir check docs/investigation/_sanity/specs/sharp.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.20, 0/3 requirements untraced)
  D_const:  0.11  [warn]   (threshold 0.35, 10 markers / 84 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)
```

`D_const` is `[warn]` here, not `[block]` — deliberately: D_const is a
lexical proxy (markers vs. prose) and never blocks the exit code; only
K_drift can (REQ-CHK-06).

A directory instead of a file aggregates every `*.md` recursively
(skipping `.`-/`_`-prefixed subdirectories — `.git`, `_sanity`, etc.):

```bash
bin/tumanomir check docs/
```

A custom K_drift threshold and `--format json`:

```bash
bin/tumanomir check --k-drift-max 0.5 --format json docs/requirements.md | jq .
```

```json
{
  "result": {
    "k_drift": {"requirements": 30, "hanging": 0, "hanging_ids": null, "value": 0},
    "d_const": {"constraint_markers": 95, "prose_tokens": 3161, "value": 0.029176904176904175},
    "k_drift_verdict": "ok",
    "d_const_verdict": "warn"
  },
  "thresholds": {"k_drift_max": 0.5, "d_const_min": 0.35, "d_pair_max": 0.3}
}
```

### 4.2. `measure`

`--instrument` (`backend:model`, e.g. `ollama:qwen3-coder:30b`),
`--num-ctx`, and `--num-predict` are required. `--num-ctx` must have
headroom for both the prompt and `--num-predict` — otherwise the run is
rejected before any HTTP call (silent truncation is a measurement-integrity
bug, not a warning, REQ-MSR-06):

```bash
bin/tumanomir measure --instrument ollama:qwen3-coder:30b \
  --num-ctx 100 --num-predict 2048 \
  docs/investigation/_sanity/specs/sharp.md
```

```
measure: generation failed: instrument: estimated prompt tokens (427, len(prompt)/3 heuristic) + num_predict (2048) exceeds num_ctx (100); increase num_ctx or reduce num_predict
```

`-n`/`--samples` must be `>= 2` (a pair is needed for pairwise
similarity). `--sim-threshold` is the single-linkage clustering threshold
for H/H_norm, default 0.95.

**The discard counter and the >40% warning (REQ-MSR-05).** Each sample
gets up to 3 attempts (1 initial + 2 retries); if none produce valid Go,
the sample is discarded — the count is never hidden. When the discarded
fraction exceeds 40% (a hypothesis, not a calibrated constant — the same
status as the 0.20/0.35/0.30 thresholds), the report prints a dedicated
warning line above the metrics:

```
⚠ discard rate: 50% (2/4 generations invalid) — exceeds the 40% hypothesis threshold (REQ-MSR-05); results may be unreliable
```

(The actual `%d/%d` numbers depend on your run — the line above shows the
format from `internal/report/report.go`, not a fabricated example number;
the §3.2 run's 0% discard rate is shown inline in the `D_pair` line as
`0 discarded`.) The same mechanism separately warns about
`done_reason=length` (a generation truncated by `num_predict`) and about
generations whose actual prompt-token count significantly exceeded the
preflight estimate — both also REQ-MSR-06/issue #57, never hidden.

**Instrument-relative measurement.** Every `measure` report opens with the
full instrument configuration (backend, model, temperature, N, think,
num_ctx, num_predict, sim_threshold, prompt version) — not decoration:
a `D_pair` measured under one configuration is **not comparable** to a
`D_pair` under another (a different model, temperature, or N) without
recalibration. "D_pair = 0.33" by itself means nothing — it's only
meaningful together with the `Instrument config` block printed above it.

**Thresholds are uncalibrated hypotheses.** `--d-pair-max` (default 0.30),
like `--k-drift-max`/`--d-const-min`, are starting values from the
methodology's source article, not empirically validated constants. Before
relying on the default as a pass/fail decision for your own project,
accumulate a labeled corpus and run `calibrate` (§4.4) — that's exactly
what it's for.

### 4.3. `gate`

`gate` = `check` + `measure` (if an instrument resolves) in one pass, one
exit code — meant for CI. Without `--instrument` and without an
`instrument:` section in `.tumanomir.yaml`, it runs deterministic-only:

```bash
bin/tumanomir gate docs/investigation/_sanity/specs/sharp.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.20, 0/3 requirements untraced)
  D_const:  0.11  [warn]   (threshold 0.35, 10 markers / 84 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)

exit code: 0 (gates pass)
```

With an instrument — both layers, `--format json`:

```bash
bin/tumanomir gate --instrument ollama:qwen3-coder:30b -n 3 \
  --num-ctx 8192 --num-predict 2048 --format json \
  docs/investigation/_sanity/specs/sharp.md | jq '.result.measure.dispersion'
```

```json
{
  "n": 3,
  "discarded": 0,
  "mean_sim": 0.812044357642638,
  "d_pair": 0.18795564235736195,
  "d_pair_ci_low": 0,
  "d_pair_ci_high": 0.1885752229329093,
  "clusters": 2,
  "sim_thresh": 0.95,
  "h": 0.9182958340544896,
  "h_norm": 0.579380164285695
}
```

**REQ-GATE-02: no silent downgrade.** If you pass a measure-specific flag
(`--temp`, `-n`/`--samples`, `--sim-threshold`, `--num-ctx`,
`--num-predict`, `--think`, `--d-pair-max`) explicitly while no instrument
resolves (neither CLI nor `.tumanomir.yaml`), `gate` does not silently
ignore the flag or fall back to deterministic-only — it refuses with exit
code 2:

```bash
bin/tumanomir gate --temp 0.5 docs/investigation/_sanity/specs/sharp.md
```

```
gate: --temp was passed but no instrument resolved (no --instrument and no .tumanomir.yaml instrument: section) — refusing to silently downgrade to deterministic-only (REQ-GATE-02)
```

This is the same class of measurement-integrity bug as silent truncation
in REQ-MSR-06 — not a convenience that can be turned off.

### 4.4. `calibrate`

`calibrate` never touches the network or invokes an LLM — `d_pair` is
always read from the corpus, never re-measured. The corpus is JSONL, one
row per historical spec:

```jsonl
{"spec_path": "docs/investigation/_sanity/specs/sharp.md", "instrument": "ollama:qwen3-coder:30b", "d_pair": 0.19, "outcome": 0.2}
{"spec_path": "docs/investigation/_sanity/specs/fog.md", "instrument": "ollama:qwen3-coder:30b", "d_pair": 0.55, "outcome": 0.9}
{"spec_path": "docs/investigation/_sanity/specs/baseline.md", "instrument": "ollama:qwen3-coder:30b", "d_pair": 0.30, "outcome": 0.5}
```

```bash
bin/tumanomir calibrate corpus.jsonl
```

```
Calibration over 3 valid row(s), 0 skipped

⚠ fewer than 5 valid rows — correlation coefficients below are not statistically meaningful yet

K_drift   spearman=+0.00
  outcome <= median:  min=0.00 mean=0.00 max=0.00
  outcome >  median:  min=0.00 mean=0.00 max=0.00

D_const   spearman=-0.87
  outcome <= median:  min=0.00 mean=0.05 max=0.11
  outcome >  median:  min=0.00 mean=0.00 max=0.00

D_pair    spearman=+1.00
  outcome <= median:  min=0.19 mean=0.24 max=0.30
  outcome >  median:  min=0.55 mean=0.55 max=0.55

No threshold is auto-selected or written to .tumanomir.yaml — use these numbers to inform your own choice (REQ-NFR-03).
```

(The example above is deliberately tiny — 3 rows instead of the
recommended ≥5 — to show the small-sample warning; your own corpus's
`d_pair`/`outcome` values should come from real `measure` runs and actual
downstream outcomes, not invented numbers.)

`spec_path` must point to an immutable snapshot of the spec (not a live
working file that keeps changing) — the exact snapshot that produced the
paired `d_pair`/`outcome`. All rows in one run must share one `instrument`
value; a second, distinct value anywhere in the corpus is a hard abort
with exit code 2 (REQ-CAL-02), never a per-row skip:

```bash
bin/tumanomir calibrate corpus-mixed.jsonl
```

```
calibrate: corpus mixes instruments "ollama:qwen3-coder:30b" and "ollama:glm-5.1:cloud" — all rows in one run must share the same instrument (REQ-MSR-04)
```

Malformed rows (unparseable, unreadable `spec_path`, `d_pair`/`outcome`
outside `[0,1]`) are skipped and counted, never silently dropped; zero
valid rows is exit code 2.

`calibrate` **informs, it never sets a threshold itself** — it never
writes to `.tumanomir.yaml` and never proposes a single "set
`--d-pair-max` to X" number (REQ-CAL-03/04). The same principle from §4.2
applies here: the default 0.20/0.35/0.30 are hypotheses from the article;
`calibrate` is the tool you use to test them against your own data, not a
replacement for the human decision.

`calibrate` doesn't accept `--format` — text output only, no JSON mode
(REQ-OUT-03 covers only `check`/`measure`/`gate`).

## 5. The `.tumanomir.yaml` config file

`check`/`measure`/`gate` look for `./.tumanomir.yaml` (current working
directory only, no upward search) and load it if present; an explicit
`--config <path>` is authoritative — the named file must exist and parse,
or the command exits 2. Full schema (mirrors
`internal/config/config.go`):

```yaml
thresholds:
  k_drift_max: 0.20    # float, [0,1]
  d_const_min: 0.35    # float, [0,1]
  d_pair_max: 0.30     # float, [0,1]
instrument:
  backend: ollama       # v0.1: "ollama" only
  model: qwen3-coder:30b
  temperature: 1.0
  samples: 10           # int, >= 2
  think: false
  num_ctx: 8192          # int, must have headroom for prompt + num_predict
  num_predict: 2048       # int
  sim_threshold: 0.95     # float, [0,1]
```

`prompt`/`prompt_version` are deliberately absent from the schema — not
configurable (REQ-MSR-04: the prompt must be reproducible from the
report, not an arbitrary per-project value).

**Precedence: CLI flag > config file > built-in default** (REQ-CFG-03).
Example: with the file above (`k_drift_max: 0.10`) present in the current
directory:

```bash
bin/tumanomir check docs/investigation/_sanity/specs/sharp.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.10, 0/3 requirements untraced)
  D_const:  0.11  [warn]   (threshold 0.40, 10 markers / 84 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)
```

The threshold came from the file (`0.10`, `0.40`). An explicit CLI flag
overrides it:

```bash
bin/tumanomir check --k-drift-max 0.5 docs/investigation/_sanity/specs/sharp.md
```

```
  K_drift:  0.00  [ok]     (threshold 0.50, 0/3 requirements untraced)
  D_const:  0.11  [warn]   (threshold 0.40, 10 markers / 84 prose tokens)
  D_pair:   —     (stochastic layer: run `tumanomir measure` with an instrument)
```

`--k-drift-max 0.5` won over the file's `0.10`; `d_const_min` (`0.40`)
from the file is left unchanged since the CLI didn't touch it.

## 6. `--format json`

`check`, `measure`, and `gate` accept `--format json`: exactly one compact
JSON object to stdout, nothing else. Field names and shape are defined by
the Go structs' `json` tags (`CheckResult`, `MeasureResult`, `Report`, and
their embedded types) and are not re-derived here — the authoritative
source is [`docs/requirements.md`](requirements.md)'s REQ-OUT-03 and the
`@schema Report`/`@schema Thresholds`/`@schema InstrumentConfig` blocks in
§1. Any `--format` value other than `text`/`json` is a usage error (exit
2).

Examples (`| jq` pulling out a specific field):

```bash
bin/tumanomir check --format json docs/requirements.md | jq '.result.k_drift.value'
# 0

bin/tumanomir measure --instrument ollama:qwen3-coder:30b -n 3 \
  --num-ctx 8192 --num-predict 2048 --format json \
  docs/investigation/_sanity/specs/sharp.md | jq '.result.dispersion.d_pair'
# 0.18795564235736195

bin/tumanomir gate --format json docs/investigation/_sanity/specs/sharp.md | jq '.result.exit_code'
# 0
```

`check`/`measure`'s JSON carries no `exit_code` field — the process's
actual exit status remains the only signal for these two commands;
`gate`'s JSON carries `result.exit_code` (REQ-GATE-03).

## 7. Workflow patterns

### 7.1. `check` as a git pre-commit hook

`check` is zero-network, zero-LLM — safe for pre-commit:

```bash
#!/bin/sh
# .git/hooks/pre-commit
bin/tumanomir check docs/ || {
  echo "tumanomir check failed — see above" >&2
  exit 1
}
```

(`chmod +x .git/hooks/pre-commit` after creating it.) Remember: exit code
1 here means only K_drift `[block]` — `D_const` never blocks
(REQ-CHK-06), so the hook won't fail on lexical warnings alone.

### 7.2. `gate` as a CI step

```yaml
# .github/workflows/spec-gate.yml (excerpt)
- name: tumanomir gate
  run: |
    bin/tumanomir gate --instrument ollama:qwen3-coder:30b \
      --num-ctx 8192 --num-predict 2048 \
      docs/spec.md
```

One exit code (0/1/2), CI-composable by construction (REQ-OUT-02). If CI
has no Ollama access, omit `--instrument` and `gate` runs
deterministic-only (§4.3) — just don't pass any other measure-specific
flag either, or REQ-GATE-02 refuses early.

### 7.3. Building a `calibrate` corpus — incrementally, over time

`calibrate` needs a labeled corpus (`d_pair` + an actual downstream
outcome) that doesn't exist at the start of a project. A generic recipe,
not tied to any specific project:

1. **Today**, once a spec is settled and you've run `measure`: record the
   `d_pair` (`measure --format json | jq '.result.dispersion.d_pair'`)
   and save an immutable snapshot of the spec itself (copy the file to a
   versioned path, e.g. `specs-archive/2026-07-10-feature-x.md`, or pin a
   git revision — what matters is that `spec_path` later points at
   exactly what was measured).
2. Add a row to your `corpus.jsonl` with the fields known so far
   (`spec_path`, `instrument`, `d_pair`) — `outcome` is still unknown.
3. **Later**, once the actual outcome is known (how many iterations the
   agent needed, whether rework was required, how long review took) —
   define `outcome` on your own scale (higher = worse) and append (or
   edit) that row.
4. Re-run `calibrate corpus.jsonl` periodically — once rows reach ≥5, the
   small-sample warning disappears and the Spearman coefficients become a
   more meaningful signal for whether `D_pair` actually predicts your
   `outcome` on your instrument.

## 8. Troubleshooting

Every line below is a verbatim message from the current code (re-verified
while writing this guide), not a paraphrase.

| Message | Cause | Fix |
| --- | --- | --- |
| `measure: --instrument is required, format backend:model (e.g. ollama:qwen3-coder:30b)` | `measure` without `--instrument` | add `--instrument backend:model` |
| `measure: unsupported backend "openai"; v0.1 supports only "ollama"` | a backend other than `ollama` | v0.1 only supports `ollama` (other instruments are on the roadmap) |
| `measure: --num-ctx is required (must exceed the prompt token count)` | `--num-ctx` not passed (or `<= 0`) | add `--num-ctx <N>` with headroom over the prompt size |
| `measure: generation failed: instrument: estimated prompt tokens (427, ...) + num_predict (2048) exceeds num_ctx (100); increase num_ctx or reduce num_predict` | the preflight check (REQ-MSR-06): prompt + `num_predict` don't fit `num_ctx` | raise `--num-ctx` or lower `--num-predict` |
| `check: --format must be "text" or "json", got "xml"` | `--format` with an unrecognized value | only `text` or `json` |
| `gate: --temp was passed but no instrument resolved (...) — refusing to silently downgrade to deterministic-only (REQ-GATE-02)` | a measure-specific flag on `gate` with no instrument resolved | add `--instrument` (or drop that flag if you wanted a deterministic-only run) |
| `calibrate: corpus mixes instruments "ollama:qwen3-coder:30b" and "ollama:glm-5.1:cloud" — all rows in one run must share the same instrument (REQ-MSR-04)` | a second, distinct `instrument` in the corpus | split the corpus by instrument — one `calibrate` run per instrument |
| `check: exactly one <file.md\|dir> argument required` | `check` called with zero or multiple arguments | pass exactly one file/directory |
| `measure: exactly one <file.md> argument required` | `measure` called with zero or multiple arguments | pass exactly one spec file |
| `gate: exactly one <file.md> argument required` | `gate` called with zero or multiple arguments (and never a directory) | pass exactly one spec file |
| `calibrate: exactly one <corpus.jsonl> argument required` | `calibrate` called with zero or multiple arguments | pass exactly one JSONL corpus path |

## 9. Where to go next

- [`docs/architecture.md`](architecture.en.md) — how the tool is built:
  packages, the full CLI flag table, methodological invariants.
- [`docs/requirements.md`](requirements.md) — the authoritative behavior
  specification (`[REQ-*]` traces, `@schema` blocks) — the source of truth
  for the JSON schema and each flag's exact behavior.
- [`docs/roadmap.md`](roadmap.en.md) — what's not built yet, and in what
  order (tactical debt lives in [GitHub issues](https://github.com/valpere/tumanomir/issues)).
