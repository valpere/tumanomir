# tumanomir — external documentation audit: docs/requirements.md (cursor)

> Model: cursor-agent --model auto. Full-codebase context (agent explored the repo
> directly via its own tools, not embedded doc text). Part of a
> 5-agent x 3-document audit round, 2026-07-04. Read-only: agent was
> explicitly instructed not to edit/create/delete any repo file;
> confirmed clean afterward via git status.

---

## What's good

**Traceability discipline is real, not decorative.** Every numbered requirement in sections 2–3 carries a `-> [FUN-*]`, `-> [LOG-*]`, or `-> [PHY-*]` edge to a concrete symbol or enforcement point. The document dogfoods its own `check` semantics — `make dogfood` runs `tumanomir check docs/requirements.md` in CI. That is unusually rigorous for a v0.1 spec.

**Methodological invariants are stated with unusual honesty.** REQ-CHK-06 (D_const advisory-only, never blocks), REQ-MSR-02 (H/H_norm ordinal, not the primary gate), REQ-MSR-05 (invalid rate visible, 40% warning is a hypothesis), and REQ-CFG-01 (thresholds are uncalibrated hypotheses) all encode article-level epistemic humility as testable policy, not marketing.

**Operational definitions where it matters.** REQ-MSR-01 defers the AST feature-vector definition to `internal/dispersion/astfeat.go` rather than duplicating it — a sound anti-drift pattern. REQ-CHK-01's "no requirements found" vs numeric `0.00` distinction (verdict `skipped` / `[n/a]`) is precise and backed by `TestRunCheckNoRequirementsIsSkipped`.

**@schema blocks ground the data model.** `Thresholds`, `InstrumentConfig`, and `Report` give structure to CLI flags, instrument config, and output. The inline honesty on `sim_threshold` ("no default currently wired in code") is better than silent fiction.

**Section 4 cleanly scopes v0.1.** Explicit out-of-scope items (`gate`, `calibrate`, RFLP-graph D_const, assisted K_drift) align with `docs/roadmap.md` and prevent the spec from pretending the article is fully productized.

**Test coverage is meaningfully tied to requirements where it exists.** REQ-CHK-05 has `internal/nonetwork_test.go`. REQ-MSR-05/06 have thorough `cmd/tumanomir/main_test.go` and `internal/instrument/ollama_test.go` coverage (discard warnings, truncation via `done_reason=length`, preflight before HTTP). REQ-CHK-01/04/06 logic is exercised through `TestAggregate` and integration-style `runCheck` tests.

---

## What's bad

**Stale provenance reference.** Line 5 points to `context/history.md`; the repo uses `docs/investigation/history.md`. A controlling spec should not reference a path that does not exist.

**Internal inconsistencies in the spec itself.**

- `@schema Report` defines `verdict: Enum["ok","warn","block"]` and a unified `Report` with optional metric fields, but the implementation uses `VerdictSkipped` / `[n/a]` (REQ-CHK-01) and splits rendering across inline `checkResult`/`measureResult` in `main.go` — not `report.Render(w, Report)` as REQ-OUT-01 claims.
- `@schema InstrumentConfig` lists `sim_threshold` default `0.95` while simultaneously noting it is not wired — schema and reality disagree in one block.
- REQ-OUT-01 requires "one line per metric with value, verdict, and threshold." `measure` output for H and H_norm omits verdict and threshold (`ordinal signal only, not gated`). REQ-OUT-01 is not met as written.

**REQ-MSR-06 wording vs implementation.** The requirement says reasoning-capable models "must set think=false," but the CLI exposes `--think` to enable think mode, and `ollama_test.go` explicitly tests `think=true`. The code treats `false` as the measurement-default, not a hard prohibition — the requirement overstates what is enforced.

**D_const operationalization is underspecified and slightly misleading.** REQ-CHK-03 says `markers / (markers + prose tokens)`; the implementation counts prose as `len(bytes.Fields(doc))` over the entire document, including words inside marker regions (`dconst.go` comment acknowledges this). The ratio is therefore not "markers vs prose outside markers" as a reader of the requirement would assume. No test pins the prose-token semantics.

**Gating policy is incomplete in the requirements layer.** REQ-CHK-06 explicitly documents that D_const never blocks. Nothing symmetric states that D_pair gates `measure` exit code 1, or that `check` exit code 1 is exclusively K_drift-driven. REQ-OUT-02 says "all gates pass" without enumerating which metrics gate which command. A reviewer must infer this from `@schema Thresholds`, CLI usage strings, and code.

**Requirements with weak or no verification.**

