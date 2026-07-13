---
name: debug
description: "Systematic bug diagnosis — reproduce, isolate, hypothesize, verify, fix. Usage: /debug [what's broken]"
---

# Skill: /debug
# Systematic Bug Diagnosis

---

## PROTOCOL

Bugs have two parts: the **symptom** (what you observe) and the **cause** (what's actually wrong). The protocol moves from symptom → cause → fix, one verified step at a time.

### Phase 1 — Reproduce

**Goal:** Reliable, minimal reproduction before touching any code.

1. State the symptom precisely: what input, what output, what was expected.
2. Find the smallest input that triggers it.
3. Verify you can reproduce it consistently.
4. Is this a regression? `git log --oneline -20` — when did it last work?

**If you can't reproduce it:** Stop. Gather more information. Unreproducible bugs are unsolvable.

### Phase 2 — Isolate

**Goal:** Narrow the search space to the smallest possible area.

1. Identify the layers involved: UI → API → service → DB → external?
2. Test each boundary — where does correct input produce wrong output?
3. Binary-search the call stack: disable half, does the bug disappear?
4. Is it environment-specific? dev vs prod? one machine vs all?

**Isolation heuristics:**
- Works in tests, breaks live → check environment (config, secrets, timing)
- Started after a deploy → `git bisect`
- Intermittent → look for shared mutable state, races, time-dependent logic

### Phase 3 — Hypothesize

State one falsifiable hypothesis before changing any code:

```
Hypothesis  : [what I think is wrong]
Evidence for: [what supports this]
Against     : [what doesn't fit]
Test        : [one action that confirms or refutes]
```

**One hypothesis at a time.** Testing multiple simultaneously makes it impossible to know what fixed it.

### Phase 4 — Verify

1. Run the test that confirms or refutes.
2. **Confirmed** → move to fix.
3. **Refuted** → form a new hypothesis. Don't modify the failing thing yet.
4. Keep a short log: what you tried, what you learned.

### Phase 5 — Fix

1. Make the minimal change that fixes the root cause — not the symptom.
2. Run the full test suite.
3. Add a regression test that would have caught this bug.
4. Commit message explains *why*, not just *what*:
   ```
   fix: [what was broken]

   Root cause: [why it happened]
   Fix: [what changed and why this is correct]
   ```

---

## COMMON BUG PATTERNS

| Pattern | Symptom | Where to look |
|---------|---------|---------------|
| **Off-by-one** | Wrong last/first element | Loop bounds, slice indices, pagination |
| **Nil/null dereference** | Crash on access | Unguarded pointer use, missing null checks |
| **Race condition** | Intermittent, timing-dependent | Shared state, goroutines/threads, async code |
| **Wrong input assumptions** | Works in tests, breaks in prod | Input validation, edge cases, empty/max values |
| **Config/env mismatch** | Works locally, breaks in CI/prod | `.env` files, env var names, defaults |
| **Stale state** | Shows old data | Caches, memoization, DB transaction isolation |
| **Type coercion** | Wrong math, unexpected falsy | JS `==`, int/float truncation, string/int mixing |
| **Dependency version** | Broke after upgrade | Changelog, breaking changes, peer dependencies |

---

## REGRESSION TEST FORMAT

```
// Regression: [short description of what was broken]
// Arrange: exact conditions that triggered the bug
// Act:     the action that was broken
// Assert:  the correct outcome
```

The test must reproduce the **exact** failing scenario, not a simplified analogue.

---

## PROJECT QUICK REFERENCE

### Test commands

Run all  : `go build ./... && go vet ./... && go test ./...`
Run one  : `go test ./... -run TestName` (e.g. `go test ./internal/metrics/... -run TestKDriftHanging`)
Coverage : `go test ./... -cover` (not wired into CI yet — run manually)

### Debug logging

No `LOG`/`DEBUG`/`VERBOSE` env vars exist in the codebase yet — it's all pure
functions over `[]byte` (metrics, dispersion) with no logging today. For ad-hoc
tracing, add `fmt.Fprintln(os.Stderr, ...)` at the call site and remove before
commit; there's no logging framework to wire into.

For the not-yet-built `internal/instrument` (Ollama) layer, expect to need
request/response dumps — per `docs/investigation/history.md` §"Граблі Ollama", the
failure modes are silent (empty content from thinking-mode truncation, no
error), so add explicit verbose logging around the HTTP call from day one
rather than debugging blind later.

### Known fragile areas

Project is early (2 commits, v0.1 scaffold) — no TODO/FIXME markers, no
historically fragile files yet (every `.go` file touched exactly once, in
the initial commit). The known *future* fragile area, called out explicitly
in `docs/investigation/history.md` and `docs/requirements.md` (REQ-MSR-06):

- **`internal/instrument` (not yet implemented)** — Ollama integration.
  Known failure modes from the article's experiment, already baked into
  requirements: `think:true` on reasoning models silently empties `content`
  (`num_predict` consumed by hidden thinking tokens); default `num_ctx=4096`
  silently truncates large spec prompts; `num_predict` below natural output
  length truncates generations into a false "separate cluster" signal. All
  three are silent — no error surfaces — so verify request params against
  `num_ctx`/`num_predict` explicitly before trusting a `measure` run's output.

### Stack-specific debug notes

- `go vet ./...` and `golangci-lint run` (installed, no `.golangci.yml` —
  runs with default linter set) both catch real issues here; run both, not
  just `go test`.
- `internal/dispersion` parses generated Go source at runtime
  (`go/parser`) — a parse failure there is *expected signal* (invalid
  generation, counted in `Discarded`), not a bug. Don't "fix" a
  `ValidGo() == false` case without checking whether the input was actually
  malformed Go first.
- Reference data for dispersion regression tests (not yet written) lives
  in-repo: `docs/investigation/_sanity/out*/` (120 generated files, 2
  instruments — see `docs/investigation/_sanity/README.md` for the expected
  reference numbers). `_`-prefixed so `go build`/`go vet` ignore it.

---

## RULES

- **Reproduce before touching code.** A fix without reproduction is a guess.
- **One change at a time.** Multiple simultaneous changes make causation unknowable.
- **Fix the root cause, not the symptom.** Wrapping a bug in an `if` is not a fix.
- **Always add a regression test.** If it broke once, it can break again.
- **`git bisect` for regressions.** Faster than reading the diff.
- Tag unverified claims `[hypothesis]` when communicating status.
