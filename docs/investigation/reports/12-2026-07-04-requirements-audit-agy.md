# tumanomir — external documentation audit: docs/requirements.md (agy)

> Model: Gemini 3.1 Pro (High) via agy -p. Full-codebase context (agent explored the repo
> directly via its own tools, not embedded doc text). Part of a
> 5-agent x 3-document audit round, 2026-07-04. Read-only: agent was
> explicitly instructed not to edit/create/delete any repo file;
> confirmed clean afterward via git status.

---

I will start by checking the current working directory and listing its contents to understand the structure of the "tumanomir" project.
I will list the contents of the `docs` directory to locate the requirements document, investigation notes, and architecture documentation.
I will list the contents of the `docs/investigation` directory to check for `SourceOfTheUnknown.md`.
I will read the target document under audit, `docs/requirements.md`.
I will run a command to list the markdown headings in `docs/investigation/SourceOfTheUnknown.md` to understand its structure and content before diving deeper.
I will search for all Go files in the project to see the actual implementation layout and see what tests and source files exist.
I will run the project's test suite to verify the current codebase's state and see if all tests pass.
I will check `internal/dispersion/dispersion.go` to see the structure and implementation of D_pair and entropy calculation.
I will view `internal/dispersion/cluster.go` to understand how clustering and entropy are calculated.
I will view `internal/dispersion/astfeat.go` to inspect the feature extraction logic.
I will check the test files in `internal/metrics/metrics_test.go` to see what is verified by the unit tests.
I will check `internal/metrics/kdrift.go` to see the implementation of K_drift.
I will view `internal/metrics/dconst.go` to check how syntactic constraint density (D_const) is computed.
I will view `internal/types.go` to see the definitions of metrics and results.
I will check `cmd/tumanomir/main.go` using `view_file` to see the command-line interface logic, flag parsing, gating logic, and TTY output rendering.
I will view `cmd/tumanomir/main_test.go` to see what tests are written to verify the main CLI interface.
I will view the rest of `cmd/tumanomir/main_test.go` to see the remaining flag validation tests.
I will view `internal/instrument/instrument.go` to see the Generator and Generation structures.
I will view `internal/instrument/prompt.go` to see the structure of the prompt sent to the LLM and verify how it is constructed.
I will view `internal/instrument/ollama.go` to inspect the Ollama client implementation.
I will view `internal/nonetwork_test.go` and `internal/spec/spec.go` to see how the no-network imports are enforced and how spec loading is implemented.
I will view `internal/spec/spec.go` to see the logic for loading files/directories.
I will view `internal/spec/spec_test.go` to see the unit tests for loading markdown specs.
I will run a grep search on `docs/investigation/SourceOfTheUnknown.md` to locate the definitions and formulas for the metrics to verify if the translation into requirements was accurate and complete.
I will grep for "const" in the investigation document to see the definition and details of D_const.
I will search for `pair` and `entropy` (including Ukrainian spelling) in the investigation document to check how the stochastic layer metrics are defined.
I will search for the term `ентропі` (Ukrainian for entropy) in the investigation document.
I will view lines 153 to 185 of `docs/investigation/SourceOfTheUnknown.md` to see the details of D_pair and entropy calculation.
I will search the investigation document for the numbers "0.3", "0.2", "0.35" to see where they are introduced.
I will view lines 186 to 220 of `docs/investigation/SourceOfTheUnknown.md` to see the details of the thresholds and orchestration loop context.
I will run tumanomir's deterministic `check` on its own requirements document (`docs/requirements.md`) to see if the dogfooding measurement passes or reveals any traceability gaps (hanging requirements or density warnings).
I will read `CLAUDE.md` to see if there are any specific guidelines or methodology invariants mentioned there.
I will view `docs/architecture.md` to see the architecture details and context.
I will view `docs/roadmap.md` to see the roadmap items, which helps identify gaps between what is documented as present versus what is planned.
I will view `docs/investigation/history.md` to see if there is any other historical context or design rationale that is relevant to our audit.
This report presents an audit of the [docs/requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md) specification document for the Go CLI project **tumanomir** (v0.1, in development), evaluating its quality, technical consistency, coverage of the source article [SourceOfTheUnknown.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/investigation/SourceOfTheUnknown.md), and alignment with the current implementation.

---

## What's good

