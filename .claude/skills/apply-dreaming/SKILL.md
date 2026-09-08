---
name: apply-dreaming
description: "Read the latest tumanomir dreaming report and apply
  high-confidence findings. Routes spec/architecture/code changes
  through the project's standard /backlog → tech-lead approval →
  GitHub issue → /ship workflow. Direct branch+PR only for tooling
  scripts that don't change methodology or requirements. Annotates
  report with [applied YYYY-MM-DD] markers."
user-invocable: true
argument-hint: "[week|latest]"
---

# /apply-dreaming (tumanomir)

Walks each dreaming-report finding interactively and leaves an audit
trail (`[applied YYYY-MM-DD]` / `[planned …]` / `[skipped …]` markers
appended to the report).

## When to invoke

- Monday morning after the Sunday cron-run produced a fresh report.
- Or any time after a manual `.claude/dreaming/dreaming.sh` run.
- Or `/apply-dreaming` (with optional `latest` or `2026-W##` argument).

## Inputs

- Optional argument: `latest` (default) or `YYYY-W##`.
- Project root: `~/wrk/projects/tumanomir/tumanomir/`.

## Steps

### 1. Locate report

```bash
WEEK="${1:-latest}"
DIR=".claude/dreaming/reports"
if [[ "$WEEK" == "latest" ]]; then
  # find + mtime-sort, not ls -1t: handles special characters robustly,
  # and the digit-class year prefix doesn't hardcode "2026" (an ISO-week
  # filename like 2026-W01.md rolls into 2027-W## eventually — a literal
  # year prefix would silently stop matching future reports). Same fix
  # as curate-minions/SKILL.md's report selector.
  REPORT=$(find "$DIR" -maxdepth 1 -type f -name '[0-9][0-9][0-9][0-9]-W*.md' \
           -printf '%T@ %p\n' 2>/dev/null \
           | sort -rn | head -1 | cut -d' ' -f2-)
else
  REPORT="$DIR/$WEEK.md"
fi
```

### 2. Parse into structured items

Read REPORT. For each numbered sub-item under a section, extract:

- `id`, `title`, `confidence` (high/medium/low)
- `evidence` — file:line / commit sha / `[REQ-*]`/`[FUN-*]` id
- `suggestion`
- `category` — infer from the suggestion:
  - "update `docs/requirements.md`" / "update `docs/architecture.md`" /
    "update `docs/roadmap.md`" → `update-spec`
  - "code change / fix drift / implement" → `code-change`
  - "resume or delete plan" → `stale-plan`
  - "new `CLAUDE.md` convention / new skill / merge skills" → `update-tooling`
  - "self-learn pattern" → `update-memory`
  - else → `other`

**Confidence inheritance.** If a sub-item has no explicit `confidence`
field, inherit it from the enclosing section. Default `medium` only
when neither declares one.

**Idempotency — skip already-marked items.** If the next non-blank
line after an item starts with `> [applied …]`, `> [planned …]`,
`> [skipped …]`, or `> [manual-review-required …]`, skip it silently.
The report is appended to (never rewritten) on each pass. Print a
summary at the start: `2026-W##: M new items (N already-processed
skipped)`.

### 3. Show TL;DR + counts

```
2026-W##: N items (X high, Y medium, Z low)
TL;DR: ...
Process all? [y/select/skip-low/abort]
```

### 4. Triage walk

Iterate `high → medium → low`. For each item:

```
[H 1/N] §<id>  <title>
  Evidence: <file:line / commit / REQ-id>
  Suggestion: <suggestion>

  [a]pply / [s]kip / [v]erify-first / [e]vidence / [q]uit
```

For `low`: skip silently unless the user opted in at step 3.

### 5. Apply per category

**Routing rule.** Two paths:

- **Plan-and-gate path** for anything touching the methodology, the
  spec, or code behavior: `update-spec`, `code-change`,
  `update-tooling` (new `CLAUDE.md` convention or skill). These all go
  through `/backlog` (draft → tech-lead approval → GitHub issue) →
  `/ship`. `docs/requirements.md`/`docs/architecture.md` and the
  "Methodological invariants" section of `CLAUDE.md` are explicitly
  **do-not-change-silently** per the file's own header — never edit
  them directly from a dreaming finding, even a high-confidence one.
