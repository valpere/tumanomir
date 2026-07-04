# Independent Review: SourceOfTheUnknown-Plan.md

## 1. Strong Points

**Conceptual framing is excellent.** The shift from "spec as truth" to "spec as distribution with measurable tolerance" is the right mental model for AI-era engineering. The three failure modes (ambiguity, incompleteness, untraceability) are cleanly separated and each maps to a distinct metric—this avoids the common trap of conflating them into one "quality score."

**Metrics have engineering teeth.** 
- $K_{drift}$ is fully deterministic, computable as a git hook, no LLM involved—this is the anchor metric teams can adopt today.
- $D_{const}$ correctly penalizes *unbound* prose, not prose per se. The caveat (normalize per requirement, not per document) shows practical awareness.
- $H_{spec}$ adapts self-consistency sampling to *input* uncertainty rather than output uncertainty—novel and directionally correct.

**Honest about instrument noise.** The baseline calibration ($H_{baseline}$) and $\Delta H = H_{spec} - H_{baseline}$ formulation is the single most defensible choice in the document. It transforms $H_{spec}$ from a marketing number into an engineering measurement. The Miller–Madow bias acknowledgment and ceiling-effect warning ($\log_2 N$) are exactly what a senior reviewer would demand.

**Two-mode policy (implementation vs. exploration)** resolves the "entropy is bad" dogma. Gating in implementation, cultivating in exploration—this is actionable orchestration logic, not philosophy.

**Sanity check is real, not performative.** The experiment uses two independent models, reports failure modes (think-mode token exhaustion, context truncation, syntax invalidation), and shows the metric *responding to intervention* (adding architecture docs raised similarity 0.278→0.393). This is rare and valuable.

---

## 2. Weaknesses & Technical Errors

### 2.1 $H_{spec}$ Mathematical Issues

**Cluster threshold (0.95) is arbitrary and uncalibrated.** Single-linkage at 0.95 cosine similarity on AST vectors has no statistical justification. Different thresholds produce different cluster counts → different $H_{spec}$. The paper acknowledges this for kimi (0.95 too strict, 0.80 needed) but doesn't propose a *principled* way to set it. **Fix:** Use stability selection (vary threshold, pick plateau) or replace clustering with continuous mean pairwise distance (which avoids discrete threshold entirely).

**Shannon entropy on cluster proportions is the wrong functional for this decision.** The gate decision is binary: "one cluster (deterministic) vs. many (ambiguous)." Entropy adds no information beyond cluster count $k$ when $N$ is small. Worse, entropy is inflated by many singleton clusters even if they're all *near* each other. **Use $k$ or mean pairwise distance directly for gating; reserve entropy for trend visualization.**

**Plug-in estimator bias at $N=10$ is severe.** Miller–Madow corrects bias for *true* entropy estimation, but here the quantity of interest is $\Delta H$ (difference from baseline). Since baseline also uses $N=10$, bias *partially cancels*—but only if baseline and spec have similar cluster structures. They don't (baseline: 1 cluster; fog: 10 clusters). **Fix:** Report $\Delta H$ with bootstrap confidence intervals, or use a bias-corrected divergence estimator (e.g., NSB).

**Ceiling saturation at $\log_2 10 \approx 3.32$ bits destroys discrimination.** Both real-world specs hit ceiling. The paper correctly notes mean pairwise distance discriminates better—but then *doesn't use it as the primary metric*. **Make mean pairwise AST distance (or 1 - sim) the primary gate metric; entropy is derivative.**

### 2.2 $D_{const}$ Definition is Underspecified

$$D_{const} = \frac{\text{typed edges}}{\text{unstructured text nodes}}$$

- "Typed edges" = `must_satisfy`, `performs`, `flows_to` — but these are *relation types*, not a complete schema. What about attribute constraints (`@constraint(min: 0.01)`)? Are they edges or node properties?
- "Unstructured text nodes" — does a `@schema` block with 5 fields count as 1 node or 5?
- No handling of *nested* structures (a schema containing enums, which contain values).
- **Result:** $D_{const}$ is not reproducible across implementations. Needs a formal graph schema (node/edge types, cardinality rules) and a reference implementation.

### 2.3 $K_{drift}$ Threshold (0.2) is a Free Parameter

