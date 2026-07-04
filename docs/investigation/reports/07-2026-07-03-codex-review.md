# Independent review of `SourceOfTheUnknown-Plan.md`

Date: 2026-07-03  
Reviewer stance: technical sanity review for a senior engineering audience, with focus on metrics, math, measurement validity, and defensibility of claims.

## Executive summary

The article plan has a strong and timely central idea: in AI-agent workflows, specifications should be treated as measurement-bearing engineering artifacts, not as static "truth". The best parts are the separation of ambiguity, incompleteness, and traceability drift; the notion of a precision budget; and the admission that LLM-based measurement has instrument noise.

However, the current version overclaims in several places. The proposed metrics are promising as operational heuristics, but they are not yet mathematically clean enough to carry the weight implied by terms like "dispersion", "entropy of the specification", or "budget of precision". The sanity experiment is useful, but it supports a narrower claim than the plan currently suggests: it shows that one or two model-plus-prompt instruments produce more diverse Go type outputs for looser inputs. It does not yet prove that `H_spec` is a stable, model-independent property of a specification, nor that the thresholds in the YAML gate are generally valid.

The article will be much harder to attack if it explicitly frames the metrics as **instrument-relative risk indicators** and separates "measurement theory" from "practical CI heuristic".

## 1. Strong points

### The core thesis is good

The article's strongest idea is that a specification consumed by agents is not a deterministic instruction but a distribution over possible implementations. This is a useful mental model for agentic engineering. It explains why "looks clear to a human reviewer" is not enough once the consumer is a stochastic system without the team's implicit context.

The "precision budget" analogy is also strong. It gives architects a familiar engineering pattern: define tolerance, measure against tolerance, and block work when the artifact falls outside the allowed range.

### The failure-mode taxonomy is valuable

Separating ambiguity, incompleteness, and untraceability is a major improvement over generic complaints about "bad requirements". These are genuinely different problems:

- Ambiguity: several plausible interpretations.
- Incompleteness: missing constraints, schemas, edge cases, or acceptance criteria.
- Untraceability: requirements not decomposed into downstream architectural or functional commitments.

This taxonomy makes the rest of the article more credible because it avoids pretending that one metric can catch every kind of specification failure.

### `K_drift` is the most defensible metric

The traceability metric is the cleanest of the three. If the graph is built from explicit markup or a validated parser output, then the graph traversal is deterministic and auditable. Senior readers will accept this more readily than the LLM-sampling metric.

The article should lean on `K_drift` as the "boring but production-ready" foundation.

### The plan already includes important measurement caveats

The section on baseline noise, temperature, and small-sample bias is a significant strength. Many articles in this space would stop at "sample ten outputs and compute entropy"; this plan correctly acknowledges that the model and sampling configuration are part of the measuring instrument.

The sanity README also documents real experimental failure modes: thinking mode consuming output budget, context truncation, invalid generations, and `num_predict` limits. Those details make the methodology feel grounded rather than purely rhetorical.

### The realization/exploration split is important

The distinction between implementation mode and exploration mode is technically sound. High output diversity is dangerous when the system needs a correct API contract or database schema. The same diversity can be useful during architectural search. This prevents the article from becoming an oversimplified "entropy bad" argument.

## 2. Weak spots and technical issues

### `H_spec` is not really "dispersion"

The formula

```text
H_spec = - sum_i P(x_i) log2 P(x_i)
```

is Shannon entropy over discrete output clusters. That is a diversity/uncertainty measure, not dispersion in the usual statistical sense. Calling it "dispersion" is rhetorically understandable, but a senior audience may object because dispersion usually suggests variance, standard deviation, pairwise distance, or spread in a metric space.

Recommended fix: call it **cluster entropy** or **interpretation entropy**. If the article wants a dispersion metric, report mean pairwise AST distance separately:

```text
D_pair = mean_{i<j}(1 - sim(x_i, x_j))
```

Then `H_cluster` answers "how many equivalence classes appeared?" and `D_pair` answers "how far apart were the generated artifacts?"

### The additive variance equation is likely too strong

The plan says:

```text
sigma^2_output = sigma^2_specification + sigma^2_model/sampling
```

This is a useful intuition, but mathematically it assumes separability, independence, and a variance-like continuous quantity. The observed output diversity is almost certainly an interaction:

```text
output_diversity = f(specification, model, prompt, temperature, decoder, output task, parser, clustering threshold)
```

The same vague spec may produce low diversity under a conservative prompt, and a precise spec may produce high diversity if the requested projection is underconstrained or the model varies naming conventions.

Recommended fix: replace the equation with a measurement-model statement:

```text
Observed diversity is instrument-relative:
M(spec; model, prompt, sampling, projection, parser, clustering).
The baseline estimates the instrument floor for a chosen configuration; it does not isolate a model-independent sigma_spec.
```