- **Direct branch+PR path** only for non-methodology tooling:
  `stale-plan` cleanup and `.claude/dreaming/*.sh` script fixes that
  don't touch prompts or requirements. Still needs its own branch + PR
  — **main has no direct-commit convention here** (`git commit` to
  `main` is never allowed, per `CLAUDE.md`'s Conventions section) —
  but skips the `/backlog` planning step since there's no design
  decision to gate.
- `update-memory` (self-learn pattern files, if git-tracked here —
  check `git check-ignore` first, `.claude/skills/self-learn/_patterns/`
  and `_knowledge-base/` are gitignored in this repo) is a direct edit,
  no branch needed, since it's local/gitignored state.

#### `update-spec` / `code-change` / `update-tooling` — plan-and-gate

1. Draft a plan referencing the dreaming finding, following `/backlog`'s
   own conventions (`.claude/plans/<priority>-dreaming-W##-<slug>.md` —
   note `.claude/plans/` is gitignored here, so this is scratch space,
   safe to draft freely).
2. Plan body: cite report §<id>, evidence (file:line / REQ-id / commit
   sha), suggested change, files to touch, acceptance criteria.
3. Tell the user: "Created plan `<priority>-dreaming-W##-<slug>`. Run
   `/backlog <slug>` to gate it through tech-lead approval and create
   the GitHub issue. Then `/ship` for implementation."
4. Don't implement directly — `/backlog` → tech-lead approval → issue
   → `/ship` is the only path for anything spec/methodology/code.

#### `stale-plan` — direct cleanup (no PR, gitignored scratch)

1. Read the plan file's content and mtime.
2. If it already has a matching GitHub issue (`gh issue list --search
   <slug>`), delete the plan file — it's done its job.
3. If it has no issue and looks abandoned, ask the user: resume via
   `/backlog <slug>` or delete.
4. `.claude/plans/` is gitignored — no commit needed either way.

#### `update-tooling` (dreaming script only, not prompts) — branch+PR

For fixes to `dreaming.sh` itself (not `dreaming-prompt.md`, which
shapes what the pass looks for and counts as methodology-adjacent —
route that through plan-and-gate instead):

1. `git switch -c fix-dreaming-w##-<slug>` off `main`.
2. Edit the script.
3. Smoke-test: `bash -n .claude/dreaming/dreaming.sh`.
4. Commit, push, open a PR. Merge after CI/review per the project's
   normal PR convention (same as any other tumanomir PR — no special
   exception for tooling scripts).

#### `other` — manual review

Print suggestion + evidence. Don't apply. Annotate
`[manual-review-required 2026-MM-DD]`.

### 6. Annotate report

After each applied item, append (never rewrite the original):

```markdown
> [applied 2026-MM-DD: <action>; commit <sha>; PR <num>]
```

For created plans:
```markdown
> [planned 2026-MM-DD: .claude/plans/<file>; awaiting /backlog]
```

For skipped:
```markdown
> [skipped 2026-MM-DD: <reason>]
```

### 7. Final summary

```
Applied: N (direct plan cleanup / tooling PRs)
Plans created: M (awaiting /backlog → tech-lead → /ship)
Manual review: K
Skipped: P

PRs opened: <list>
Plans pending: <list>

Next steps:
1. Run /backlog on each plan to gate through tech-lead approval.
2. Run /ship on approved plans — /fix-review runs automatically as part of /ship.
```

## Constraints (CRITICAL)

- **NEVER commit or push directly to `main`** — this project's
  Conventions section is explicit: "never commit directly to main."
  Every change, tooling included, goes through a branch (+ PR where
  the change is git-tracked).
- **NEVER edit `docs/requirements.md`, `docs/architecture.md`, or
  `CLAUDE.md`'s Methodological invariants directly** — always
  plan-and-gate through `/backlog` → tech-lead approval, even for a
  high-confidence finding. These are explicitly "do not change
  silently; requirements first."
- **NEVER auto-apply low confidence** without explicit request.
- **ALWAYS cite report-section** (`§<id>`) in commit messages and PR
  bodies.
- **One PR per category-batch** — don't mix a `dreaming.sh` tooling
  fix with a spec/methodology change in the same PR.
- **Confirm before destructive ops** (deleting a stale plan) even at
  high confidence.

## Anti-patterns

- ❌ Edit `docs/requirements.md`/`docs/architecture.md` directly from a
  dreaming finding — always through `/backlog`.
- ❌ Commit or push to `main` directly, for tooling or anything else.
- ❌ Mix a `dreaming.sh` fix with a methodology-adjacent change in one PR.
- ❌ Modify the report's original suggestions (annotate only).
- ❌ Apply a finding without verifying it against current code first —
  the dreaming pass is a heuristic read, it can be wrong.

## Companion skills

- `/backlog` — plan → tech-lead approval → GitHub issue.
- `/ship` — issue → implement → `/fix-review` → merge → close.
- `/fix-review` — parallel multi-model review + Claude Arbiter (part of `/ship`).
- `/housekeeping` — synchronous repo-health snapshot, complementary to dreaming.

## See also

- `.claude/dreaming/dreaming-prompt.md` — what the dreaming pass looks for.
- `.claude/dreaming/dreaming.sh` — how the pass is run (systemd timer).
- `CLAUDE.md` — "Methodological invariants" section (target of many
  `update-spec` suggestions; never edit without `/backlog`).
