# tumanomir — external documentation audit: docs/requirements.md (kilo)

> Model: kilo/nvidia/nemotron-3-super-120b-a12b:free. Full-codebase context (agent explored the repo
> directly via its own tools, not embedded doc text). Part of a
> 5-agent x 3-document audit round, 2026-07-04. Read-only: agent was
> explicitly instructed not to edit/create/delete any repo file;
> confirmed clean afterward via git status.

---

## What's good
- The document uses the tool's own traceable markup (`[REQ-*] -> [FUN-*]`, `@schema`) for self-verification (dogfooding).
- Clear separation of deterministic (check) and stochastic (measure) layers with traceable function stubs (e.g., `-> [FUN-CHK-01] metrics.KDrift`).
- Explicit schemas for thresholds, instrument configuration, and report structure enforce instrument-relative measurements.
- Correctly identifies methodological invariants: D_pair as the working metric, H_norm as ordinal/advisory, thresholds as uncalibrated hypotheses.
- Mandates deterministic layer is network- and LLM-free (git-hook ready) via `internal/nonetwork_test.go`.
- Requires invalid generation rate reporting and prohibits hiding it (REQ-MSR-05).
- Includes non-functional constraints: subprocess 100ms latency, stdlib-only, single binary.
- Clearly marks v0.1 out-of-scope items (e.g., `gate` command, calibration).

## What's bad
- **D_const miscalculation**: `internal/metrics/dconst.go` counts markers correctly but then uses `bytes.Fields(doc)` for prose tokens, which includes the marker text itself. This double-counts markers in the denominator, systematically underestimating lexical constraint density.
- **Incomplete D_pair feature vector**: `internal/dispersion/astfeat.go` only extracts `const` declarations, omitting `var` declarations despite the requirement explicitly stating "top-level const/value names" and citing the file as the authoritative implementation.
- **Missing think enforcement**: While `think` defaults to `false`, the tool does not prevent users from enabling `think=true` for reasoning models, risking silent prompt truncation. The requirement states the instrument "must set think=false" for such models.
- **Output format incomplete for ordinal metrics**: REQ-OUT-01 requires "one line per metric with value, verdict (ok/warn/block) and the threshold it was judged against." However, `H` and `H_norm` output only shows the value and a note, omitting verdict and threshold fields.
- **Missing performance benchmark**: REQ-NFR-01 references a benchmark in `internal/metrics`, but no `Benchmark` function exists to verify the 100ms claim for 1MB spec corpus.

## What it doesn't cover
### Relative to SourceOfTheUnknown.md
- Omits the methodological distinction between "zone of realization" (where dispersion is gated) and "zone of exploration" (where dispersion is cultivated). This is an orchestrator-layer concept (out of scope for v0.1) but could be noted as future work.
- Does not explicitly articulate that `H_norm`'is ordinal-only because raw H saturates at log₂N (though this is covered in CLAUDE.md).

### Relative to the codebase
- No missing features: all documented aspects have corresponding code (though some are incorrect/incomplete as noted above).
- The document aligns with implemented behavior except for the specific inaccuracies listed in "What's bad".
- The article's preference for D_pair as a continuous, non-saturating metric over entropy is reflected in the focus on D_pair as the working metric.

## Verdict
The document requires revision before it can serve as the controlling specification. The D_const calculation error and incomplete D_pair feature vector directly compromise measurement correctness, while the missing think enforcement and output format deviations risk misuse and non-compliance with stated requirements. Addressing these issues is essential for alignment with both the source article and the actual implementation.