If the equation remains, mark it explicitly as a simplifying approximation, not a validated decomposition.

### `Delta H = H_spec - H_baseline` is not always safe

Subtracting a baseline is reasonable, but entropy estimates can be noisy and threshold-sensitive. With `N=10`, a baseline of 0.00 does not prove zero instrument noise; it only says that this particular run produced one cluster. A different prompt, version, or output task may not.

Also, if the baseline is "fully specified Go types" and the target task is "generate Go types", the baseline is too close to copying. It may understate instrument noise for less direct projections such as "specification to architecture", "requirements to DB schema", or "business rules to service boundary".

Recommended fix:

- Use several baselines: copy-like baseline, formal schema baseline, realistic precise requirement baseline.
- Report confidence intervals or bootstrap ranges for `H` and mean similarity.
- Avoid implying that `H_baseline = 0` is a universal noise floor.

### The sanity experiment supports the direction, not the full claim

The README results are directionally useful:

- `fog` has lower similarity and max entropy.
- real specs also produce high entropy.
- adding architecture/context to `real-ragivka` improves mean similarity from 0.278 to 0.393 on instrument B.

But the experiment is small and instrument-relative. It should be presented as a sanity check, not validation. Specific limitations:

- `N=10` is too small for robust entropy estimates.
- `temperature=1.0` is a stress test, while the plan recommends lower temperature for working measurement.
- Only two instruments were used, and both are LLM generators under a single projection task.
- The clustering threshold has a large effect (`sharp` on B is `H@0.95=3.32` but `H@0.80=2.12`).
- Single-linkage clustering can chain samples through intermediate similarities and create unstable clusters.
- Regenerating invalid Go until `N` valid samples may hide an important signal: ambiguous specs may increase invalid-output rate.

Recommended wording: "The sanity check shows the metric can detect large differences under a fixed instrument; it does not establish universal thresholds."

### The `sharp` result weakens the simple narrative

In the sanity results, `sharp` is much better than `fog` in mean similarity, but its entropy can still be high:

```text
sharp, instrument B: mean sim 0.682, H@0.95 = 3.32, H@0.80 = 2.12
```

This is not a failure, but it must be explained. A strict threshold of 0.95 may split outputs that are semantically equivalent but differ in naming, helper types, method placement, or enum representation. That means `H@0.95` may measure formatting/projection freedom as much as specification ambiguity.

Recommended fix: use the `sharp` result to teach the reader that threshold choice and equivalence relation are part of the instrument. Consider making `meanPairwiseSim` or `H@0.80` the primary sanity figure, with `H@0.95` as a stricter diagnostic.

### The AST feature vector is a lossy semantic proxy

The analyzer extracts features like type names, struct fields, method signatures, constants, and function declarations. That is a reasonable first pass, but it is not semantic equivalence.

Examples of false differences:

- `type Currency string` plus constants vs plain `string` with validation elsewhere.
- `PaymentProvider` enum vs `Provider` enum.
- `Receipt` returned directly vs wrapped in `PaymentResult`.
- `decimal.Decimal` vs `string` or `int64` cents, depending on libraries.

Examples of false similarities:

- Same field names but wrong units or constraints.
- Same function signatures but different implied idempotency behavior.
- Same table-like structs but missing state machine semantics.

Recommended fix: describe AST similarity as **structural similarity of generated Go interface artifacts**, not implementation correctness or full semantic similarity.

### `D_const` needs a better denominator and scale

The current formula is:

```text
D_const = typed links / unstructured text nodes
```

This has several issues:

- It can exceed 1, so "density" is not naturally bounded.
- If a document has zero unstructured text nodes, the denominator is zero.
- More prose can reduce the score even if the prose is useful rationale.
- More links can inflate the score even if links are low-value or mechanically duplicated.
- It mixes two different units: edges and nodes.

The plan already warns that raw density can punish good rationale, but the formula still invites objections.

Recommended alternatives:

```text
constraint_coverage = requirements_with_at_least_one_typed_constraint / total_requirements
trace_coverage = requirements_with_REQ_to_FUN_link / total_requirements
schema_coverage = domain_entities_with_schema / total_domain_entities
```

For density, normalize per requirement or per entity:

```text
constraints_per_requirement = typed_constraint_edges / total_requirements
```

This is easier to interpret in CI than "typed edges divided by text nodes".

### `K_drift` is good but underspecified around graph construction

The formula is defensible only after graph construction is reliable. If an LLM parser builds the RFLP graph, the metric is no longer free of LLM uncertainty. The plan says the graph can be built by a light AI parser or by markup/linter. But then it says the measurement contour has no LLM. That is only true in the markup/linter path.

