# tumanomir — history and provenance

> English translation. Ukrainian original: [`history.md`](history.md) — kept
> as the source of truth (a point-in-time session record); this file is
> translated for accessibility (see issue #21) and should stay in sync
> with it. Historical facts below (e.g. dogfood numbers) are preserved as
> they were recorded on the date noted, not updated retroactively.

## Where the project came from (2026-07-03)

Born from the article **"Source of the Unknown: Stochastic Engineering and
Precision Measurement in the Age of AI Agents"** —
`docs/investigation/SourceOfTheUnknown.md`.

The article defines the methodology; tumanomir is its reference
implementation. The article teases a "separate, purely engineering
follow-up" — tumanomir and its development are the groundwork for that
second article (scanner implementation, graph schema, calibration). Keep
notes with that in mind.
Chain: article → 10 reviews (5 external CLI agents: agy, codex, cursor,
kilo, opencode) → consolidated verdict
(`docs/investigation/reports/11-2026-07-03-consolidated-verdict.md`) →
the experiment that was run (`docs/investigation/_sanity/`, 120
generations, 2 instruments) → this tool.

## Key decisions already made (with rationale)

1. **D_pair is the working metric, not entropy.** Consensus from 5/5
   reviewers plus our own experiment: H@0.95 saturates at log₂N for real
   documents (all real specs gave 3.32 bits at N=10) — only mean pairwise
   distance discriminates.
2. **Theory via conditional entropy H(C|S,θ)**, NOT via a sum of variances
   σ²spec+σ²model — 4/5 reviewers took the latter apart as a category
   error (non-separable components, entropy ≠ variance).
3. **K_drift strict mode is the foundation**: a deterministic markup
   linter, no LLM, the most defensible metric in front of a senior
   audience. Assisted mode (LLM graph parser) is roadmap — it inherits
   extraction error.
4. **D_const v0.1 is an honest lexical proxy**, not a graph-based metric.
   A full RFLP graph (Neo4j) is roadmap. The density formula penalizes
   prose without constraints, not prose in general.
5. **The article's baseline is a copy-floor** (transcribing ready-made
   types gave H=0 on both instruments), not a universal floor. A baseline
   suite is roadmap for `calibrate`.

## Empirical reference points (for tests and calibration)

Instrument A = qwen3-coder:30b (Ollama local), B = kimi-k2.7-code:cloud
(think=off); temp=1.0, N=10, Go projection, sim = mean pairwise AST cosine:

| Spec | A: sim | B: sim |
| --- | --- | --- |
| baseline (ready-made types) | 1.000 | 1.000 |
| sharp (@schema markup) | 0.730 | 0.682 |
| fog (2 sentences of prose) | 0.492 | 0.242 |
| session-indexer requirements | 0.524 | 0.258 |
| ragivka requirements | 0.282 | 0.278 |
| ragivka + architecture docs | 0.402 | 0.393 |

Main reproduced effect: adding structural docs → +0.120/+0.115 sim on two
independent instruments.

## Decision: D_pair_shape — no-go (2026-11-24, #171)

The question #106 (`naming_noise_test.go`) left open: whether to add an additive, non-gating
`D_pair_shape` field on identifier-canonicalized keys next to D_pair. The field data came from
#163/#169: dreet-dojo `req-uc` @ `32cc10f`, fixture
`internal/dispersion/testdata/dreet-dojo-req-uc-32cc10f/`, N=9. The archived `_sanity/out*/`
corpus was used alongside it. Val's decision: **no-go**.

A prototype (scratch, not in the repo) computed mean-pairwise D_pair under five key modes:

- `base`: the `astfeat.go` keys.
- `slot`: local type names rewritten to the #169 slot labels.
- `slotsig`: `slot` plus type-only signatures.
- `keepmem`: every type name rewritten to `T`.
- `drop`: every declared identifier dropped.

