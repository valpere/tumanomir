---
name: find-bugs
description: "tumanomir (valpere/tumanomir, Go stdlib-only): find bugs, security vulnerabilities, and code quality issues in branch changes. Report-only — no code changes. Usage: /find-bugs [topic]"
---

# Skill: /find-bugs
# Code Review — Bug & Vulnerability Hunt — tumanomir

Report only. Do not make changes.

---

## Phase 1 — Input Gathering

1. Get the full diff: `git diff $(git merge-base main HEAD)...HEAD`
2. If truncated, read each changed file individually until every changed line is seen.
3. List all modified files before proceeding. Do not skip any.

---

## Phase 2 — Attack Surface Mapping

For each changed file, identify and list:

- All **user-controlled inputs** (URL params, request bodies, query strings, form fields)
- All **external calls** — are errors checked? are timeouts set? are responses closed/consumed?
- All **state mutations** — shared state modified without locks? mutation visible to other goroutines/threads?
- All **file/path operations** — paths constructed from user data?
- All **resource allocations** — unbounded loops or allocations on user-supplied sizes?
- All **silent failures** — errors swallowed, empty fallbacks, missing nil/null checks?

---

## Phase 3 — Security Checklist

Check every item against every changed file:

- [ ] **Injection** — user input reaching SQL queries, shell commands, file paths, template strings?
- [ ] **Hardcoded secrets** — API keys, passwords, tokens in changed code?
- [ ] **Auth bypass** — can authentication or authorization checks be skipped?
- [ ] **Information disclosure** — error messages returning stack traces, internal paths, or sensitive data?
- [ ] **Path traversal** — file paths constructed from user-supplied strings without sanitization?
- [ ] **Request body limits** — is unbounded user-supplied data size possible?
- [ ] **Missing null/nil checks** — pointer dereference or null access on values that could be absent?
- [ ] **Race conditions** — shared mutable state accessed from concurrent paths?
- [ ] **Dependency confusion** — any new unreviewed packages introduced?

### Go Checklist

- [ ] Goroutine leaks — all goroutines bounded by context or timeout?
- [ ] Nil interface vs nil pointer — returning interface wrapping nil concrete?
- [ ] Context propagation — request contexts passed to all blocking operations?
- [ ] Mutex scope — mutex unlocked in the same scope it was locked?
- [ ] Response body close — deferred after nil check?

### tumanomir-Specific Checklist

- [ ] **Network isolation (REQ-CHK-05)** — does the diff add a network-capable import
  (`net/http`, or a transitive dependency that reaches the network) to
  `internal/metrics/` or `internal/spec/`? These two packages must stay
  zero-network by architectural invariant; `internal/nonetwork_test.go`
  runtime-checks this, but a new import can still slip past casual review.
  `internal/instrument/` is the *only* package allowed to touch the network.
- [ ] **Ollama measurement-integrity flags** — does the diff touch
  `think`/`num_ctx`/`num_predict` handling in `internal/instrument`? These
  are measurement-integrity requirements (REQ-MSR-06), not defaults to
  "simplify" — e.g. silent prompt truncation from an under-sized `num_ctx`
  is a correctness bug, not a performance tweak.
- [ ] **Threshold/invariant drift** — does the diff change
  `internal.DefaultThresholds()` (0.20 / 0.35 / 0.30), or the D_pair-vs-H_norm
  gating split (only D_pair and K_drift gate exit code; D_const and H_norm
  are advisory), without a corresponding `docs/requirements.md` update? Per
  `CLAUDE.md`, these are hypotheses from the article, not free-standing
  constants.
- [ ] **Discard/invalid-rate visibility** — does a change to the retry/discard
  loop in `internal/instrument` or the `measure` command risk hiding the
  invalid-generation rate rather than reporting it (REQ-MSR-05)? The
  methodology's core claim depends on invalid rate being visible, never
  swallowed into a silently-smaller sample.

> No React/TS, Node/Express surface in this repo (Go stdlib-only, v0.1) —
> those generic checklists are omitted.

---

## Phase 4 — Verification

For each potential issue:
- Check if it is already guarded elsewhere in the changed code.
- Read at least 10 lines of surrounding context to confirm the issue is real.
- Only report issues you can substantiate with evidence from the diff.

---

## Phase 5 — Pre-Conclusion Audit

Before finalizing:
1. List every file reviewed — confirm each was read completely.
2. List every checklist item: issue found, or confirmed clean.
3. List anything you could NOT fully verify and why.

---

## Output Format

**Priority:** security vulnerabilities > correctness bugs > performance > code quality

**Skip:** style, formatting, naming preferences

For each issue:

```
**File:Line** — brief description
Severity  : Critical / High / Medium / Low
Problem   : what's wrong
Evidence  : why this is real (no existing guard, language semantics confirm it)
Fix       : concrete suggestion
```

If nothing significant: say so — don't invent issues.
