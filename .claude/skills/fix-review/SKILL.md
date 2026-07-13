---
name: fix-review
description: "tumanomir (valpere/tumanomir) multi-model PR review with parallel fan-out. Provider: Ollama cloud (glm-5.1, deepseek-v4-flash, kimi-k2.6) primary, Ollama local failover. Claude acts as Arbiter — confirms, escalates, or dismisses findings using vote count as confidence signal. Applies a single consolidated fix commit. Does NOT merge — /ship owns the merge. Usage: /fix-review [PR-number]"
---

# Skill: /fix-review (parallel)
# tumanomir — Automated PR Review

---

## OVERVIEW

Three model rounds run **in parallel**, then a single Claude arbiter
round adjudicates and applies the consolidated fix.

```
        ┌─→ Round 1 (glm-5.1)             ┐
diff ───┼─→ Round 2 (deepseek-v4-flash)    ┼─→ aggregate (dedupe + vote count)
        └─→ Round 3 (kimi-k2.6)           ┘
                                    ↓
                              Arbiter (Claude) — rules + ONE fix → commit → push
```

**Why parallel:**
- Wall time is `max(t₁, t₂, t₃)` instead of `t₁ + t₂ + t₃`.
- Three independent perspectives on the *same* diff (no caching effect from prior fixes).
- Vote count (1/3, 2/3, 3/3) is a strong confidence signal for the arbiter.
- One fix commit instead of four → cleaner PR history.