| set | base | slot | slotsig | keepmem | drop |
|---|--:|--:|--:|--:|--:|
| naming-noise (N=3) | 0.667 | 0.667 | 0.667 | 0.667 | 0.000 |
| naming-noise-reorder | 0 | 0 | 0 | 0 | 0 |
| dreet-dojo (N=9) | 0.246 | 0.187 | 0.182 | 0.034 | 0.013 |
| sharp A / cloud | 0.270 / 0.318 | 0.257 / 0.269 | 0.257 / 0.269 | 0.131 / 0.055 | 0.177 / 0.024 |
| fog A / cloud | 0.508 / 0.758 | 0.460 / 0.747 | 0.459 / 0.747 | 0.177 / 0.192 | 0.095 / 0.137 |
| real-si A / cloud | 0.476 / 0.742 | 0.476 / 0.681 | 0.465 / 0.679 | 0.182 / 0.182 | 0.073 / 0.117 |
| real-ragivka A / cloud | 0.718 / 0.722 | 0.671 / 0.701 | 0.668 / 0.700 | 0.091 / 0.037 | 0.093 / 0.044 |
| real-ragivka-full A / cloud | 0.598 / 0.607 | 0.589 / 0.590 | 0.587 / 0.589 | 0.130 / 0.034 | 0.127 / 0.047 |

Rationale:

- **`drop` is rejected.** On instrument A it inverts sharp and fog: sharp scores 0.18 and fog 0.09.
- **`keepmem` is rejected.** It merges distinct types into one, so real-ragivka falls from 0.72
  to 0.04. That is collapse, not denoising.
- **`slot`/`slotsig` are safe but weak.** They reorder nothing on either instrument, but they move
  D_pair by only 2–24%. Parameter names contribute ≤ 0.005. A second column would enlarge the
  report without improving any decision.
- **The hypothesis is not confirmed.** On dreet-dojo, type renames explain ≈ 0.06 of 0.25. The
  remaining ≈ 0.18 comes from declaration-set divergence: types and fields that some samples
  have and others lack. That is structure, and `measure --decls` (REQ-MSR-12) already itemizes
  it. So the hypothesis that naming noise makes up most of D_pair is not confirmed.
- **The naming-noise fixture stays at 0.67 in every safe mode** and even under `keepmem`. It varies field names, not just type names, and only
  semantic matching could fold those; #106 rejected that.
- **Revisit when `calibrate` (REQ-CAL-01) has labeled rows.** Same-spec run-to-run spread
  (`32cc10f`: 0.18 vs 0.24) suggests that N-sensitivity is the more pressing question.

## Ollama pitfalls (from the experiment, baked into the requirements)

- Thinking mode eats `num_predict` on complex specs → empty content; the
  failure correlates with input complexity → systematic distortion.
  `think:false`.
- The default `num_ctx` (4096) silently truncates large prompts.
- `num_predict` below the natural output length → truncation → a false
  "separate cluster" signal.
- Cloud `:cloud` models are ~6x faster than local 30b models.

## State at the end of the 2026-07-03 session

- Scaffold: go.mod (module github.com/valpere/tumanomir), git init (no
  commits yet).
- `docs/requirements.md` — written, in its own markup (dogfood).
- Implemented: internal/types.go, internal/metrics (KDrift, DConst + 5
  tests), internal/dispersion (a port of the article's analyzer),
  internal/spec (Load: file/directory), cmd/tumanomir with the `check`
  command (thresholds via flags, exit codes, hanging IDs).
- First dogfood run (`check docs/requirements.md`): K_drift 0.00 (0/17
  untraced), D_const 0.07 [warn] — expected, rationale prose drags down
  lexical density; hence D_const is warn, not a gate (decided in code).
- Not yet done (next session): internal/instrument (Ollama, REQ-MSR-03/06),
  the `measure` command (REQ-MSR-01/02/04/05), a report package
  (REQ-OUT-01 — currently rendered inline in main.go), the REQ-NFR-01
  benchmark, a no-network-imports test for REQ-CHK-05. Dispersion
  integration tests — on data at `docs/investigation/_sanity/out*/` (now
  in-repo).
