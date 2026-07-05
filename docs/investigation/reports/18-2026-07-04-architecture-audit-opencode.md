# tumanomir — external documentation audit: docs/architecture.md (opencode)

> Model: opencode/nemotron-3-ultra-free. Full-codebase context (agent explored the repo
> directly via its own tools, not embedded doc text). Part of a
> 5-agent x 3-document audit round, 2026-07-04. Read-only: agent was
> explicitly instructed not to edit/create/delete any repo file;
> confirmed clean afterward via git status.

---

Now I have a comprehensive understanding of the codebase. Let me produce the audit report.

## What's good

- **Accurate package diagram**: The package layout in architecture.md matches the actual Go source structure exactly (`cmd/tumanomir`, `internal/types.go`, `internal/spec`, `internal/metrics`, `internal/dispersion`, `internal/instrument`).
- **Metrics table correctness**: The four metrics (`K_drift`, `D_const`, `D_pair`, `H_norm`) are correctly categorized by layer (deterministic vs stochastic) and their definitions align with the code.
- **CLI UX fidelity**: The flag names, defaults, and usage text in architecture.md match `cmd/tumanomir/main.go` almost perfectly.
- **Methodological invariants stated clearly**: The key invariants — D_pair as working metric, H_norm as ordinal signal, instrument-relative reporting, invalid-rate visibility, threshold defaults as hypotheses, zero-network deterministic layer — are all correctly described and match the implementation.
- **Non-network enforcement test**: The mention of `internal/nonetwork_test.go` enforcing REQ-CHK-05 is accurate; the test exists and runs `go list -f '{{.Deps}}'` on `internal/metrics/...` and `internal/spec/...`.
- **Provenance traceability**: The note about dispersion code being ported from `sanity/analyze/main.go` is correct and helpful for reviewers.

## What's bad

- **Missing `--d-pair-max` flag in CLI section**: `architecture.md` lists flags for `measure` but omits `--d-pair-max` (default 0.30), which exists in `main.go:41` and is wired to `th.DPairMax`. This is a user-facing discrepancy.
- **D_const implementation detail mismatch**: The architecture says "markers/(markers+prose tokens)" implying disjoint sets, but `dconst.go:27-49` counts markers via exact byte matching on the whole document and prose tokens via `bytes.Fields(doc)` on the *entire* document — so marker bytes are also counted as prose words. The metric is a proxy; the description oversimplifies.
- **No mention of `VerdictSkipped` states**: The code has two distinct "skipped" verdicts — K_drift when zero `[REQ-*]` tags (line 151-156), and D_pair when `<2` valid samples (line 382-385) — rendered explicitly rather than as "0.00 [ok]". Architecture.md doesn't document this design decision.
- **Discard-rate warning threshold (40%) absent**: `discardWarnThreshold = 0.40` is implemented in `main.go:184` and triggers a prominent `⚠` warning in `printMeasureResult`, but architecture.md never mentions this threshold or the warning UX.
- **Truncation detection (`done_reason=length`) not documented**: REQ-MSR-06's output-truncation signal is implemented (Ollama response `DoneReason` surfaced, counted in `Truncated` field, warned in report), but architecture.md only mentions input-truncation preflight (`num_ctx` check).
- **Instrument config report includes prompt size but architecture.md doesn't note this**: `printMeasureResult` prints `PromptV1 (%d bytes)` (line 441), a useful reproducibility detail not called out.
- **`internal/report` TODO mentioned but current inline structs not described**: The `checkResult` and `measureResult` types in `main.go` (lines 115-120, 193-206) are the *de facto* report structure today; architecture.md only references the future package.

## What it doesn't cover

Gaps relative to `docs/requirements.md` — a reader of architecture.md alone would not know the system must satisfy these:

| Requirement | What's missing in architecture.md |
|-------------|-----------------------------------|
| **REQ-CHK-01** (zero reqs → "skipped" not "0.00") | Not mentioned; the distinct `VerdictSkipped` signal is a key UX decision. |
| **REQ-CHK-02** (hanging IDs listed) | Not mentioned; actionable output is a stated goal. |
| **REQ-CHK-04** (directory recursion, `*.md` only) | Mentioned in passing but no detail on recursive walk or non-`.md` exclusion. |
| **REQ-CHK-06** (D_const advisory-only, never blocks) | Stated but the "never `VerdictBlock`" constraint could be emphasized as a hard invariant. |
| **REQ-MSR-01** (AST feature vector definition) | The operational definition of "mean pairwise structural AST similarity" (type/struct/interface declarations, fields, methods, func signatures, consts) lives only in `requirements.md` and `astfeat.go`; architecture.md says "AST features" without listing them. |
| **REQ-MSR-05** (discard rate 40% warning) | The threshold and its prominent warning UX are implemented but absent from architecture.md. |
| **REQ-MSR-06** (output truncation via `done_reason`) | Only input-truncation preflight is described; output-truncation detection is a separate, implemented signal. |
| **REQ-OUT-01** (one-line-per-metric TTY format) | The actual render format (value, verdict, threshold, context like "N=10 valid, 2 discarded") is not described. |
| **REQ-NFR-01** (100ms check on 1MB corpus) | Performance contract not mentioned. |
| **REQ-NFR-02** (stdlib-only, Go ≥ 1.26) | `go.mod` confirms zero deps, but architecture.md doesn't state this NFR. |

## Verdict

`docs/architecture.md` is a **strong but incomplete** description of the current system. It accurately captures the high-level architecture, package boundaries, metric definitions, CLI surface, and methodological invariants — a new engineer would get the right mental model. However, it has **concrete omissions** that matter for implementation fidelity: the `--d-pair-max` flag, the `VerdictSkipped` states, the 40% discard-rate warning, output-truncation detection, and the exact AST feature set for D_pair. It also doesn't fully trace several `[REQ-*]` requirements to architectural components. Treat it as a living document that needs a targeted update pass before it can serve as a complete reference.