**Tradeoff vs. sequential:** no cascading feedback (R2 doesn't see what R1 fixed).
The arbiter handles dedup; in practice most "R2 reacted to R1's fix" cases were
just the second model lagging.

**Merge:** this project runs `/ship`, which owns the merge (its STEP 3). `/fix-review`
here **never merges** — see STEP 10.

---

## RUN COMPLETION CONTRACT (do not skip)

A run is **not** considered complete until **both** of these have happened
in this order, after Step 10 returns:

1. **Step 9 telemetry** — `telemetry.jsonl` has been appended with one
   row per model round + one arbiter row.
2. **Step 11 final summary** — printed to the user.

Step 10 never merges in this project (see below) — it only computes
`GATES_OK` and reports readiness. Control MUST flow into Step 11 and print
the summary. Do not stop after Step 10 — the human reads the summary, not
the silence.

---

## STEP 0: Resolve PR + Load Config

```bash
# PR number — argument wins, else current branch's open PR
PR_NUMBER="${1:-$(gh pr view --repo valpere/tumanomir --json number --jq '.number' 2>/dev/null)}"
[ -z "$PR_NUMBER" ] && { echo "No PR found. Pass /fix-review <number>"; exit 1; }

BASE_BRANCH=$(gh pr view "$PR_NUMBER" --repo valpere/tumanomir --json baseRefName --jq '.baseRefName')
CONFIG=".claude/skills/fix-review/config.yaml"
```

Read fields from `$CONFIG` via `yq` (or grep/awk fallback):
- `provider` — `ollama` (the only supported value; anything else aborts)
- `reviewers.ollama.round_{1,2,3}.model` — primary models (cloud)
- `reviewers.ollama_local.round_{1,2,3}.model` — local failover tier
- `ollama_api_url`
- `api_key_env` — not set in this project's config; defaults to `OLLAMA_API_KEY`
- `post_summary_to_pr`, `telemetry_enabled`, `pricing.{model}.{input,output}`

**Validate provider + load API key:**

```bash
source .claude/skills/lib/env.sh
source .claude/skills/lib/rest.sh

if [ "$PROVIDER" != "ollama" ]; then
  echo "ERROR: unsupported provider '$PROVIDER' in config.yaml — only 'ollama' is supported." >&2
  exit 1
fi

API_KEY_ENV="OLLAMA_API_KEY"
load_env_key "$API_KEY_ENV"   # reads .env.local → .env → shell env; this project keeps it in .env
API_KEY="${!API_KEY_ENV}"

if [ -z "$API_KEY" ]; then
  echo "ERROR: OLLAMA_API_KEY not found. Set it in .env for Ollama cloud." >&2
  exit 1
fi
```

**Provider probe + failover:**

After loading the key, probe the primary provider with a minimal call before
running the full review. If the probe fails (401, 402, "Insufficient credits",
"User not found", network error), fall back to Ollama local automatically and
**always print a visible notice to the user**:

```bash
ollama_error() {
  printf '%s' "$1" | jq -r '
    (.error | if type == "string" then .
               elif type == "object" then (.message // .code // "")
               else "" end)' 2>/dev/null
}

probe_provider() {
  local url="$1" key="$2" model="$3"
  local probe_payload probe_resp
  probe_payload=$(jq -n --arg m "$model" \
    '{model:$m,messages:[{role:"user",content:"OK"}],stream:false,max_tokens:3}')
  probe_resp=$(REST_TIMEOUT=10 rest_post "$url" "$probe_payload" "$key") || probe_resp=""
  if [ -z "$probe_resp" ] || ! printf '%s' "$probe_resp" | jq -e . >/dev/null 2>&1; then
    echo "PROBE_ERROR: empty or non-JSON response"; return 1
  fi
  local err; err=$(ollama_error "$probe_resp")
  [ -n "$err" ] && { echo "PROBE_ERROR: $err"; return 1; }
  return 0
}

ACTIVE_PROVIDER="ollama"
ACTIVE_API_URL="$API_URL"
ACTIVE_KEY="$API_KEY"
ACTIVE_MODELS=("$MODEL_R1" "$MODEL_R2" "$MODEL_R3")   # glm-5.1:cloud, deepseek-v4-flash:cloud, kimi-k2.6:cloud
FAILOVER_TIER=""
FAILOVER_REASON=""

read_ollama_models() {
  local tier="$1"
  M1=$(yq -r ".reviewers.${tier}.round_1.model // \"\"" "$CONFIG" 2>/dev/null)
  M2=$(yq -r ".reviewers.${tier}.round_2.model // \"\"" "$CONFIG" 2>/dev/null)
  M3=$(yq -r ".reviewers.${tier}.round_3.model // \"\"" "$CONFIG" 2>/dev/null)
}

read_ollama_models "ollama_local"
LOCAL_TIER_EXISTS="no"
[ -n "$M1" ] && [ -n "$M2" ] && [ -n "$M3" ] && LOCAL_TIER_EXISTS="yes"   # yes: granite3.3:8b, qwen2.5-coder:7b, gemma4:31b

if ! probe_provider "$API_URL" "$API_KEY" "$MODEL_R2" 2>&1; then
  if [ "$LOCAL_TIER_EXISTS" = "yes" ]; then
    echo "⚠️  FAILOVER: Ollama cloud unavailable — using Ollama local"
    ACTIVE_KEY=""; FAILOVER_TIER="ollama_local"; FAILOVER_REASON="cloud probe failed"
    ACTIVE_MODELS=("$M1" "$M2" "$M3")
    echo "⚠️  FAILOVER: using Ollama local (${ACTIVE_MODELS[*]})"
  else
    echo "⚠️  WARNING: Ollama cloud unavailable and no ollama_local tier configured — reviews will fail."
    FAILOVER_TIER="cloud_unavailable"; FAILOVER_REASON="cloud probe failed, no failover tier"
  fi
fi
```

Use `$ACTIVE_PROVIDER`, `$ACTIVE_API_URL`, `$ACTIVE_KEY`, and `$ACTIVE_MODELS`
everywhere from this point forward — **including the Step 3 `run_round` calls**.

**Ollama execution mode:** run the three rounds **sequentially** (not in
parallel) and raise the per-call timeout to 300s — local and cloud LLMs can't
reliably handle three concurrent large-model calls without timing out.

**Reasoning-model empty content:** Ollama's native `/api/chat` (not the
OpenAI-compatible endpoint — this project never uses that one) takes `think`
as a top-level request field, not inside `options`. The shared
`chat_payload_system` helper doesn't set it (documented no-op for Ollama), so
reasoning-capable cloud models (`deepseek-v4-flash`, `kimi-k2.6`) can burn
their entire token budget on hidden thinking and return empty content on
large diffs — confirmed on PR #88 and #89. Step 3 builds the payload directly
with `think:false` and a capped `num_predict` for every round by default
(see `ollama_review_payload` below) instead of relying on retries after the
fact.

---

