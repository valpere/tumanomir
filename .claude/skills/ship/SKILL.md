---
name: ship
description: "tumanomir ship pipeline: issue → implement → review → merge → close. Usage: /ship [--yes|-y] [issue-number | title]"
---

# Skill: /ship
# Issue → Merged PR Pipeline

```
/ship                    → auto-select next open issue (asks confirmation)
/ship --yes              → auto-select and proceed, skip UI prompts
/ship -y                 → same as --yes
/ship {issue-id}         → ship a specific issue by number
/ship {issue title}      → find by title, then ship
/ship --yes {issue-id}   → skip UI prompts (technical ambiguities always ask)

  → resolve issue        (fetch title + description)
  → analyze ambiguities  (scan description + codebase → ask decisions before coding)
  → mark in-progress
  → code-generator       (implement → tests → review agents → simplify → docs → PR)
  → /fix-review          (multi-model rounds + Claude Arbiter — does NOT merge)
  → merge PR             (/ship owns the merge)
  → post comment         (summary: what changed, PR link)
  → close issue
  → log to /self-learn (if installed)
  → final report
```

One command. Pauses only for genuine technical decisions before coding.

---

## STEP 0: Resolve Issue

**Flag detection:** Check for `--yes` / `-y` anywhere in arguments. Set `YES_MODE=true`, strip flag.

**Argument classification** (after stripping flags):
- No argument → auto-select (see below)
- Numeric or `#N` → `gh issue view {N} --repo valpere/tumanomir --json number,title,body,url`
- Otherwise → title search

**YES_MODE:**
- Skips "Proceed?" confirmation prompts and title disambiguation
- **Does NOT skip STEP 0.5** — technical decisions always require input

### If no argument — auto-select

```bash
gh issue list --repo valpere/tumanomir --assignee @me --state open \
  --json number,title,url,milestone,labels --limit 10
```

Prefer issues in the active milestone. Within milestone, prefer `priority:high` / `urgent` labels
(neither exists as a distinct label in this repo yet — fall back to issue order / `enhancement` vs `bug`).

If `YES_MODE=false`, present top candidate and wait:
```
Suggested: #{number} — {title}
Milestone: {milestone} | {url}
Proceed? [Y / pick another / cancel]
```

If `YES_MODE=true`, print selection and proceed immediately.

### If a title string was given

```bash
gh issue list --repo valpere/tumanomir --state open --search "{title}" \
  --json number,title,url --limit 10
```

One close match → use directly. Multiple → present up to 5. None → stop.

### Validation

Extract `ISSUE_NUMBER`, `ISSUE_TITLE`, `ISSUE_BODY`, `ISSUE_URL`.

If issue is not open: stop — "Issue #{N} is already closed."

Print:
```
Shipping: #{ISSUE_NUMBER} {ISSUE_TITLE}
{ISSUE_URL}
```

Mark in-progress:
```bash
gh issue edit {ISSUE_NUMBER} --repo valpere/tumanomir --add-label "in-progress"
```

(`in-progress` and `in-review` are custom labels created for this pipeline —
`gh label create` was run once at install time; if missing, `gh issue edit`
will error — recreate them: `gh label create in-progress --repo valpere/tumanomir --color FBCA04` /
`gh label create in-review --repo valpere/tumanomir --color 0E8A16`.)

---

## STEP 0.5: Analyze Task & Resolve Ambiguities

Before touching any code, scan the issue body and codebase for decisions that must be made
first. **Runs even in YES_MODE — technical decisions are not UI prompts.**

Scan for:
- "Decision:", "Options:", "or (b)", "discuss before starting", "if/or" branches
- Two implementation strategies side-by-side
- Scope conditionals: "if not already fixed in #{N}" → verify in code
- Dependency on another issue's output
- Constants or limits where the value isn't specified
- Whether the task touches a methodological invariant (`CLAUDE.md`
  §"Методологічні інваріанти") — if so, `docs/requirements.md` must be updated
  first; this is a technical decision, always surface it even in YES_MODE

For each ambiguity:
1. Search codebase for 1–3 concrete options grounded in existing patterns
2. Classify: multiple options → present with tradeoffs; dependency → verify in code; no options → ask directly
3. Collect all decisions in one pass before proceeding

**Format:**
```
Before I start coding, I need to resolve {N} decision(s):

1. {Ambiguity title}
   Context: {one sentence}
   A) {option} — {tradeoff}
   B) {option} — {tradeoff}
```

Zero ambiguities: print `"No ambiguities — proceeding."` and continue.

---

## STEP 1: Launch Code Generator

```
Agent(code-generator):
  "Implement: {ISSUE_TITLE}

   Issue body:
   {ISSUE_BODY}

   Issue URL: {ISSUE_URL}

   Implementation pipeline:
   1. Git branch: <type>-<slug> off main (this repo's convention — no issue-number prefix)
   2. Implement the feature/fix
   3. Run: golangci-lint run && go build ./... && go vet ./... && go test ./...
   4. static-analysis review (tech-lead handles architecture separately)
   5. Apply review findings
   6. code-simplifier → docs-maintainer
   7. Create PR (include '{ISSUE_URL}' in body + a 'Rationale' section:
      key decisions stated in WHY/PURPOSE terms per code-generator's Comment
      Discipline — flag explicitly if either condition is missing)
   8. Return PR number + URL"
```