Recommended fix: split two modes:

- Strict mode: explicit markup only; deterministic metric.
- Assisted mode: LLM parser proposes graph edges; metric is deterministic after extraction, but extraction quality must be validated.

Also define what counts as a valid outgoing edge. A low-quality `REQ -> FUN` link should not satisfy traceability if the function does not implement the requirement.

### Thresholds look arbitrary

Examples:

```yaml
delta_H max: 0.75
D_const min: 0.35
K_drift max: 0.2
```

The plan labels some thresholds as hypotheses, which is good. But the YAML example may still be read as prescriptive.

Recommended fix: present thresholds as placeholders and add a calibration procedure:

1. Collect historical specs.
2. Label downstream outcomes: rework, compile failures, schema churn, review defects, agent loop count, token waste.
3. Compute metrics before implementation.
4. Fit project-local thresholds against outcomes.
5. Recalibrate when model/prompt/orchestrator changes.

### Some causal claims are too dramatic for technical credibility

The article's style is intentionally forceful, but some claims are easy to attack:

- "context window fills with spaghetti and apologies"
- "telemetry.jsonl explodes with recursive errors"
- "token usage grows nonlinearly"
- "AI inevitably chooses `map[string]any`"
- "JSONB fully ruins relational integrity"

These are plausible failure stories, not universal consequences. Senior readers will tolerate rhetorical color, but only if the technical core is precise.

Recommended fix: frame the payment example as a representative failure mode observed in agent pipelines, not as a deterministic chain. Use "may", "often", "in one common failure mode", or include an actual trace if available.

### Formal methods are treated a bit unfairly

The plan correctly says formalization moves uncertainty into abstraction choice, model scope, and traceability maintenance. But the statement that formalism makes teams spend years learning notation instead of designing real systems is too broad.

Senior readers with TLA+, Alloy, Dafny, or model-checking experience may see this as a strawman. The better claim is:

"Formal methods reduce semantic ambiguity inside the formalized slice, but they do not solve coverage, abstraction, or maintenance by themselves."

That version is both stronger and harder to dismiss.

## 3. What to improve

### Reframe the metrics as instrument-relative

The most important improvement is terminology. Avoid implying that `H_spec` is an intrinsic property of the document. Use language like:

```text
H_cluster(spec | instrument)
```

where instrument includes:

- model and version;
- system prompt;
- output projection task;
- temperature and decoder settings;
- sample count;
- parser;
- similarity function;
- clustering algorithm and threshold.

This single change makes the methodology much more defensible.

### Rename and split the metrics

Suggested metric names:

- `H_cluster`: entropy over equivalence classes of generated artifacts.
- `D_pair`: mean pairwise structural distance.
- `invalid_rate`: fraction of generations that fail syntax/schema parsing before regeneration.
- `constraint_coverage`: percentage of requirements/entities with machine-readable constraints.
- `trace_coverage` or `K_drift`: percentage of requirements lacking valid downstream links.

This gives readers a dashboard with interpretable components rather than one overloaded "entropy" score.

### Treat invalid generations as signal

The README says invalid Go is regenerated until `N` valid samples are obtained. That is useful for computing AST similarity, but the discarded count should be a first-class metric:

```text
invalid_rate = discarded_generations / total_generation_attempts
```

A vague or oversized spec may increase invalid output, truncation, or parser failure. Hiding those failures can make the spec look cleaner than it is.

### Add uncertainty estimates

For `N=10`, report results as rough diagnostics. For serious gates:

- increase `N` where cost allows;
- bootstrap entropy and mean similarity;
- report min/median/max over repeated runs;
- compare thresholds against confidence intervals, not single values.

Even a small bootstrap table would make the article feel much more rigorous.

### Improve clustering

Single-linkage is simple but can produce chaining effects. Consider adding at least one alternative:

- complete-linkage for stricter clusters;
- average-linkage as a compromise;
- connected components at threshold, but explicitly call out chaining;
- medoid-based clustering if distance behaves well.

The article does not need a full clustering survey, but it should say that clustering choice is part of the measurement instrument.

### Calibrate against outcomes

The article should connect metrics to actual engineering pain:

- number of clarification rounds;
- compile/test failure rate;
- post-generation review defects;
- schema churn;
- agent loop count;
- token/cost overhead;
- manual rework time.

Without this, the metrics risk being elegant but unvalidated. With it, the article becomes much more compelling: "higher measured ambiguity predicts more downstream rework."

### Separate rhetorical essay from technical appendix

The Acts I-V structure is engaging, but the math needs a calmer section or appendix. Senior readers often scan for definitions, assumptions, and validation. Give them a compact technical block:

- Definitions.
- Instrument configuration.
- Estimators.
- Known biases.
- Calibration method.
- What the metric does and does not claim.