## STEP 1: Detect Project Context

```
TEST_CMD="go build ./... && go vet ./... && go test ./..."
LINT_CMD="golangci-lint run"
```

Project context, in priority order (first that exists):
1. `CLAUDE.md` — first ~150 lines, focus on "Методологічні інваріанти" section
2. `README.md` — ambient context

```bash
PROJECT_CONTEXT=$(head -150 CLAUDE.md)
TELEMETRY_FILE=".claude/skills/fix-review/telemetry.jsonl"
```

Known review agents in this project:
```
# Security  : none
# Simplifier: code-simplifier
# Architect : tech-lead
```

---

## STEP 2: Build the Review Prompt

Single generic prompt, with `$PROJECT_CONTEXT` injected so each model can
apply project-specific rules.

```
You are a senior code reviewer. Review the following git diff using the
Code Review Pyramid — evaluate from bottom to top, spending the most
attention on lower layers and least on higher ones:

  5 (top)  — Code style        → DO NOT FLAG. Formatters / linters handle this.
  4        — Tests             → Are critical paths covered? New branches tested?
  3        — Documentation     → Is complex logic explained? Public APIs documented?
  2        — Implementation    → Bugs, logic errors, null/nil deref, error handling,
                                 resource leaks, race conditions, security holes,
                                 missing context propagation, performance pitfalls.
  1 (base) — API / Architecture → Layer violations, contract drift, banned patterns,
                                  state-machine violations, hidden coupling.

== Project context ==
{PROJECT_CONTEXT}
== End project context ==

Apply the project rules above to layer 1 and layer 2 findings — they are
load-bearing for this codebase. In particular: internal/metrics and
internal/spec must never gain a network import (REQ-CHK-05); the default
thresholds (0.20/0.35/0.30) and D_pair-vs-H roles are fixed invariants, not
implementation details to "improve".

Return ONLY a JSON array — no prose, no markdown fences, just raw JSON.
Each item must have exactly these fields:
  "file"     — relative file path (string)
  "line"     — line number on the + side of the diff (integer)
  "layer"    — pyramid layer 1–4 (integer)
  "severity" — "error" | "warning" | "suggestion" (string)
  "body"     — clear description of the issue and how to fix it (string)

Severity guide:
  error      — must fix before merge (bug, security hole, layer-1 violation)
  warning    — should fix (missing test for critical path, undocumented public API)
  suggestion — nice to have

Do NOT flag: formatting, blank lines, import order (layer 5 — automated).
Do NOT flag code not present in this diff.
Do NOT propose architectural rewrites — focus on what the diff actually changes.

If there are no issues, return an empty array: []

Git diff:
---
{DIFF}
---
```

Substitute:
- `{PROJECT_CONTEXT}` — value of `$PROJECT_CONTEXT`
- `{DIFF}` — output of `gh pr diff ${PR_NUMBER} --repo valpere/tumanomir`

---

## STEP 3: Fan Out — Three Models in Parallel

