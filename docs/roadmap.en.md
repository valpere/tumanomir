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

`check` (K_drift, D_const), `measure` (D_pair, H_norm), and `gate` (both
layers in one pass, one exit code for CI, REQ-GATE-01..03) all work
end-to-end against a real Ollama instance. Details: [`architecture.md`](architecture.md).

## Mid-term — discussed, not scheduled

1. **`tumanomir calibrate`.** The 0.20/0.35/0.30 thresholds are hypotheses
   from the article, not calibrated on any real team's data. `calibrate`
   would run `measure` over a labeled corpus (known-sharp vs. known-foggy
   specs) and propose domain/team-specific thresholds. Needs an
   accumulated history of real measurements first (no issue filed yet —
   waiting on enough real-world `measure` usage).
2. **Bootstrap CI for D_pair.** N=10 generations is a point estimate with
   visible sampling noise (evident even in
   `docs/investigation/_sanity/README.md`'s A/B instrument comparison). A
   confidence interval instead of a single number would be more honest
   about that uncertainty.

## Exploratory — an idea from the article, not yet scoped

3. **RFLP graph (Neo4j) for full D_const.** The current D_const is a
   lexical proxy (markers vs. prose). A full
   Requirement-Flow-Linkage-Property graph would give a structural
   constraint-density measure instead of a lexical approximation.
4. **Assisted K_drift mode (LLM parser).** The current K_drift requires
   explicit `[REQ-*] -> [FUN/LOG/PHY-*]` markup. An LLM-assisted parser
   could infer traceability from specs without markup — at the cost of
   the deterministic layer's determinism (REQ-CHK-01..06 explicitly
   require zero-LLM).
5. **Other instruments.** OpenAI/Anthropic API backends alongside Ollama —
   `instrument.Generator` is already designed as a pluggable interface
   for exactly this.
6. **Other projections.** SQL DDL, OpenAPI instead of only Go type
   definitions as `measure`'s generation target — extends applicability
   beyond Go projects.