| Requirement | Gap |
|---|---|
| REQ-NFR-01 (1 MB / 100 ms, benchmark in `internal/metrics`) | No `Benchmark*` anywhere in the repo |
| REQ-MSR-01 (`dispersion.Analyze`, AST features) | No `internal/dispersion/*_test.go`; `testdata/` with reference numbers exists for a *future* test per its README |
| REQ-OUT-01 (`report.Render`) | Package does not exist; `TODO(REQ-OUT-01)` in `main.go` |
| REQ-CFG-01 (CLI threshold overrides) | No test passes `--k-drift-max` / `--d-pair-max` / `--d-const-min` |
| REQ-CHK-01 LOG/PHY trace edges | Tests cover only `-> [FUN-*]` |
| REQ-OUT-02 end-to-end exit codes | `TestAggregate` and `runMeasureWithGenerator` test verdict logic; no test asserts `runCheck` returns 1 on K_drift block or `runMeasure` returns 1 on D_pair block |

**Cross-document contradiction on H gating.** `docs/requirements.md` and the code agree: H/H_norm are ordinal, not gated. `CLAUDE.md` line 45 says H_norm is "the one actually reported/**gated** on" — contradicting REQ-MSR-02 and actual behavior. For a document billed as controlling, sibling docs drifting on gating policy is a governance failure.

---

## What it doesn't cover

### Relative to `docs/investigation/SourceOfTheUnknown.md`

**Explicitly deferred (good):** `gate` command, `.tumanomir.yaml`, `calibrate`, bootstrap CIs, RFLP-graph D_const, assisted K_drift, non-Ollama instruments, non-Go projections — all listed in section 4.

**Silently dropped or weakened without a roadmap flag:**

- **Three failure modes framing** (ambiguity / incompleteness / untraceability mapped to D_pair, D_const, K_drift) — central to Act II of the article, absent from requirements. The metrics are there; the pedagogical and diagnostic taxonomy is not.
- **ΔH baseline calibration** (`ΔH = H(S) − H(S_base)`) — article treats this as mandatory for measurement validity; requirements mention ordinal H but not baseline subtraction or a `calibrate` prerequisite beyond section 4's one-liner.
- **Article YAML gate on `delta_H` max 0.75** — article's orchestrator example blocks on H; requirements correctly demote H to ordinal (post-experiment lesson) but never note this deliberate divergence from the article's CI example.
- **Dual temperature protocol** — article distinguishes stress-test (`temp ≈ 1.0`) vs working measurement (`temp ≈ 0.3–0.5`) and says gate thresholds apply only at working temperature. Requirements default `temperature: 1.0` with no mention of the two-mode protocol or which mode v0.1 thresholds assume.
- **Implementation vs exploration zones** — article's "gate dispersion in realization zone / cultivate in search zone" policy engine is entirely absent (fair for v0.1, but not flagged as a methodology gap).
- **Closed-loop remediation** (measure → locate entropy sources via clusters → propose clarifications → remeasure) — article's operational loop; requirements stop at measurement and gating.
- **Methodology limits** — "low entropy ≠ correctness," "metrics are gameable," instrument-relative comparisons — discussed in the article and `history.md`, not encoded as requirements or documented limitations.
- **D_const article formula** (typed graph edges / unstructured nodes) — requirements correctly use lexical proxy, but the relationship to the article's graph-based definition is only implicit via section 4; the proxy's known inaccuracy vs the article's intent is not stated in requirements prose.

### Relative to the codebase

**Implemented behavior with no corresponding requirement:**

- `tumanomir version` subcommand
- `measure` rejects directory arguments (tested in `TestRunMeasureFlagValidation`; rationale is only a code comment)
- D_pair / H skipped state when fewer than 2 valid samples survive discards (implemented and tested; not a numbered REQ)
- Ollama operational details: default base URL, HTTP timeout, context cancellation, malformed-response handling
- `check` prints a static D_pair placeholder line directing users to `measure`
- `--d-pair-max` flag on `measure` (implied by `Thresholds` schema but not named in REQ-CFG-01's trace edge, which only points to `DefaultThresholds()` and "flag wiring in cmd")

**Requirement claiming implementation that does not exist:**

- REQ-OUT-01 → `report.Render` / unified `Report` type
- REQ-NFR-01 → benchmark in `internal/metrics`
- REQ-PHY-NFR-01 trace edge is aspirational

---

## Verdict

`docs/requirements.md` is a strong v0.1 specification for the deterministic layer and the core stochastic measurement protocol: traceable, methodologically self-aware, and mostly aligned with what the code actually does today. It is not yet solid enough to serve as the project's sole controlling specification without qualification. Stale references, schema/implementation drift on `Report` and output rendering, incomplete gating documentation, untested dispersion core and performance NFR, and unacknowledged divergences from the source article (H gating, ΔH baseline, temperature modes, orchestrator loop) mean a conscientious implementer or auditor must triangulate against `main.go`, `docs/architecture.en.md`, and `docs/roadmap.md` to resolve ambiguities. For v0.1 scope — `check` + `measure` as measurement instruments, not a full orchestrator productization — it is usable as a controlling spec for the metrics and CLI behavior already built, provided the gaps above are treated as known debt rather than assumed complete. Tightening gating policy into explicit REQs, fixing the stale path and `Report`/`REQ-OUT-01` mismatch, adding dispersion fixture tests against `testdata/`, and documenting deliberate article divergences would move it from "good draft with honest section 4" to "controlling specification as-is."