Wait for completion. On error or no PR: print error, stop.

Extract `PR_NUMBER`, `PR_URL`, `BRANCH_NAME`.

Mark in-review:
```bash
gh issue edit {ISSUE_NUMBER} --repo valpere/tumanomir \
  --add-label "in-review" --remove-label "in-progress"
```

---

## STEP 1.5: Ponytail Review (cheap pre-check)

Run `/ponytail-review` on the PR diff before the expensive multi-model
`/fix-review` round. One-line findings only (over-engineering, dead
code, reinvented stdlib) — does not apply fixes.

If findings exist: fix them directly (small, obvious cuts), then
proceed to STEP 2. If none: proceed directly.

---

## STEP 2: Run /fix-review

### Docs-only check

```bash
gh pr diff {PR_NUMBER} --repo valpere/tumanomir --name-only
```

If every changed file matches `*.md`, `*.txt`, `docs/**`, `.github/**/*.md` →
skip /fix-review, go to STEP 3. Print:
```
Skipping /fix-review — PR is docs-only. Code Review Pyramid has nothing to evaluate.
```

Mixed PRs (any code file present): /fix-review runs. When in doubt, run it.

### Standard

```
/fix-review {PR_NUMBER}
```

Multi-model rounds + Claude Arbiter. Per this project's `.claude/skills/fix-review/config.yaml`
(`auto_merge: false`), it **never merges** — STEP 3 here owns the merge.

Collect summary (fixed / skipped / open). If unresolved items remain, note them and continue.

---

## STEP 3: Merge PR

```bash
gh pr merge {PR_NUMBER} --repo valpere/tumanomir --squash --delete-branch
```

If checks pending: `gh pr merge {PR_NUMBER} --repo valpere/tumanomir --auto --squash --delete-branch`
then poll every 30s (max 30 min).

If conflicts: stop — "PR #{PR_NUMBER} has merge conflicts. Resolve and re-run."

**Timeout:** 30 min. If still open: ask user to merge manually, then re-run `/ship {ISSUE_NUMBER}`
to complete the tracker update.

Verify:
```bash
gh pr view {PR_NUMBER} --repo valpere/tumanomir --json state,mergedAt --jq '{state,mergedAt}'
```

---

## STEP 4: Post Completion Comment

```bash
gh issue comment {ISSUE_NUMBER} --repo valpere/tumanomir --body "$(cat <<'EOF'
## Shipped — PR #{PR_NUMBER}

**Summary:** {one-paragraph description of what changed and why}

**Key changes:**
- {file or component}: {what changed and why}

**Tests:** {what was added or verified}

**PR:** {PR_URL}
EOF
)"
```

If the call fails: warn and continue.

---

## STEP 5: Close Issue

```bash
gh issue close {ISSUE_NUMBER} --repo valpere/tumanomir \
  --comment "Shipped in PR #{PR_NUMBER}."
```

On failure: warn, ask user to close {ISSUE_URL} manually.

---

## STEP 6: Log to /self-learn (if installed)

Skip silently if `.claude/skills/self-learn/` doesn't exist in this project
(воно є в tumanomir, тож цей крок реально спрацює).

Log one entry via `/self-learn log`, classified by what actually happened
this run:
- Smooth run, zero `/fix-review` escalations → **win**
- `/fix-review` caught something that should have been prevented earlier
  (wrong pattern, missed edge case, convention violation) → **mistake**
- STEP 0.5 ambiguity analysis surfaced a decision that would otherwise have
  produced a wrong implementation → **win**

If the log call fails, don't block the pipeline on it — note it in the
final report's Warnings and move on.

---

## STEP 7: Final Report

```
## /ship complete — #{ISSUE_NUMBER} {ISSUE_TITLE}

Issue:   {ISSUE_URL}  (closed)
PR:      {PR_URL}  (merged, branch deleted)

/fix-review: {fixed} fixed · {skipped} skipped · {open} still open

Pipeline:
✓ Issue resolved + ambiguities cleared
✓ Code generated (code-generator)
✓ golangci-lint run && go build ./... && go vet ./... && go test ./... passed
✓ static-analysis
✓ code-simplifier + docs-maintainer
✓ PR created + /fix-review (multi-model + Arbiter)
✓ PR merged (squash)
✓ Completion comment posted
✓ Issue closed
✓ Logged to /self-learn
```

Append **Warnings** for any skipped items or non-fatal failures.

---

## RULES

1. Issue must be ready/approved before shipping — not auto-verified.
2. One PR per issue — reuse existing branch PR if it exists.
3. Never force-push — rebase if branch diverged from main.
4. **`/ship` owns the merge (STEP 3), not `/fix-review`.**
5. All review agents run inside code-generator — do not re-launch in `/ship`.
6. Tracker updates are best-effort — surface failures, never silently skip.
7. 30-minute merge timeout — stop and report. Never loop indefinitely.
8. Docs-only PRs skip /fix-review — the Pyramid has nothing to evaluate.
9. Any change touching a methodological invariant (`CLAUDE.md`) must have
   `docs/requirements.md` updated first — tech-lead enforces this at plan
   review, but `/ship`'s STEP 0.5 should catch it earlier if possible.
10. `/self-learn` logging is best-effort — never block completion on it.