* **Traceability Discipline**: Each of the 18 requirements defined in [docs/requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md) is explicitly linked to its corresponding package, function, struct, or environment restriction (e.g., `[REQ-CHK-01] -> [FUN-CHK-01]`, `[REQ-NFR-03] -> [LOG-NFR-03]`). This structure creates a highly auditable requirements trace.
* **Dogfooding as a Smoke Test**: The requirements document itself is written in the spec's own traceable markup (`[REQ-*] -> [FUN-*]`, `@schema`), allowing the tool to execute its parsing logic on its own specification (demonstrated by the `check` command running successfully against [docs/requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md)).
* **Clean Separation of Layers**: The document clearly distinguishes the deterministic layer (`check` command), which operates without external model dependencies, from the stochastic layer (`measure` command), which relies on LLMs.
* **Network Isolation Requirement**: The constraint defined in `[REQ-CHK-05]` ensures that core deterministic metric calculation remains isolated from network imports, guaranteeing its usability as a lightweight pre-commit hook (enforced programmatically in [TestNoNetworkImports](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/nonetwork_test.go#L15-L42)).
* **Methodological Realism on Lexical Proxies**: `[REQ-CHK-06]` explicitly recognizes that $D_{const}$ is merely a lexical proxy. It limits failure results to `VerdictWarn` at most, preventing CI build-blocking (`VerdictBlock`) over rich prose descriptions (such as design justifications or trade-offs).

---

## What's bad

* **Deflating $D_{const}$ Formula (Mathematical Bug)**: 
  Requirement `[REQ-CHK-03]` defines lexical constraint density as $markers / (markers + prose\_tokens)$. However, the actual implementation of [metrics.DConst](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/metrics/dconst.go#L24-L51) computes prose tokens using `len(bytes.Fields(doc))`. Because the marker tokens (e.g. `@schema`, `@constraint`, `[REQ-PAY-01]`, `-> [FUN-`) are space-separated strings, they are double-counted as *both* markers and prose tokens. Under this formula, a document consisting of *only* valid engineering markup and zero prose incorrectly yields a calculated density of `~0.50` instead of a perfect `1.00`.
* **Inaccurate Prompt Token Estimation for Cyrillic Inputs**: 
  `[REQ-MSR-06]` states that silent truncation of the input spec is a measurement-integrity bug. To prevent this, the preflight check in [Ollama.Generate](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/instrument/ollama.go#L86-L154) computes `estimatedPromptTokens := (len(prompt) + 2) / 3`. A 3-byte-per-token heuristic is highly inaccurate for Cyrillic UTF-8 text (where characters occupy 2 bytes each). This underestimate compromises the preflight verification, risking silent input truncation for specifications written in Ukrainian (such as the project's own source article).
* **Contradictory Gating and Temperature Settings**:
  `[REQ-CFG-01]` and the `@schema InstrumentConfig` define a default temperature of `1.0` and a default quality gate threshold of `0.30` for `d_pair_max`. However, the source article states that a temperature of `1.0` is a "stress-test" configuration that maximizes model noise, and that quality gates should be evaluated at lower working temperatures (`0.3-0.5`). Gating against `0.30` at `temp: 1.0` defaults is a methodological inconsistency that violates the article's core warning against mixing stress-test noise with gate thresholds.
* **Unimplemented Schema Constraint Engine**: 
  Although [docs/requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md) defines explicit schema fields (such as `Thresholds`, `InstrumentConfig`, `Report`) with constraints like `@constraint(range: [0,1])` and `@constraint(rule: "...")`, the codebase provides no parser or engine to read or validate these schemas. They are treated purely as static lexical markers.
* **SimThreshold Default Wiring Inconsistency**: 
  The schema constraint comment for `sim_threshold` in [docs/requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md#L29) states: `"no default currently wired in code"`. However, the command-line flags in [cmd/tumanomir/main.go](file:///home/val/wrk/projects/tumanomir/tumanomir/cmd/tumanomir/main.go#L229) explicitly wire a default of `0.95` for `--sim-threshold`.

---

## What it doesn't cover

### 1. Gaps relative to the source article ([SourceOfTheUnknown.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/investigation/SourceOfTheUnknown.md))

* **Calibration Baseline ($S_{base}$ and $\Delta H$)**: The article defines establishing a calibration baseline using a zero-uncertainty spec ($S_{base}$) and measuring the delta ($\Delta H = H - H_{base}$) as an "obligatory step" (`обов'язковий крок`) to separate spec ambiguity from model noise. However, [docs/requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md) pushes the `calibrate` command and the calculation of relative delta ($\Delta H$ / $\Delta D_{pair}$) to the roadmap (Out of scope for v0.1). Consequently, the specification gates on raw absolute values, violating a key methodology requirement of the source article.
* **Cluster Extraction & Debugging UX**: The article explains that finding where a specification "drifts" or "spreads" requires inspectable clusters ($H_{spec}$ equivalence classes). While [docs/requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md) requests reporting $H$ and $H_{norm}$ as ordinal signals, it lacks requirements for exposing the contents of these clusters to help users trace specification flaws.
* **Orchestrator Policies**: The article proposes policy actions based on metrics (e.g. reducing model reasoning effort if a spec is highly constrained). The requirements document drops these downstream orchestration integration scenarios entirely.

### 2. Gaps relative to the actual codebase

* **Missing `internal/report` Package**: Requirements `[REQ-OUT-01]` and `[REQ-OUT-02]` trace to a non-existent `report.Render(w io.Writer, r Report)` function. There is no `internal/report` package; the rendering logic is hardcoded inside [cmd/tumanomir/main.go](file:///home/val/wrk/projects/tumanomir/tumanomir/cmd/tumanomir/main.go#L114) with inline `TODO(REQ-OUT-01)` comments.
* **Stochastic Layer (`internal/dispersion`) Testing Gaps**: There are no unit tests verifying the mathematical calculations (mean similarity, clustering, entropy) in [internal/dispersion](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/dispersion) (`[no test files]`). The calculations are only verified transitively through CLI-level mock tests.
* **`Report` and `exit_code` Struct Mismatch**: The `@schema Report` details a structure that does not exist in [internal/types.go](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/types.go). The exit code validation is only handled programmatically as a CLI exit status rather than being verified against schema constraints.
* **The `think` Constraint Engine**: `[REQ-MSR-06]` specifies that the tool must set `think=false` for reasoning-capable models, but the codebase does not implement any discovery or lookup engine to detect reasoning models, leaving this verification entirely to user-supplied command line flags.

---

## Verdict

[docs/requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md) is **not** solid enough to serve as the controlling specification as-is and requires work first. While its traceability structure is commendable, it suffers from significant methodology drift relative to the source article (specifically by omitting mandatory baseline calibration and misapplying stress-test temperatures to quality gates) and outlines functional targets (such as schema validation, the `Report` data structure, and the `internal/report` package) that are completely absent, inlined as TODOs, or unverified in the codebase.
