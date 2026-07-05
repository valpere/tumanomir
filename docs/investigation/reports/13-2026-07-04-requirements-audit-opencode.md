# tumanomir — external documentation audit: docs/requirements.md (opencode)

> Model: opencode/nemotron-3-ultra-free. Full-codebase context (agent explored the repo
> directly via its own tools, not embedded doc text). Part of a
> 5-agent x 3-document audit round, 2026-07-04. Read-only: agent was
> explicitly instructed not to edit/create/delete any repo file;
> confirmed clean afterward via git status.

---

Now I have a thorough understanding of the entire codebase. Let me compile the audit report.

---

## Audit Report: `docs/requirements.md` — tumanomir v0.1

### What's good

1. **Traceable markup discipline is genuine** — The document uses its own `[REQ-*] -> [FUN-*]` and `@schema/@constraint` markup consistently (18 REQs, 18 FUNs, plus LOG/PHY references). This enables dogfooding: `bin/tumanomir check docs/requirements.md` actually works and produces a meaningful report (K_drift=0.00, D_const=0.07).

2. **Methodological invariants are explicit and non-negotiable** — The "Methodological invariants" section in CLAUDE.md (mirrored in REQ-CHK-06, REQ-MSR-02, REQ-MSR-04, REQ-MSR-05, REQ-MSR-06) correctly codifies the article's core principles: D_pair as working metric, H_norm as ordinal-only, instrument-relative measurements, invalid-rate transparency, thresholds as uncalibrated hypotheses, deterministic layer zero-network. This is rare clarity for a v0.1 spec.

3. **Schema blocks are machine-readable and precise** — `@schema Thresholds`, `@schema InstrumentConfig`, `@schema Report` with `@constraint` annotations (defaults, ranges, rules) make the data model testable and document the "hypothesis, not constant" status of defaults explicitly (e.g., the comment on `sim_threshold` in InstrumentConfig).

4. **Traceability from REQ → FUN → code is verifiable** — Every REQ maps to a FUN with a concrete Go signature (e.g., `metrics.KDrift`, `dispersion.Analyze`, `instrument.Generator`). The main.go `aggregate` and `runMeasureWithGenerator` functions directly implement the aggregation/gating logic described in REQ-CHK-06, REQ-MSR-05, REQ-MSR-06.

5. **Test coverage verifies requirements, not just code** — Tests like `TestRunCheckNoRequirementsIsSkipped` (REQ-CHK-01), `TestRunMeasureWithGeneratorLowDPairAllValid` (REQ-MSR-01/04), `TestPrintMeasureResultDiscardWarningVisibility` (REQ-MSR-05's 40% threshold), `TestOllamaGeneratePreflightRejectsWithoutHTTPCall` (REQ-MSR-06), and `TestNoNetworkImports` (REQ-CHK-05) validate the *spec's behavioral contracts*, not just implementation details.

6. **Honest about limitations** — README.md "Limitations" section, REQ-MSR-05's "hypothesis not constant" language, and the article's "Limits of the approach" section all explicitly state what the metrics *cannot* do (separate spec fog from model noise, guarantee correctness, transfer thresholds across models).

---

### What's bad

1. **REQ-MSR-04's "prompt as versioned named constant" is not enforced by the type system** — `InstrumentConfig.Prompt` is a plain `string` populated at callsite (`cfg.Prompt = instrument.PromptV1`). Nothing prevents a caller from passing an inline literal or mutated string. The schema comment says "named, versioned package-level constant, not an inline literal" but there's no compile-time or runtime check. This is a measurement-integrity hole the spec claims to close.

2. **REQ-MSR-02 claims H_norm is "the one actually reported/gated on" but the code gates on D_pair, not H_norm** — The requirement says: *"H_norm is an ordinal signal only... and it's the one actually reported/gated on."* Yet `runMeasureWithGenerator` gates exit code 1 exclusively on `DPair > th.DPairMax` (line 386), and `printMeasureResult` prints H/H_norm as "ordinal signal only, not gated". The requirement text contradicts the implemented gating logic.

3. **REQ-MSR-05's 40% discard-rate threshold is hardcoded in main.go, not in Thresholds** — `discardWarnThreshold = 0.40` is a package-level constant in `cmd/tumanomir/main.go:184`, not part of `internal.Thresholds`. The requirement says "The proposed threshold is 40%... stated here as a hypothesis, not a calibrated constant, the same treatment given to the 0.20/0.35/0.30 thresholds elsewhere in this document (see REQ-CFG-01)." But REQ-CFG-01's `DefaultThresholds()` doesn't include it, and there's no CLI flag to override it. Inconsistent treatment.

4. **REQ-MSR-06's `think=false` requirement is not validated at CLI parse time** — The `measure` command accepts `--think true` without warning. The article explicitly warns that reasoning-mode silently consumes the token budget on complex specs (sanity check finding #5). The spec says "the instrument *must* set think=false for reasoning models" but leaves enforcement to the operator. A warning or rejection at flag-parse time would match the "measurement-integrity bug" severity the requirement assigns to silent truncation.

5. **REQ-CHK-04 says `check` accepts "a single file or a directory (recursively, *.md only)" — but `measure` rejects directories with a methodological justification comment** (`measure: %s is a directory; measure takes a single spec file (directory aggregation is not methodologically meaningful for dispersion measurement in v0.1)`). This asymmetry is documented only in an error message, not in the requirements. Either both commands should accept directories (with aggregation semantics explained) or the spec should explicitly state the difference.

6. **No test verifies the AST feature extraction format** — REQ-MSR-01 says *"The authoritative implementation... is internal/dispersion/astfeat.go; this requirement points to it rather than duplicating it, so the two must not drift."* But there are zero tests in `internal/dispersion/` at all. The feature-key format (type:Name, field:Type:Name:Type, func:Recv.Name:Signature, etc.) is the *operational definition* of "structural AST similarity" — if it drifts, D_pair becomes incomparable across versions. This is a critical unverified requirement.

7. **REQ-OUT-01's "one line per metric with value, verdict, threshold" is not fully met for `measure`** — The `check` output matches (K_drift, D_const lines). But `measure` prints the instrument config block first, then D_pair/H/H_norm. The H and H_norm lines lack verdict labels and thresholds (correctly, since they're not gated), but the spec doesn't distinguish this — it says "one line per metric with value, verdict and the threshold it was judged against" universally.

