---
name: backlog
description: "Plan a task for tumanomir before touching code — read codebase, produce plan, get tech-lead approval, create a GitHub issue in valpere/tumanomir, delete plan file. Usage: /backlog [task description | issue number]"
---

# Skill: /backlog
# Plan-First Development

Prevent wasted effort by aligning on approach before writing code.
For trivial changes (one-line fix, typo) skip this and just do it.
For anything touching more than one file or requiring design decisions — plan first.

The plan file is temporary. It is deleted once a task is created in the issue tracker.
The issue is the canonical record.

---

## Step 0 — No argument: show queued plans

List draft plan files from `.claude/plans/` (files without an issue yet), sorted by
priority prefix, excluding `README.md`.

```
Queued drafts:

  1. [p2] 2-health-endpoints.md — Add /health/live and /health/ready
  2. [p3] 3-chairman-prompt.md — Pass consensus score to chairman prompt

Or type a new task description:
```

Wait for selection, then proceed.

---

## Step 1 — Understand the task

**If argument is a number:** check `.claude/plans/` for a matching prefix, or fetch
an existing issue with `gh issue view <n> --repo valpere/tumanomir`.

**If argument is a description:** check `.claude/plans/` for an existing matching plan.
If found, use it. If not, create a new one.

Read `CLAUDE.md`, `docs/requirements.md` (the spec is primary — dogfooded traceable
markup, `[REQ-*] -> [FUN/LOG/PHY-*]`), and `docs/investigation/history.md`
(provenance and decisions already made — everything a reviewer needs lives in
the repo; `../context/` is no longer referenced).

**Methodological invariants** (`CLAUDE.md` §"Методологічні інваріанти") are not up
for silent revision in a plan — if a task touches one (D_pair vs H roles,
instrument-relative reporting, invalid-rate visibility, threshold defaults,
zero-network deterministic layer), the plan must update `docs/requirements.md`
first and say so explicitly.

---

## Step 2 — Read affected files

Identify every file that will change. Read them — do not guess.

File candidates by component:

| Component | Path |
|---|---|
| CLI / commands | `cmd/tumanomir/main.go` |
| Shared types (Report, Verdict, Thresholds) | `internal/types.go` |
| Spec loading | `internal/spec/spec.go` |
| K_drift / D_const (deterministic) | `internal/metrics/kdrift.go`, `internal/metrics/dconst.go` |
| Dispersion / D_pair / entropy (stochastic) | `internal/dispersion/{dispersion,astfeat,cluster}.go` |
| Instrument (Ollama backend) | `internal/instrument/` |
| Report rendering — currently inline in `main.go` | `internal/report/` (roadmap; REQ-OUT-01) |
| Spec + requirements docs | `docs/requirements.md` |
| Architecture + roadmap | `docs/architecture.md`, `docs/roadmap.md` |
| Design notes (provenance) | `docs/investigation/history.md` |
| Source article + external review reports | `docs/investigation/SourceOfTheUnknown.md`, `docs/investigation/reports/` |
| Dispersion sanity/reference corpus | `docs/investigation/_sanity/` (`_`-prefixed — ignored by `go build`/`go vet`) |

---

## Step 3 — Determine metadata

| Field | Values |
|-------|--------|
| `type` | `bug` / `feature` / `task` / `test` |
| `priority` | `p0`–`p3` (impact + urgency) |
| `debt` | `quick-fix` / `balanced` / `proper-refactor` |
| `effort` | `xs` / `s` / `m` / `l` / `xl` |
| `component` | which modules/packages are touched |
| `labels` | closest existing GitHub label: `bug`→`bug`, `feature`→`enhancement`, `task`/`test`→ no exact match, omit or use `enhancement` |

Repo has only the GitHub default label set (`bug`, `documentation`, `duplicate`,
`enhancement`, `good first issue`, `help wanted`, `invalid`, `question`,
`wontfix`) — no custom `task`/`test` labels exist yet.

---

## Step 4 — Write (or update) the plan file

Save to `.claude/plans/{N}-{slug}.md`. No `.claude/plans/README.md` exists in this
project — use the standard structure: Summary, Acceptance Criteria, Implementation,
Not in Scope, Commit Message, After Implementing checklist.

This file is **temporary** — deleted after the issue is created.

---

## Step 5 — Tech Lead review

Launch the `tech-lead` agent with the plan content. Await verdict:

- **APPROVED** → proceed
- **APPROVED WITH CHANGES** → update plan file, proceed
- **REJECTED** → revise plan file, re-submit

Do not start implementation until Tech Lead approves and user confirms.

---

## Step 6 — Create issue in tracker

Check for duplicates first:
```bash
gh issue list --repo valpere/tumanomir --state open --search "<title keywords>"
```

Create using only Summary and Acceptance Criteria (implementation details stay internal):

```bash
gh issue create --repo valpere/tumanomir \
  --title "<type>(<component>): <title>" \
  --label "<labels>" \
  --body "$(cat <<'EOF'
## Summary
<summary>

## Acceptance Criteria
<criteria>
EOF
)"
```

---

## Step 7 — Delete the plan file

```bash
rm .claude/plans/{N}-{slug}.md
```

The issue is now the canonical record.

---

## Step 8 — Report and stop

Report the created issue URL and confirm plan deletion. Stop.
Implementation is triggered separately via `/ship`.

---

## Output format

```
## Plan: <task name>

**Scope:** <one sentence>
**Type:** bug | feature | task | test
**Priority:** p1: high
**Effort:** s (1–2 hours)

**Files to change:**
- `internal/metrics/kdrift.go` — what changes

**Approach:**
1. ...

**Risks:**
- ...

**Not in scope:**
- ...

---
Issue: #<number> — <url>
Plan file deleted.
```