The paper admits thresholds are "hypotheses to calibrate on your corpus"—correct. But it gives no *method* for calibration. **Add:** ROC analysis on historical data (specs that caused rework vs. specs that didn't) to pick threshold with target precision/recall.

### 2.4 Baseline Construction is Underspecified

"Reference fully formalized spec" — but what *counts* as fully formalized? The baseline in the experiment is hand-written Go types. That's not a *specification*; it's an implementation. A fair baseline should be a spec *written in the same notation* (markdown + `@schema`) but with zero intentional ambiguity. Otherwise $\Delta H$ conflates notation expressiveness with spec ambiguity.

### 2.5 Temperature Conflation

The protocol runs at `temp=1.0` for "stress test" and `temp=0.3–0.5` for "working measurement" but **never defines which temperature the gate thresholds (0.75, etc.) apply to**. Gates calibrated at temp=1.0 will be far more conservative than at temp=0.3. **Must specify: gate thresholds are defined at a standard temperature (recommend 0.3), and stress-test is a separate reporting mode.**

### 2.6 Single-Linkage Clustering is Fragile

Single-linkage suffers from chaining: a chain of pairwise-similar generations (A≈B, B≈C, C≈D) clusters all together even if A≉D. With $N=10$, one outlier can merge distinct modes. **Use complete-linkage or HDBSCAN with min_cluster_size=2.**

### 2.7 AST Cosine Similarity on Feature Vectors — What Features?

"Go AST → feature vectors → cosine similarity" — but the feature extraction is not described. Bag of node types? Tree kernels? Path-based embeddings? **This must be specified or the metric is not reproducible.** Different feature sets yield different similarity rankings.

---

## 3. What to Improve

| Area | Action |
|------|--------|
| **$H_{spec}$ primary metric** | Replace entropy with **mean pairwise AST distance** (or 1 - mean cosine sim). Report entropy as supplementary. |
| **Cluster threshold** | Eliminate threshold dependency: use continuous distance distribution (e.g., 95th percentile pairwise distance) or stability-selected clustering. |
| **Baseline spec** | Define baseline as *same notation, zero intentional ambiguity* (e.g., the `sharp.md` spec but with all `@constraint` filled, no enums left open). |
| **$D_{const}$ formalization** | Publish a minimal graph schema (JSON Schema or OpenAPI) for RFLP nodes/edges. Provide a reference `go`/`python` implementation. |
| **Gate temperature** | Explicitly declare: all thresholds reference `temp=0.3, N=20`. Stress-test at `temp=1.0` is informational only. |
| **Confidence intervals** | Bootstrap $H_{spec}$ and $\Delta H$ (resample generations with replacement, 1000 iterations). Report 95% CI. |
| **Calibration protocol** | Add Appendix: "How to calibrate thresholds on your corpus" — collect 20-50 historical specs, label by rework outcome, fit logistic regression. |

---

## 4. What to Add

1. **Threat model for metric gaming.** The paper notes $D_{const}$ can be gamed by adding `@schema` tags without content. Add a *coherence check*: every `@schema` field must appear in at least one `must_satisfy` edge to a `Functional` node. $K_{drift}$ partially catches this but a dedicated "schema groundedness" metric is cleaner.

2. **Cross-model portability protocol.** Since $H_{spec}$ is instrument-relative, teams need a way to *translate* thresholds when switching models. Propose: run baseline + 3 reference specs on new model, compute linear rescaling of $\Delta H$, validate on held-out spec.

3. **Cost model.** $H_{spec}$ at $N=10$ costs ~10× generation latency. For large specs (31KB prompt), this is minutes and dollars. Add a **sequential testing** variant: generate sequentially, stop early if running mean distance exceeds gate threshold with 95% confidence (SPRT).

4. **Integration with spec authoring loop.** The "measure → find entropy sources → propose clarification → remeasure" loop is described but not toolized. Sketch a VS Code extension or CLI UX: `spec-check fog.md --interactive` highlights low-similarity AST regions and suggests `@constraint` insertions.

5. **Related work positioning.** Cite: 
   - *Self-consistency* (Wang et al. 2022) — origin of the sampling idea.
   - *Semantic entropy* (Kuhn et al. 2023) — clustering LLM outputs by meaning.
   - *SpecRL* / *SpecFuzz* — specification ambiguity detection via fuzzing.
   - *TLA+ / Alloy* — the formal methods baseline the paper contrasts against.

6. **Failure mode taxonomy extension.** The three modes are good but incomplete. Add:
   - **Contradiction**: Spec contains mutually exclusive requirements (detectable via SMT on formalized constraints).
   - **Obsolescence**: Spec drift relative to implemented code (detectable via $K_{drift}$ on code-to-spec traceability).

7. **Statistical power analysis.** With $N=10$, what's the minimum detectable $\Delta H$? Show power curve: at $N=10$, you can detect $\Delta H > 0.8$ bits with 80% power. If you need finer discrimination, increase $N$ or switch to mean pairwise distance (higher power).

---

## Summary Verdict

**Strong conceptual contribution, weak metric operationalization.** The framing (spec as distribution, tolerance budget, three failure modes, two regimes) is publishable as-is. The metrics ($K_{drift}$, $D_{const}$, $H_{spec}$) are the right *ideas* but need mathematical hardening before a senior audience will trust the gate thresholds. Prioritize: (1) replace entropy with mean pairwise distance, (2) formalize $D_{const}$ graph schema, (3) define baseline as same-notation zero-ambiguity spec, (4) add bootstrap CIs and calibration protocol. With those fixes, this becomes a reference methodology for AI-era specification engineering.
