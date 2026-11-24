# tumanomir — roadmap

> English translation. Ukrainian original: [`roadmap.md`](roadmap.md) —
> kept as the source of truth; this file is translated for accessibility
> and should stay in sync with it.
>
> This used to be an unordered list at the end of `design.md`. Here it's
> ordered by horizon, with reasoning for the order. **Tactical debt**
> (bugs, small improvements, test gaps) lives in
> [GitHub issues](https://github.com/valpere/tumanomir/issues), not here;
> this file is only about functionality that doesn't exist yet.

## v0.1 — done

`check` (K_drift, D_const), `measure` (D_pair, H_norm, with a 95% bootstrap
CI for D_pair — REQ-MSR-07), and `gate` (both layers in one pass, one exit
code for CI, REQ-GATE-01..03) all work end-to-end against a real Ollama
instance. Details: [`architecture.md`](architecture.md).

## v0.2 — done

`measure --keep-samples` persists generations, discarded samples, timeouts
and the instrument configuration (REQ-MSR-11); `measure --decls` adds a
per-declaration agreement report (REQ-MSR-12); the Ollama per-request
timeout is configurable, and a timed-out sample is regenerated instead of
aborting the run. The D_pair bootstrap CI is no longer biased low by ~1/N
(REQ-MSR-07) — intervals from v0.1 are not directly comparable.

## Mid-term — discussed, not scheduled

1. **Data for `tumanomir calibrate`.** The tool itself is now built
   (`calibrate <corpus.jsonl>`, issue #94, REQ-CAL-01..05): it reads a
   JSONL corpus of historical specs, each paired with a pre-measured
   D_pair and a caller-defined outcome score, computes each of
   K_drift/D_const/D_pair's Spearman rank correlation against outcome,
   and prints a median-split summary — informing, never auto-setting, a
   threshold. What's still open isn't the tool, it's the data: no real
   outcome-labeled corpus exists yet (ragivka/session-indexer haven't
   started accumulating rows). This item stays on the roadmap until such
   a corpus exists.

## Exploratory — an idea from the article, not yet scoped

2. **RFLP graph (Neo4j) for full D_const.** The current D_const is a
   lexical proxy (markers vs. prose). A full
   Requirement-Flow-Linkage-Property graph would give a structural
   constraint-density measure instead of a lexical approximation.
3. **Assisted K_drift mode (LLM parser).** The current K_drift requires
   explicit `[REQ-*] -> [FUN/LOG/PHY-*]` markup. An LLM-assisted parser
   could infer traceability from specs without markup — at the cost of
   the deterministic layer's determinism (REQ-CHK-01..06 explicitly
   require zero-LLM).
4. **Other instruments.** OpenAI/Anthropic API backends alongside Ollama —
   `instrument.Generator` is already designed as a pluggable interface
   for exactly this.
5. **Other projections.** SQL DDL, OpenAPI instead of only Go type
   definitions as `measure`'s generation target — extends applicability
   beyond Go projects.
