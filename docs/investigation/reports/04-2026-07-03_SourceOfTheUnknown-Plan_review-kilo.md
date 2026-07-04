# Report: review of `SourceOfTheUnknown-Plan.md`

Date: 2026-07-03  
Scope: plan/article structure, argument, rigor, and publish-readiness.

## Executive summary
The draft has a strong “three-layer” hook: (1) a familiar engineering dogma (SSOT), (2) a current inflection point (LLM agents + intent drift), and (3) a constructive proposal (treat specs as probabilistic artifacts with measurable ambiguity). The strongest parts are the narrative framing (Acts I–IV) and the concrete failure cascade example (Act V).

The main risks are **category mixing** (philosophical manifesto vs engineering methodology), **underspecified measurement methodology** (what exactly is being measured and how it’s validated), and **missing operationalization** (how a team would adopt this next week without building a research lab). Several technical sections also need tightening: definitions, evaluation protocol, and the relationship between the proposed metrics and real outcomes (cost, regressions, reliability).

## What the plan already does well (strengths)
- **High-concept framing that maps to lived practice**: SSOT as “sacred artifact” → “source of unknownness” is memorable and will resonate with architects and tech leads.
- **Clear narrative structure**: Acts I–V provide pacing and escalating stakes; this is rare in technical writing and helps retention.
- **Concrete “entropy explosion” case study**: the Markdown payments example + agent pipeline cascade is vivid and plausibly real (type erasure, JSONB dumping, context-window blowups).
- **A productive thesis**: ambiguity is unavoidable; therefore treat specs as artifacts with *measurable precision* rather than pretending they are perfect.
- **Motivation for metrics-first thinking**: the “Without metrics, ‘accuracy’ is meaningless” line is a strong pivot from rhetoric to engineering.
- **Some implementation thinking**: the Go “metrics layer” and orchestration gating idea points toward a toolable approach (even if not yet correct/complete).

## Weaknesses / gaps (what will currently break trust)
### 1) “Plan” vs “article draft” mismatch
The file reads as a near-final article draft, not a plan:
- There is no explicit outline of sections, intended outcomes, reading time, or target audience segmentation.
- Key claims appear before definitions, assumptions, and scope boundaries.

**Impact**: reviewers will struggle to judge completeness and feasibility, and edits become “taste debates” instead of closing explicit gaps.

### 2) Missing audience + promise clarity
It’s unclear whether the reader is:
- an architect seeking a methodology,
- an engineering manager seeking governance and cost control,
- a prompt/agent engineer seeking implementation details,
- or a broad tech audience reading an essay.

**Impact**: tone oscillates between manifesto and applied engineering; expectations on rigor become inconsistent.

### 3) Definitions are implied, not specified
Core terms need crisp definitions early, otherwise “metrics” can feel like metaphor:
- **Source of Truth / Source of Unknownness**: what artifact(s) exactly (spec doc, codebase, agent chat logs, tests, traces)?
- **Accuracy / precision of a spec**: accuracy against what ground truth (user intent, reference implementation, acceptance tests)?
- **Ambiguity vs incompleteness vs inconsistency**: these are distinct failure modes and should map to different metrics.
- **Agent pipeline**: define the assumed orchestration model (single agent, multi-agent, tool-calling, CI loop, RAG, graph memory).

### 4) Metric proposals need methodological grounding
The draft introduces \(H_{spec}\), \(D_{const}\), and \(K_{drift}\), but the measurement protocol is incomplete/unclear.

Key issues:
- **Entropy formula vs empirical procedure**: you cite Shannon entropy but don’t define \(x_i\), the sample space, or how \(P(x_i)\) is estimated from 10 generations. If you mean “diversity/variance,” consider naming it accordingly (e.g., *generation dispersion*), and define a robust estimator.
- **AST cosine similarity**: “cosine similarity of AST vectors” requires a specific embedding/featurization. Otherwise it reads as handwaving. If you want AST-level diffs, define the exact diff metric and normalization.
- **Temperature = 1.0**: results will be model- and prompt-dependent. You need a protocol for controlling prompt, model, seed, and tool context, and a rationale for chosen settings.
- **Validity link to outcomes**: why should a high \(H_{spec}\) predict higher bug rate, cost, regressions, or time-to-merge? Add a small validation plan.

### 5) The Go “zero-allocation” section is currently risky
The Go snippet is a good “implementation flavor,” but it’s not defensible as-is:
- Counting spaces/newlines as “tokens of prose” is not a stable proxy for linguistic ambiguity.
- The `@schema` marker is introduced ad-hoc; the doc earlier doesn’t define a spec language or tagging convention.
- Claims about “zero heap allocation” depend on compiler escape analysis and usage patterns; the code includes `ReadFrom`, `bytes.Buffer`, and slice ops that may still allocate depending on input size and growth.
- Footnote markers like `[5]`, `[6, 7]` appear, but there is no bibliography—this undermines credibility.

