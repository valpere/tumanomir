# CLAUDE.md — tumanomir

> **Language note:** this file is English for accessibility to non-Ukrainian
> contributors, external reviewers, and AI tooling — see issue #21. It does
> **not** change how sessions on this project are conducted: Val
> communicates in Ukrainian, per his standing convention for personal
> projects; only this document's own text is English. This file (English)
> is now the actively maintained, auto-loaded version — future edits to
> conventions/invariants land here first. [`CLAUDE.uk.md`](CLAUDE.uk.md) is
> the pre-translation Ukrainian snapshot, kept for reference; it may drift
> if not updated alongside future changes here.

Specification-precision measurement tool for AI-driven projects (CLI, Go).
Productization of the methodology from the article "Source of the Unknown".

## Where to start in a new session

1. Read `docs/requirements.md` — **the specification is primary**; code is
   written to match it, not the other way around. It's written in
   tumanomir's own markup (`[REQ-*] -> [FUN-*]`, `@schema`) — dogfooding.
   This applies to how *this implementation's* development process works:
   here, the specification is the controlling artifact code is checked
   against. The methodology itself, by contrast, treats specifications as
   uncertain measurement targets — the very thing tumanomir measures the
   precision of — not as a source of truth. The apparent contradiction is
   illusory: the first statement is about development discipline, the
   second is about the subject of measurement.
2. `docs/architecture.md` — the current architecture (metrics, CLI UX,
   package layout, methodological invariants). `docs/roadmap.md` — what's
   not built yet, ordered by horizon; tactical debt lives in GitHub issues,
   not there. `docs/investigation/history.md` — the project's provenance:
   where the methodology came from, which decisions were already made and
   why. Everything a reviewer or external agent needs lives in the
   repository (`/home/val/wrk/projects/tumanomir/tumanomir`) — `../context/`
   is no longer referenced; the review surface is limited to the repo.
3. Current code state — a spike of the deterministic core plus a port of
   the dispersion experiment; check it against requirements — a mismatch
   is a bug, either in the code or in the requirements (update requirements
   first).

## Methodological invariants (do not change silently; requirements first)

- **D_pair** (1 − mean pairwise AST sim) — the working metric; **H_norm**
  (= H / log₂N, normalized cluster entropy) is an ordinal signal only
  ("one cluster or many"), and it's the one actually reported/gated on.
  Raw **H** (bits) is computed internally but saturates at log₂N for small
  N and so is not comparable across different N by itself.
- All stochastic measurements are **instrument-relative**: the instrument
  configuration (model+version, prompt, temp, N, think, num_ctx,
  clustering threshold) is fixed and printed in every report.
- **Invalid rate** is reported, never hidden (retry with a discard
  counter).
- Default thresholds (0.20/0.35/0.30) are **hypotheses** from the article,
  not constants — state this explicitly in usage text.
- The deterministic layer (`check`) — zero network, zero LLM. Git-hook-ready.
- Ollama: `think: false` for reasoning models; `num_ctx` must fit the
  prompt (silent truncation = a measurement-integrity bug); `num_predict`
  above the natural output length.

## Build and verify

The binary is built into `bin/` via `make`, not `go build`/`go run` directly.

```bash
make build     # -> bin/tumanomir
make vet
make test
make dogfood   # bin/tumanomir check docs/requirements.md — dogfood smoke test
make lint      # golangci-lint run (requires golangci-lint installed)
make ci        # build + vet + test + lint + dogfood, all together
```

## Conventions

- Go >= 1.26, stdlib-only in v0.1 (no CLI frameworks, no YAML dependencies).
- Types: shared ones in `internal/types.go`; package-specific in
  `internal/<pkg>/` (higher-level types take priority on conflicts).
- Code/comments/messages — English; session communication — Ukrainian
  (see the language note at the top of this file).
- Branches: `<type>-<slug>` off main; never commit directly to main.
- Reference data for dispersion tests: generated files from the article's
  experiment — `docs/investigation/_sanity/out*/` (120 files, now in-repo;
  the `_` prefix keeps `go build ./...`/`go vet ./...` from treating them
  as Go packages). Real reference numbers are in
  `docs/investigation/_sanity/README.md`. That's the full archival corpus;
  `internal/dispersion/testdata/` is the actual fixture location for
  future `internal/dispersion` tests (Go's `testdata/` convention,
  auto-excluded from the build): a "sharp" subset from both instruments,
  with its own README and reference numbers.
  The original article is `docs/investigation/SourceOfTheUnknown.md`; the
  11 external review reports are in `docs/investigation/reports/`.