```bash
DIFF=$(gh pr diff "${PR_NUMBER}" --repo valpere/tumanomir)
PROMPT=$(printf '%s' "$PROMPT_TEMPLATE" | sed "s|{PROJECT_CONTEXT}|$PROJECT_CONTEXT|" | sed "s|{DIFF}|$DIFF|")

RUN_DIR=$(mktemp -d -t fix-review-XXXX)
START_MS=$(python3 -c "import time;print(int(time.time()*1000))" 2>/dev/null || echo $(($(date +%s) * 1000)))

REVIEW_SYSTEM_MSG="You are a senior code reviewer. Your entire response MUST be a raw JSON array — nothing else. Start with [ and end with ]. No prose, no markdown fences, no explanations before or after. Report at most 8 findings, most severe first; each body under 30 words. Do not include informational or 'no action needed' entries — only real issues. If there are no issues output exactly: []"
export REVIEW_SYSTEM_MSG

# Builds an Ollama /api/chat payload with think:false + a capped num_predict
# — see "Reasoning-model empty content" above. num_predict defaults to 4000;
# pass a smaller value (e.g. 2000) on the empty-content retry below.
ollama_review_payload() {
  local model="$1" sys="$2" prompt="$3" num_predict="${4:-4000}"
  jq -n --arg model "$model" --arg sys "$sys" --arg prompt "$prompt" --argjson np "$num_predict" \
    '{model:$model, messages:[{role:"system",content:$sys},{role:"user",content:$prompt}], stream:false, think:false, options:{num_predict:$np}}'
}

run_round() {
  local n="$1" model="$2"
  local r_start r_end
  r_start=$(python3 -c "import time;print(int(time.time()*1000))" 2>/dev/null || echo $(($(date +%s) * 1000)))
  local payload response active_provider active_url active_key
  active_provider="$ACTIVE_PROVIDER"
  active_url="$ACTIVE_API_URL"
  active_key="$ACTIVE_KEY"
  payload=$(ollama_review_payload "$model" "$REVIEW_SYSTEM_MSG" "$PROMPT" 4000)
  response=$(rest_post_ollama "$active_url" "$payload" "$active_key") || response='{"error":"call failed"}'

  local err; err=$(ollama_error "$response")

  # Success but empty/whitespace content — the model burned its budget
  # before answering (err is empty here, so this isn't the API-error path
  # below). Retry once with a tighter cap before falling through to local
  # failover.
  if [ -z "$err" ]; then
    local content; content=$(chat_content "$active_provider" "$response")
    if [ -z "$(printf '%s' "$content" | tr -d '[:space:]')" ]; then
      echo "warn: round ${n} (${model}) returned empty content — retrying once with a tighter cap" >&2
      payload=$(ollama_review_payload "$model" "$REVIEW_SYSTEM_MSG" "$PROMPT" 2000)
      response=$(rest_post_ollama "$active_url" "$payload" "$active_key") || response='{"error":"call failed"}'
      err=$(ollama_error "$response")
    fi
  fi

  if [ -n "$err" ] && [ "$FAILOVER_TIER" = "" ] && [ "$LOCAL_TIER_EXISTS" = "yes" ]; then
    local local_model
    local_model=$(yq -r ".reviewers.ollama_local.round_${n}.model // \"\"" "$CONFIG" 2>/dev/null)
    echo "warn: round ${n} error (${err}) — trying Ollama local (${local_model})" >&2
    active_provider="ollama"
    active_key=""
    payload=$(ollama_review_payload "$local_model" "$REVIEW_SYSTEM_MSG" "$PROMPT" 4000)
    response=$(rest_post_ollama "$active_url" "$payload" "$active_key") || response='{"error":"ollama local failover failed"}'
    model="$local_model"
    printf '%s' "$local_model" > "$RUN_DIR/round_${n}.failover"
  fi

  r_end=$(python3 -c "import time;print(int(time.time()*1000))" 2>/dev/null || echo $(($(date +%s) * 1000)))
  printf '%s' "$response"   > "$RUN_DIR/round_${n}.raw.json"
  printf '%s\n%s' "$model"  "$((r_end - r_start))" > "$RUN_DIR/round_${n}.meta"
}

# Run sequentially — see STEP 0 "Ollama execution mode".
run_round 1 "${ACTIVE_MODELS[0]}"
run_round 2 "${ACTIVE_MODELS[1]}"
run_round 3 "${ACTIVE_MODELS[2]}"

if ls "$RUN_DIR"/round_*.failover >/dev/null 2>&1 && [ "$FAILOVER_TIER" = "" ]; then
  FAILOVER_TIER="ollama_local"
  FAILOVER_REASON="per-round error mid-review (see round_*.failover)"
fi
```

---

## STEP 4: Parse Each Response → Findings Array

For each round 1, 2, 3:

