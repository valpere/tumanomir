# Requirements: tumanomir

Specification-precision measurement tool for AI-driven software projects.
Productization of the "Source of the Unknown" methodology (see
`context/history.md` for provenance).

This document is written in tumanomir's own traceable markup
(`[REQ-*] -> [FUN-*]`, `@schema`) — the tool must be able to measure its
own specification (dogfooding).

---

## 1. Data model

@schema Thresholds {
  k_drift_max: Float @constraint(default: 0.20, range: [0,1]),
  d_const_min: Float @constraint(default: 0.35, range: [0,1]),
  d_pair_max:  Float @constraint(default: 0.30, range: [0,1])
}

@schema InstrumentConfig {
  backend: Enum["ollama"],
  model: String,
  temperature: Float @constraint(default: 1.0),
  samples: Int @constraint(default: 10, min: 2),
  think: Bool @constraint(default: false),
  num_ctx: Int @constraint(rule: "must exceed prompt token count"),
  num_predict: Int @constraint(rule: "must exceed natural output length")
}

@schema Report {
  k_drift: KDriftResult?,
  d_const: DConstResult?,
  dispersion: DispersionResult?,
  verdict: Enum["ok","warn","block"],
  exit_code: Int @constraint(in: [0,1,2])
}

---

## 2. Functional requirements

### 2.1 Deterministic layer (`check` command)

1. [REQ-CHK-01] The tool must compute K_drift (fraction of requirements
   lacking a trace edge) over markdown specs using explicit markup only:
   a requirement is a `[REQ-*]` tag; a trace edge is `-> [FUN-*]`,
   `-> [LOG-*]` or `-> [PHY-*]` appearing before the next `[REQ-*]` tag.
   -> [FUN-CHK-01] metrics.KDrift(doc []byte) KDriftResult

2. [REQ-CHK-02] K_drift output must list the identifiers of hanging
   requirements, not only the ratio — the metric must be actionable.
   -> [FUN-CHK-02] KDriftResult.HangingIDs []string

3. [REQ-CHK-03] The tool must compute D_const (lexical constraint density)
   as markers/(markers+prose tokens), where markers are `@schema`,
   `@constraint`, `[REQ-`, `-> [FUN-`, `-> [LOG-`, `-> [PHY-`.
   -> [FUN-CHK-03] metrics.DConst(doc []byte) DConstResult

4. [REQ-CHK-04] `check` must accept a single file or a directory
   (recursively, `*.md` only) and aggregate per-file results.
   -> [FUN-CHK-04] spec.Load(path string) ([]Spec, error)

5. [REQ-CHK-05] The deterministic layer must not invoke any LLM or
   network call; it must be usable as a git pre-commit hook.
   -> [LOG-CHK-05] packages internal/metrics, internal/spec have no
      network imports (enforced by test)

### 2.2 Stochastic layer (`measure` command)

6. [REQ-MSR-01] The tool must measure D_pair = 1 − mean pairwise
   structural AST similarity over N generated Go artifacts from one spec.
   -> [FUN-MSR-01] dispersion.Analyze(sources [][]byte, simThreshold float64) DispersionResult

7. [REQ-MSR-02] Cluster entropy H (Shannon, over single-linkage clusters
   at a configurable similarity threshold) and its normalized form
   H_norm = H/log2(N) must be reported as ordinal signals, never as the
   primary gate metric.
   -> [FUN-MSR-02] DispersionResult{H, HNorm, Clusters, SimThresh}

8. [REQ-MSR-03] Generation must go through a pluggable instrument
   interface; v0.1 ships one backend: Ollama chat API.
   -> [FUN-MSR-03] instrument.Generator interface; instrument.Ollama

9. [REQ-MSR-04] The full instrument configuration (backend, model,
   temperature, N, think mode, num_ctx, num_predict, sim threshold)
   must be fixed per run and printed in the report — measurements are
   instrument-relative and meaningless without it.
   -> [FUN-MSR-04] InstrumentConfig serialized into Report header

10. [REQ-MSR-05] Generations that fail Go parsing must be retried up to a
    bounded limit and the discard count must be reported as invalid rate;
    hiding invalid generations is forbidden.
    -> [FUN-MSR-05] measure loop: retry ≤ 2 per sample; DispersionResult.Discarded

11. [REQ-MSR-06] For reasoning-capable models the instrument must set
    think=false; requests must set num_ctx and num_predict explicitly.
    Silent truncation of the input spec is a measurement-integrity bug.
    -> [FUN-MSR-06] instrument.Ollama request builder; prompt-size check
       against num_ctx before the run

### 2.3 Output and gating

12. [REQ-OUT-01] Human-readable TTY output: one line per metric with
    value, verdict (ok/warn/block) and the threshold it was judged
    against.
    -> [FUN-OUT-01] report.Render(w io.Writer, r Report)

13. [REQ-OUT-02] Exit codes: 0 = all gates pass, 1 = at least one gate
    failed, 2 = execution error. CI-composable by construction.
    -> [FUN-OUT-02] Report.exit_code

14. [REQ-CFG-01] Thresholds are overridable via CLI flags; defaults are
    the article's hypothesis values (0.20 / 0.35 / 0.30) and must be
    documented as uncalibrated starting points.
    -> [FUN-CFG-01] internal.DefaultThresholds(); flag wiring in cmd

---

## 3. Non-functional requirements

15. [REQ-NFR-01] `check` on a 1 MB spec corpus must complete in under
    100 ms (single pass, no allocations proportional to marker count).
    -> [PHY-NFR-01] benchmark in internal/metrics

16. [REQ-NFR-02] Single static binary, Go ≥ 1.26, stdlib-only for v0.1
    (no CLI frameworks, no YAML deps until the gate command exists).
    -> [PHY-NFR-02] go.mod with zero external requires

17. [REQ-NFR-03] Methodology invariants must not be silently changed:
    D_pair is the working metric, H is ordinal; thresholds are
    hypotheses; instrument config is part of every result. Changes here
    require updating this document first.
    -> [LOG-NFR-03] CLAUDE.md "Methodology invariants" section

---

## 4. Out of scope for v0.1 (roadmap)

- `.tumanomir.yaml` config and `gate` command (CI mode)
- baseline calibration (`calibrate` command), bootstrap CIs
- graph-based D_const (RFLP/Neo4j), assisted K_drift (LLM parser)
- non-Ollama instruments, non-Go projections (SQL DDL, OpenAPI)