This lets the main article keep its voice while protecting the technical argument.

## 4. What to add

### A "claims ladder"

Add a small section that ranks claims by evidence strength:

```text
Strongly supported:
- Different specification inputs produce different output diversity under a fixed instrument.
- Explicit schemas and trace links reduce some degrees of freedom.

Supported by sanity check, not yet generalized:
- More complete real-world docs improve structural similarity.

Hypothesis requiring more data:
- Specific metric thresholds predict downstream failure.
- Delta-H can be used as a stable CI gate across projects.
```

This will make the article look honest and serious.

### A worked numeric example

Show a tiny entropy calculation:

```text
N=10 outputs -> clusters [6, 3, 1]
H = -(0.6 log2 0.6 + 0.3 log2 0.3 + 0.1 log2 0.1) = 1.295 bits
H/H_max = 1.295 / log2(10) = 0.39
```

Normalize by `H_max` for readability:

```text
H_norm = H / log2(N)
```

This makes scores comparable across different sample counts.

### A table of metric validity

Add a table like:

| Metric | Measures | Does not measure | Main failure mode |
| --- | --- | --- | --- |
| `K_drift` | explicit missing trace links | correctness of linked implementation | bad graph extraction |
| `constraint_coverage` | machine-readable constraint presence | quality/completeness of constraints | cargo-cult schemas |
| `H_cluster` | diversity of generated artifacts under an instrument | intrinsic ambiguity or correctness | model/prompt/threshold sensitivity |
| `D_pair` | structural spread of generated artifacts | semantic equivalence | shallow AST features |
| `invalid_rate` | generation/parsing fragility | ambiguity alone | output budget/context issues |

This table would preempt many objections.

### A better baseline suite

Use three baseline classes:

1. Copy baseline: exact Go types provided in the prompt.
2. Formal baseline: OpenAPI/JSON Schema/protobuf-style contract converted to Go.
3. Realistic precise baseline: natural-language requirements plus schemas, invariants, examples, and acceptance criteria.

Then compare `fog`, `sharp`, and real specs against all three.

### A counterexample section

Add one paragraph acknowledging cases where metrics can mislead:

- A verbose but precise spec may lower raw density.
- A minimal formal spec may omit critical business behavior.
- Two generated APIs may look structurally different but be equally valid.
- A model may produce consistent output from a vague spec because its pretraining prior dominates.

This does not weaken the article; it makes the claims harder to dismiss.

### A stronger definition of "precision budget"

Define the budget operationally:

```yaml
precision_budget:
  implementation_zone:
    H_norm_max: 0.25
    mean_pairwise_distance_max: 0.20
    invalid_rate_max: 0.05
    trace_coverage_min: 0.90
    constraint_coverage_min: 0.80
  exploration_zone:
    H_norm_min: 0.40
    sandbox_required: true
    no_direct_repo_write: true
```

The exact numbers should be labeled project-local, but the structure is good.

### A clearer link from metrics to remediation

The loop "measure -> find entropy sources -> clarify -> remeasure" is good. Make it more concrete:

- If clusters differ by entity fields, add schema fields or examples.
- If clusters differ by service boundaries, add component ownership and API contracts.
- If clusters differ by state transitions, add a state machine.
- If `K_drift` is high, add REQ-to-FUN links.
- If invalid rate is high, reduce output scope or split the spec.

This turns the metrics from gatekeeping into an authoring tool.

## Suggested revised technical framing

The most defensible version of the article's technical thesis is:

> A specification consumed by AI agents induces a distribution of generated artifacts under a chosen model/prompt/projection instrument. We can estimate the diversity of that distribution by repeated sampling, structural normalization, similarity measurement, and clustering. This estimate is not an intrinsic property of the document, but it is an operational risk indicator: under the same instrument, higher artifact diversity often signals underspecified implementation choices. Combined with deterministic traceability and constraint-coverage metrics, it can drive CI gates and clarification loops.

This keeps the original insight while removing the weakest mathematical claim.

## Bottom line

The plan is worth turning into an article. Its conceptual direction is strong, and the sanity experiment is a useful first empirical anchor. The main work before publication is to reduce overclaiming:

- rename `H_spec` to something instrument-relative;
- stop treating cluster entropy as direct specification variance;
- make `D_const` bounded or replace it with coverage metrics;
- explain the `sharp` result honestly;
- treat invalid generations and threshold sensitivity as first-class signals;
- present thresholds as project-calibrated hypotheses, not universal constants.

With those changes, the article can survive senior review. Without them, the likely criticism will be: "interesting metaphor, but the math measures the model/prompt setup more than the spec." The article already has enough material to answer that criticism; it just needs to make that answer explicit.