```bash
parse_round() {
  local n="$1"
  local raw content
  raw=$(cat "$RUN_DIR/round_${n}.raw.json")
  content=$(chat_content "$PROVIDER" "$raw")
  content=$(printf '%s' "$content" | sed -E 's/^```(json)?//; s/```$//')
  if ! echo "$content" | jq -e 'type == "array"' >/dev/null 2>&1; then
    echo "[]" > "$RUN_DIR/round_${n}.findings.json"
    echo "warn: round ${n} response not a JSON array — treating as 0 findings" >&2
    return
  fi
  echo "$content" > "$RUN_DIR/round_${n}.findings.json"
}
parse_round 1; parse_round 2; parse_round 3
```

If a round returned prose: skip its findings (already 0). Don't retry.

---

## STEP 5: Aggregate — Dedupe + Vote Count

```bash
jq -s '
  flatten
  | group_by(.file + ":" + (.line|tostring))
  | map({
      file:     .[0].file,
      line:     .[0].line,
      votes:    length,
      models:   [.[] | .model // ""],
      bodies:   [.[] | .body],
      body:     ([.[] | .body] | sort_by(length) | last),
      severity: ([.[] | .severity] | unique | (if any(. == "error") then "error" elif any(. == "warning") then "warning" else "suggestion" end)),
      layer:    ([.[] | .layer]    | min)
    })
  | sort_by(.layer, (if .severity == "error" then 0 elif .severity == "warning" then 1 else 2 end), -.votes)
' "$RUN_DIR/round_1.findings.json" "$RUN_DIR/round_2.findings.json" "$RUN_DIR/round_3.findings.json" \
  > "$RUN_DIR/aggregated.json"
```

Tag findings with their model during Step 4 before aggregation for accurate
`models[]`: pipe each `findings.json` through `jq --arg m "$MODEL_RX" 'map(. + {model:$m})'` first.

---

## STEP 6: Arbiter (Claude)

Read `$RUN_DIR/aggregated.json`. For each finding, rule:

| Ruling | When |
|---|---|
| **CONFIRM** | Real issue. Default for `votes ≥ 2` unless clearly false-positive. |
| **ESCALATE** | Real issue, more severe than tagged. |
| **DISMISS** | False positive, conflicts with project context, or layer-5 noise. Default for `votes == 1` unless obviously real. |
| **DEFER** | Real but out of scope. Log, don't fix. |

**Vote count is a confidence prior, not a verdict.** A 3-vote finding that
contradicts a methodological invariant listed in `CLAUDE.md` (e.g. "make
D_const graph-based" or "hide invalid generations") should be dismissed with
reason — those are explicit intentional deviations, not bugs.

**Independent scan**: also walk the full diff once for anything all three
models missed.

**Apply CONFIRM + ESCALATE fixes** via the Edit tool. Minimal change per fix.

**Run gates** after fixes:
```bash
go build ./... && go vet ./... && go test ./... 2>&1 | tail -30
golangci-lint run 2>&1 | tail -20
```

### Diff-scope check before reverting

Before reverting a fix on gate failure:
1. `gh pr diff "${PR_NUMBER}" --repo valpere/tumanomir --name-only`
2. Map failure to touched file types — a Go build/vet/test failure only
   implicates changed `.go` files; a lint failure implicates those plus lint
   config.
3. If the failing layer is touched by the diff → find the fix at fault,
   revert via Edit, log `reverted — caused go test failure`, re-run gates.
   If not touched → mark pre-existing, log it, do not revert, do not
   silently pass either.

A docs/config/`.claude/`-only PR cannot cause Go build failures by
construction — such failures are always pre-existing there.

```bash
GATES_OK=yes  # default optimistic; flip to "no" if any in-scope failure remains
```

---

## STEP 7: Commit + Push (single commit)

```bash
git add -A
git restore --staged .claude/skills/fix-review/telemetry.jsonl 2>/dev/null || true

git commit -m "fix(pr#${PR_NUMBER}): address /fix-review findings

$(jq -r '.[] | select(.ruling=="CONFIRM" or .ruling=="ESCALATE") | "- \(.file):\(.line) — \(.body[0:80])"' "$RUN_DIR/arbiter.json" | head -20)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"

git push
```

Skip the commit entirely if zero fixes applied.

---

## STEP 8: Optional PR Summary Comment

`post_summary_to_pr: true` in config — always post:

```bash
gh pr comment "$PR_NUMBER" --repo valpere/tumanomir --body "$(cat <<EOF
<details>
<summary>/fix-review — ollama parallel pass · ${TOTAL_FINDINGS} findings · ${CONFIRMED} fixed · ${DISMISSED} dismissed</summary>

| File:Line | Votes | Layer | Sev | Ruling | Note |
|-----------|-------|-------|-----|--------|------|
${TABLE_ROWS}

Models: ${MODEL_R1}, ${MODEL_R2}, ${MODEL_R3}
Arbiter: Claude (vote count used as confidence prior)
</details>
EOF
)"
```

---

## STEP 9: Telemetry — JSONL Append

One entry per round + one arbiter entry, all in the same run.

**Round entry:**
```jsonc
{"timestamp": "2026-07-03T12:34:56Z", "pr_number": 1, "round_number": 1, "model": "glm-5.1:cloud", "provider": "ollama", "findings_count": 4, "prompt_tokens": null, "completion_tokens": null, "estimated_cost_usd": 0.0, "duration_ms": 8200}
```

**Arbiter entry:**
```jsonc
{"timestamp": "2026-07-03T12:35:10Z", "pr_number": 1, "round_number": "arbiter", "model": "claude", "provider": "local", "confirmed": 5, "escalated": 1, "dismissed": 3, "added_new": 1, "parallel": false, "wall_time_ms": 24000}
```

`wall_time_ms` is the Step 3 block duration (START_MS → end of round 3) — note
rounds run **sequentially** here (see STEP 0), so there is no parallel speedup
to report; `wall_time_ms` ≈ sum of round `duration_ms`.

**Append (fail-open):**
```bash
if [ "$TELEMETRY_ENABLED" = "true" ]; then
  jq -cn ... '{...}' >> "$TELEMETRY_FILE" 2>/dev/null || \
    echo "warn: telemetry write failed — continuing" >&2
