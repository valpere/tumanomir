You are doing a **dreaming pass** for the **tumanomir** project — async,
scheduled curation of project context. This is sleep-time consolidation:
review what accumulated since last pass, identify patterns, suggest
curation. Read-only — output a report, don't modify anything.

## Project context

- **tumanomir** — specification-precision measurement tool for
  AI-driven projects (Go CLI, stdlib-only except `gopkg.in/yaml.v3`).
  Productization of the methodology from the article "Source of the
  Unknown".
- **The specification is primary**: `docs/requirements.md` (tumanomir's
  own markup — `[REQ-*] -> [FUN-*]`, `@schema`) is checked against code,
  not the other way around — a mismatch is a bug, in the code or in the
  requirements.
- Workflow: `/backlog` (plan → tech-lead approval → GitHub issue) →
  `/ship` (issue → implement → `/fix-review` → merge → close). Branches
  `<type>-<slug>` off `main`; **never commit directly to main**.
- Source of truth: `CLAUDE.md` (English, actively maintained;
  `CLAUDE.uk.md` is a historical pre-translation snapshot, may drift).

## Targets

| Path | What to look for |
|------|-------------------|
| `docs/requirements.md` | Spec vs. code drift — a `[REQ-*]`/`[FUN-*]` with no matching implementation, or code behavior with no matching requirement |
| `docs/architecture.md` | Stale description of metrics/CLI UX/package layout vs. current `internal/` structure |
| `docs/roadmap.md` | Items that already shipped (check against `git log`/closed issues) but weren't removed; items that should move to GitHub issues instead of lingering here |
| `docs/investigation/history.md` | Only touch if genuinely stale — this is provenance, rarely needs updates |
| `CLAUDE.md` "Methodological invariants" section | Any recent commit that silently changed D_pair/H_norm behavior, instrument-relative reporting, or invalid-rate handling without a requirements-first update — these are **do-not-change-silently** per the file's own header |
| `.claude/plans/` | Plan files older than 14 days — stuck or forgotten, should have become a GitHub issue via `/backlog` already |
| `.claude/skills/` | Skills with overlapping responsibility (e.g. `find-bugs` vs `review-deps` vs `fix-review` — do their descriptions still carve out distinct territory?) |
| `.claude/agents/` | Agent prompts that may be stale or duplicated |
| Recent PR review comments | Recurring `/fix-review` themes across merged PRs — candidates for a new `CLAUDE.md` convention or a `self-learn` pattern |
| `git log --since="30 days ago"` | Recurring failure patterns, oft-reverted commits, commits that touch `internal/dispersion/` (the core AST-similarity/clustering engine) without touching `docs/requirements.md` |

## What to find

### 1. Spec/code/architecture drift

For each `[REQ-*]`/`[FUN-*]` entry in `docs/requirements.md`, spot-check
whether the corresponding code in `internal/` still matches. Flag:
- A requirement with no matching implementation (dead spec).
- Code behavior (especially in `internal/dispersion/`, the CLI flags,
  or `cmd/`) with no matching requirement (undocumented behavior).
- `docs/architecture.md` describing a package layout or CLI surface
  that no longer matches `internal/*/` on disk.

### 2. Methodological invariant violations

Search recent commits (`git log --since="30 days ago" -p -- internal/`)
for anything touching: default thresholds (0.20/0.35/0.30), the
D_pair/H_norm computation, `think`/`num_ctx`/`num_predict` Ollama
settings, or invalid-rate handling — and check whether
`docs/requirements.md` or `CLAUDE.md` was updated in the same commit.
A silent change here is exactly what the file's own header warns
against ("do not change silently; requirements first").

### 3. Recurring `/fix-review` themes

`gh pr list --state merged --limit 20` + `gh pr view N --json comments`
on a sample of recent PRs. A theme repeating across 3+ PRs is a
candidate for: a new `CLAUDE.md` convention, a `.claude/skills/`
addition, or a `self-learn` pattern entry (check
`.claude/skills/self-learn/_patterns/` if it exists).

### 4. Stale plans

`.claude/plans/*.md` older than 14 days (check file mtime and/or first
commit date) that never became a GitHub issue via `/backlog`. These are
either stuck or forgotten — flag for cleanup or resumption.

### 5. Skill/agent overlap or drift

Read `.claude/skills/*/SKILL.md` frontmatter descriptions — do any two
skills claim overlapping territory in a way that would confuse which
one to invoke (`find-bugs` / `review-deps` / `fix-review` /
`comprehension-gate` / `doubt-driven-development` all touch
"is this code correct/safe" from different angles — check they still
read as genuinely distinct, not redundant)?

### 6. Housekeeping/self-learn health

If `.claude/skills/housekeeping/` or `.claude/skills/self-learn/` have
their own state files (patterns, check history), note if they look
stale or contradictory — don't run the skills themselves, just look.

## Report format

```markdown
# tumanomir dreaming — <ISO week>

## TL;DR
<3-5 bullet summary>

## 1. Spec/code/architecture drift
### a) <finding>
- Confidence: high|medium|low
- Evidence: <file:line, commit sha, requirement id>
- Suggest: <action>

## 2. Methodological invariant checks
...

## 3. Recurring /fix-review themes
...

## 4. Stale plans
...

## 5. Skill/agent overlap
...

## 6. Open questions
<what you couldn't verify from a read-only pass>
```

Confidence levels: **high** = directly verified against a specific
file/commit; **medium** = pattern observed but not exhaustively
checked; **low** = a hunch worth someone's attention, not a confirmed
finding. Don't fabricate evidence — say "couldn't verify" rather than
guess.
