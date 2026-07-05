# tumanomir — external documentation audit: docs/roadmap.md (cursor)

> Model: cursor-agent --model auto. Full-codebase context (agent explored the repo
> directly via its own tools, not embedded doc text). Part of a
> 5-agent x 3-document audit round, 2026-07-04. Read-only: agent was
> explicitly instructed not to edit/create/delete any repo file;
> confirmed clean afterward via git status.

---

## What's good

**Horizon ordering is coherent and mostly well justified.** The near-term pair (`internal/report` → `.tumanomir.yaml` + `gate`) forms a clear dependency chain: shared rendering must exist before a CI command can emit human and machine-readable output without duplication. That matches what `architecture.md` explicitly defers (`TODO(REQ-OUT-01)`) and what `requirements.md` §4 lists as out of scope for v0.1.

**The v0.1 “done” baseline is accurate.** Code confirms `check` and `measure` are implemented end-to-end (`cmd/tumanomir/main.go`, `internal/instrument/`, `internal/dispersion/`). Only two code TODOs exist, both tagged `REQ-OUT-01` and both referenced in item 1. Nothing on the roadmap is already shipped under another name.

**Items are concrete enough to act on.** Near-term work names packages, commands, config filenames, and predecessor links. Mid-term items (`calibrate`, bootstrap CI) describe inputs (labeled corpus, N=10 sampling noise) and point at empirical evidence (`docs/investigation/_sanity/README.md`). Exploratory items name technologies (Neo4j, OpenAI/Anthropic, OpenAPI/SQL DDL) and tie them to methodological constraints (zero-LLM deterministic layer vs assisted mode).

**The tactical-debt boundary is structurally respected.** The preamble explicitly redirects bugs, small improvements, and test gaps to GitHub issues. Open issues on `valpere/tumanomir` are currently empty (GitHub API), and nothing on the roadmap reads like a one-line bugfix (“fix typo”, “add missing test for X”) except where it names real not-yet-built functionality. The document aligns cleanly with `requirements.md` §4, which is essentially the same eight-item list in compressed form.

**Cross-document consistency is strong.** `architecture.md` → roadmap pointer works; `README.md` “Limitations” references `calibrate` consistently; Ukrainian original + English translation (`roadmap.en.md`) is maintained.

---

## What's bad

**JSON / machine-readable output is implied but never a first-class roadmap item.** Item 1 says structured output is a “prerequisite” for `gate`, but there is no standalone deliverable such as `--format json` or a documented `Report` serialization path. `requirements.md` already defines a `@schema Report` and points `FUN-OUT-01` at `report.Render(w io.Writer, r Report)`, yet no unified `Report` type exists in `internal/types.go` today. A future implementer could extract inline TTY formatters into `internal/report` and still leave JSON unbuilt.

**Tier placement reasoning is thin for several items.**  
- Bootstrap CI (#4) is independent of `internal/report`, `gate`, and `calibrate`; it could plausibly sit near-term as a reporting honesty improvement to existing `measure`. The doc says “discussed, not scheduled” but not *why* it waits.  
- “Other instruments” (#7) is labeled exploratory despite `instrument.Generator` already existing and Ollama being the only backend — this is a bounded extension, not an unscoped research idea.  
- The “Explicitly out of scope (for now): None” section is meta rather than actionable; it adds no planning value.

**`gate` scope is underspecified.** Item 2 says `gate = check + measure` with one exit code and YAML config, but not: config schema shape, whether `measure` is optional in CI, how instrument secrets/endpoints are supplied, or whether directory aggregation (supported by `check`, not by `measure`) applies. That may be intentional deferral, but it weakens “concrete enough to act on” for the highest-priority item after report extraction.

**Stale sibling docs amplify ambiguity.** `docs/investigation/history.md` still lists `measure`, `internal/instrument`, and the no-network test as “not yet done” — all implemented now (`internal/nonetwork_test.go` exists). The roadmap itself is current; adjacent provenance text is not, which can mislead anyone cross-reading.

**No duplication with GitHub issues was verifiable as a positive.** With zero open issues, the boundary is untested in practice: several gaps below fit the roadmap’s own “test gaps → issues” rule but have no issue filed either, so they fall into a documentation void rather than a clean split.

---

## What it doesn't cover

Relative to **`docs/architecture.md` specifically**, the only explicit deferred marker is `TODO(REQ-OUT-01)` / `internal/report/` — **covered** by roadmap item 1. Architecture does not name other unfinished features beyond TTY-only output (implicitly subsumed under item 1’s JSON rationale).

Gaps that **architecture.md, requirements.md, or code** point at but **roadmap.md does not**:

| Gap | Where it’s referenced | In roadmap? |
|-----|----------------------|-------------|
| **`internal/report` / REQ-OUT-01** | `architecture.md`, `main.go` TODOs | Yes (#1) |
| **`.tumanomir.yaml` + `gate`** | `requirements.md` §4, REQ-NFR-02 (“no YAML until gate”) | Yes (#2) |
| **`calibrate` + bootstrap CI** | `requirements.md` §4, `README.md` Limitations | Yes (#3–4) |
| **RFLP graph, assisted K_drift, other instruments/projections** | `requirements.md` §4, `history.md` decisions | Yes (#5–8) |
| **REQ-NFR-01 performance benchmark** (`check` on 1 MB corpus < 100 ms) | `requirements.md` REQ-NFR-01 → `benchmark in internal/metrics`; `history.md` “not yet done” | **No** — no `Benchmark*` tests in repo |
| **Dispersion fixture/integration tests** | `internal/dispersion/testdata/README.md` (“future test package”); `history.md` “integration tests on sanity data” | **No** — roadmap says test gaps belong in issues, but no issue exists |
| **Unified `Report` model** matching requirements `@schema Report` | `requirements.md` data model + FUN-OUT-01 | **No** (only partial via item 1 wording) |
| **Non-TTY output mode as user-facing feature** | Implied by CI/`gate`; architecture: “human-readable in TTY” | **Partial** — mentioned as prerequisite, not deliverable |

Code TODO/FIXME grep found **only** the two `REQ-OUT-01` markers; nothing else in source comments demands roadmap entry.

Not architectural gaps but worth noting: **`measure` accepts a single file while `check` accepts a directory** — documented in `architecture.md` CLI UX, not framed as intentional limitation or future work on the roadmap. **`sim-threshold` calibration / stability selection** (raised repeatedly in investigation reviews) has no roadmap home except indirectly via `calibrate`.

---

## Verdict

`docs/roadmap.md` is a **useful and mostly trustworthy** planning document for the methodology-scale features deferred from v0.1: it mirrors `requirements.md` §4, correctly records what is done, and orders the next CI-facing work (`report` → `gate`) with real dependency logic. It is **not yet complete as the single registry of known unfinished work**: REQ-NFR-01, dispersion regression tests against prepared fixtures, and explicit JSON/`Report` serialization are real gaps with no home here and (currently) no GitHub issues either. Tightening near-term scope (what `gate` and structured output actually ship), explaining why bootstrap CI and alternate instruments sit where they do, and either filing issues for NFR/test debt or adding a minimal “requirements fulfillment” line would make the document fully trustworthy before treating it as the authoritative backlog.