fi
```

Cost is always 0 — all configured models are Ollama cloud/local (free tier).

---

## STEP 10: Gate Check (no merge — /ship owns merging)

This project runs `/ship`, which owns the merge step. `/fix-review` **never
calls `gh pr merge`**. It only computes and reports gate status:

```bash
MERGEABLE=$(gh pr view "$PR_NUMBER" --repo valpere/tumanomir --json mergeable --jq '.mergeable')
HAS_REVERT=$(jq -e 'any(.[]?; .ruling == "reverted")' "$RUN_DIR/arbiter.json" >/dev/null 2>&1 && echo "yes" || echo "no")
GATES_OK="${GATES_OK:-yes}"

READY="yes"
[ "$MERGEABLE" != "MERGEABLE" ] && READY="no"
[ "$HAS_REVERT" = "yes" ]       && READY="no"
[ "$GATES_OK"  != "yes" ]       && READY="no"

echo "Ready for /ship to merge: ${READY}"
[ "$READY" = "no" ] && echo "  Blocking: mergeable=${MERGEABLE} revert=${HAS_REVERT} gates=${GATES_OK}"
```

**→ Now proceed to Step 11.** See *Run Completion Contract* near the top.

---

## STEP 11: Final Summary (printed)

```
## /fix-review (parallel) — PR #${PR_NUMBER}

Provider: ${ACTIVE_PROVIDER}
Models:   ${MODEL_R1} | ${MODEL_R2} | ${MODEL_R3}
Wall time: ${WALL_TIME_MS} ms (sequential — no parallel speedup in this project's Ollama config)

Aggregated findings: ${TOTAL}
  3/3 votes: ${THREE_VOTE} (high confidence)
  2/3 votes: ${TWO_VOTE}
  1/3 votes: ${ONE_VOTE}

Arbiter:
  Confirmed: ${CONFIRMED}
  Escalated: ${ESCALATED}
  Dismissed: ${DISMISSED}
  Deferred:  ${DEFERRED}
  Added new: ${ADDED_NEW}

Tests: ${TEST_RESULT}
Lint:  ${LINT_RESULT}

Commit: ${COMMIT_SHA}
PR:     ${PR_URL}
Ready for /ship to merge: ${READY}
Telemetry: ${TELEMETRY_FILE}
```

**Failover reporting (mandatory):** if `FAILOVER_TIER` is non-empty, append a
`### ⚠️ Provider failover` section — see the canonical library skill
(`~/wrk/common/skills/fix-review/SKILL.md`) STEP 11 for the exact wording per
failover kind (`ollama_local` vs `cloud_unavailable`).

---

## SWITCHING PROVIDERS

Config: edit `.claude/skills/fix-review/config.yaml`.

Or ask: "switch fix-review to ollama local only" (sets local models under
`reviewers.ollama` directly and drops the `ollama_local` tier).