**Fix direction**: either (a) make this section explicitly “illustrative pseudocode,” or (b) back it with a defined spec annotation scheme + benchmarks + a real tokenizer/constraint extractor.

### 6) Table formatting is broken and the comparison is under-specified
The “Порівняльна таблиця” has multi-line cells that will render incorrectly in many Markdown renderers, and some rows are conceptually vague (e.g., “динамічне коригування reasoning.effort” needs a concrete mechanism and evidence).

### 7) Over-strong claims without citations
Statements like “LLM cannot read between the lines” and sweeping conclusions about “engineering crime” may alienate experienced readers unless softened or supported (citations, empirical examples, or clearer framing as opinion).

## What to improve (concrete edits with high ROI)
### A) Add a “front-matter contract” (first 10–15%)
Add an explicit block near the top:
- **Audience**: who this is for and who it’s not for.
- **Problem statement**: 2–3 sentences, measurable.
- **Thesis**: 1–2 sentences.
- **What the reader will get**: e.g., a measurement framework + a gating workflow + a minimal implementation sketch.
- **Non-goals**: e.g., “not a full formal methods paper,” “not an LLM safety treatise.”

### B) Introduce a taxonomy of spec failure modes
Before proposing metrics, define 3–5 failure modes, each with examples and candidate measurements, for instance:
- **Ambiguity** (multiple plausible implementations)
- **Incompleteness** (missing constraints / acceptance criteria)
- **Inconsistency** (conflicting requirements)
- **Non-testability** (requirements that can’t be turned into checks)
- **Drift** (spec vs code/tests divergence over time)

### C) Tighten metric definitions and rename if needed
Keep the spirit but make them falsifiable:
- **\(H_{spec}\)**: define as *output dispersion under controlled generation*; specify sampling, canonicalization, and distance metric.
- **\(D_{const}\)**: define what counts as a “machine-readable constraint” (schemas, invariants, examples with exact values, OpenAPI, DB constraints, property-based tests).
- **\(K_{drift}\)**: define the knowledge graph schema and what “hanging requirement” means; provide a minimal example graph.

### D) Add a minimal “adoption workflow” (make it actionable)
A short section like “How to use this in a real team”:
- **Step 1**: enforce a spec template that includes constraints, examples, and acceptance checks.
- **Step 2**: compute metrics on every spec change (CI job).
- **Step 3**: define thresholds (e.g., block auto-codegen when dispersion > X).
- **Step 4**: remediation loop: prompt the author to add constraints/examples until metrics improve.
- **Step 5**: track correlation with outcomes (PR rework, bug leakage, agent token spend).

### E) Add a validation / evaluation section
Even a lightweight evaluation will dramatically increase credibility:
- **Offline**: pick 10 historical tasks; compare agent outcomes with/without metric-gated spec improvements.
- **Online**: A/B on similar tickets; track time-to-merge, number of review cycles, regression rate, and token/cost.
- **Confounders**: model changes, tool availability, developer skill.

### F) Tone control: keep the manifesto energy, but bracket it
Preserve rhetorical punch in Acts I–IV, but clearly transition into an “engineering spec” voice for Acts V+:
- Use fewer absolutes (“always/never”).
- Replace moral language (“crime”) with risk framing (“high-risk”, “strongly correlated with failures”).

## What should be added (missing sections/content)
- **Abstract + TL;DR**: 5–8 lines; many readers won’t read the whole piece.
- **Glossary**: define SSOT, intent, entropy, drift, constraint density, orchestration modes.
- **Related work / references**:
  - requirements engineering metrics,
  - ambiguity detection in specs,
  - LLM output variance / sampling dispersion,
  - spec-to-test practices (executable specs, property-based testing).
- **Spec template (appendix)**: provide a concrete Markdown template with:
  - explicit invariants (“must”, “must not”),
  - examples with exact values,
  - acceptance criteria,
  - schemas (OpenAPI / JSON Schema),
  - edge cases and failure handling.
- **A worked “before/after”**: same payment spec improved to raise \(D_{const}\) and lower dispersion; show resulting interface stability.
- **Operational cautions**:
  - metrics can be gamed,
  - “constraint density” can produce over-specification,
  - some domains require exploration, not premature determinism.
- **Tooling sketch**: a minimal CLI architecture (inputs/outputs) and where it plugs into CI and agent orchestrators.

## Quick checklist for revision (editorial)
- Add audience + thesis + non-goals up front.
- Fix the table rendering (single-line cells or HTML table).
- Either remove the Go “zero-alloc” performance claims or back them with benchmark data and a defined extraction method.
- Replace placeholder footnote markers with real references or remove them.
- Make each metric section include: definition → measurement protocol → how to interpret → how to improve the spec.

## Overall assessment
This is a compelling piece with a genuinely useful central idea. With clearer definitions, a rigorous measurement protocol, and an adoption workflow + lightweight validation, it can move from “great essay” to “repeatable engineering method,” which is where it will have the most impact.