---

### What it doesn't cover

#### 1. Gaps relative to *SourceOfTheUnknown.md* (the source article)

| Article concept | In requirements.md? | Notes |
|----------------|---------------------|-------|
| **ΔH = H(S) − H(S_baseline)** calibration baseline | ❌ Dropped entirely | The article makes ΔH the *diagnostically meaningful* quantity (§5.3 "Instrument and its noise", formula 3). requirements.md only reports absolute H/H_norm. Without baseline subtraction, a single D_pair/H number is unactionable — exactly what the article warns against. |
| **Two temperature regimes** (stress-test @ temp=1.0 vs working @ temp=0.3–0.5) | ❌ Dropped | Article §5.3 explicitly separates these: stress-test numbers don't go into gates. requirements.md has a single `--temp` flag (default 1.0) with no regime distinction. |
| **RFLP graph (Neo4j) for true D_const** | ✅ Acknowledged as out-of-scope (roadmap #5) | Article's D_const is graph-based (typed edges); requirements.md implements only the lexical proxy. This is honest but the proxy's limitations should be more prominent in the spec. |
| **Assisted K_drift (LLM parser)** | ✅ Acknowledged as out-of-scope (roadmap #6) | Article describes both strict (markup) and assisted (LLM) modes. requirements.md correctly restricts v0.1 to strict mode (REQ-CHK-05). |
| **Exploration vs Implementation zones** (different gate policies) | ❌ Dropped | Article §6 "Dispersion: defect or resource?" defines two zones with opposite policies (gate vs cultivate). requirements.md has only the implementation-zone gates. |
| **Bootstrap CIs for D_pair** | ✅ Acknowledged as mid-term (roadmap #4) | Article notes N=10 bias and saturation. requirements.md doesn't mention CI at all — the single-point estimate is presented as the metric. |
| **Calibration procedure** (historical corpus → downstream consequences) | ✅ Acknowledged as mid-term (roadmap #3) | Article §7 "Ladder of claims" lists threshold calibration as "hypotheses needing corpus". requirements.md REQ-CFG-01 says "calibrate them on your own spec corpus" but provides no procedure. |
| **Cross-instrument baseline suite** (formalized spec without Go types) | ❌ Dropped | Article sanity check finding #1: "copy-floor was zero on both instruments *but our baseline contained ready-made Go types* — not a universal floor." requirements.md doesn't require or define a baseline suite. |

#### 2. Gaps relative to the actual codebase

| Implemented behavior | In requirements.md? | Notes |
|---------------------|---------------------|-------|
| **`sim_threshold` default 0.95 is hardcoded in CLI, not in `DefaultThresholds()`** | ⚠️ Partially | Schema comment admits "no default currently wired in code — SimThreshold is a caller-supplied parameter... 0.95 is a proposed hypothesis default". But the CLI default (main.go:229) and the schema default diverge in principle. |
| **`DoneReason` truncation detection (REQ-MSR-06)** | ✅ Covered | Implemented in `runMeasureWithGenerator` (truncated count) and `printMeasureResult` (warning line). Tests cover it. |
| **CRLF normalization in `ExtractGoBlock`** | ❌ Not specified | Implementation detail with measurement-integrity impact (prevents fence-match failures inflating discard rate). Tested but not required. |
| **Single-message prompt vs system+user split** | ⚠️ Acknowledged in code comment | `BuildPrompt` comment: "v0.1 simplification... instrument-relativity is about fixing *some* prompt and reporting it, not matching gen.sh's message structure exactly." Spec doesn't mention this deviation. |
| **`measure` rejects directories** | ❌ Not in spec | See "What's bad" #5. |
| **`internal/dispersion` has zero tests** | ❌ Critical gap | REQ-MSR-01 points to `astfeat.go` as authoritative; no tests guard against drift. |
| **`VerdictSkipped` for zero-requirement and <2-valid-sample cases** | ✅ Covered | Both `check` and `measure` render distinct "— [n/a/skipped]" lines, not misleading 0.00. Tests verify this. |
| **`measure` prints instrument config with `PromptV1 (N bytes)`** | ✅ Covered | REQ-MSR-04 satisfied; the prompt constant name and byte count are printed. |

---

### Verdict

**`docs/requirements.md` is a strong, unusually honest v0.1 specification — but it is not yet a fully controlling artifact.** It excels at traceability, methodological clarity, and testable functional requirements for the deterministic layer. The stochastic layer (`measure`) has three material gaps between spec and implementation: (1) H_norm gating claim vs D_pair-only gating, (2) 40% discard threshold not in Thresholds/CLI, (3) missing baseline/ΔH/temperature-regime concepts from the article are absent, making single-run numbers unactionable per the article's own validity argument. The zero tests in `internal/dispersion/` means the operational definition of D_pair (the working metric) is unguarded against drift. **Fix these four items and the spec becomes genuinely controlling; as-is, it's a reliable guide for the deterministic layer and a partial, slightly misleading guide for the stochastic layer.**
