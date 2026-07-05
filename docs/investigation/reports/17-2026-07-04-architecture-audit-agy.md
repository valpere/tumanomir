# tumanomir — external documentation audit: docs/architecture.md (agy)

> Model: Gemini 3.1 Pro (High) via agy -p. Full-codebase context (agent explored the repo
> directly via its own tools, not embedded doc text). Part of a
> 5-agent x 3-document audit round, 2026-07-04. Read-only: agent was
> explicitly instructed not to edit/create/delete any repo file;
> confirmed clean afterward via git status.

---

I will check the active permissions to understand what directories I can access in this workspace.
I will run a command to discover the location of the `tumanomir` codebase and see the files in the directory.
I will run `find` and `ls -la` to locate any directories or files in our scratch folder.
I will run `ls -la` to see if there are other files in `/home/val` or if there is a `tumanomir` directory nearby.
I will run a command to find the location of the `tumanomir` project or `architecture.md` file on the filesystem.
I will check the status of the search task.
I will wait for the background task to complete.
I am waiting for the background search task to complete.
I will check the status of the search task.
I will run a command to list the contents of the `/home/val` directory, to see if there is a repository like `tumanomir`.
I will run `ls -la ../../..` to list the contents of the `/home/val` directory to find where the `tumanomir` codebase is located.
I will check if the `find` task has completed and view its output.
I will view the log of the `find` task to see if it has found the `tumanomir` architecture file.
I will run a command to list the contents of the `/home/val/wrk` directory, where many repositories seem to be located.
I will list the user's `projects` directory.
I will view the log of the `find` task again to see if it has finished or made more progress.
I will view the Geany project file to find where the `tumanomir` source code is located.
I will run `ls -F` on `/home/val/wrk/projects/tumanomir` to see its subdirectories and verify the repository root.
I will list `/home/val/wrk/projects/tumanomir/tumanomir` to see the structure of the repository.
I will list the files in the `docs` directory of the `tumanomir` repository.
I will view the top part of `docs/architecture.md` to see its language and outline.
I will run a command to display the entire contents of `docs/architecture.md` to study it.
I will run a command to display the requirements file so we can check if `architecture.md` satisfies all requirement references.
I will list all the files in the `tumanomir` codebase to understand its physical layout.
I will run a command to cat `CLAUDE.md` to see system instructions, build targets, and testing notes.
I will run the project's tests to verify the current state of the codebase.
I will print the contents of `internal/dispersion/dispersion.go`.
I will view the `internal/dispersion/astfeat.go` file to see how AST features are extracted.
I will print the contents of `internal/dispersion/cluster.go`.
I will run a command to view `internal/types.go`.
I will view `cmd/tumanomir/main.go` to see the CLI implementation and flag/exit code wiring.
I will print the first 200 lines of `cmd/tumanomir/main.go` to see the initialization, commands structure, and flags.
I will print the remaining lines of `cmd/tumanomir/main.go` to make sure we see the complete CLI logic.
I will print lines 160 to 250 of `cmd/tumanomir/main.go` to see the flags defined for the `measure` command and see if they match the architecture document.
I will search for `d-pair-max` in `docs/architecture.md` and `docs/architecture.en.md`.
I will run a command to search for `D_pair` in `docs/requirements.md`.
I will run a command to display the top 30 lines of `docs/requirements.md` to see the schema definitions.
I will read the invariants section from `CLAUDE.uk.md` to see the Ukrainian phrasing.
I will search for the word "gate" in `docs/requirements.md` to see which metrics are intended to be gates.
I will print the context around line 133 in `docs/requirements.md` to understand what "consistent with D_pair/H_norm staying ordinal/advisory in v0.1" refers to.
I will read `docs/architecture.en.md` to see how the Ukrainian text was translated into English.
I will run a command to view `internal/nonetwork_test.go` to check how the no-network constraint is tested.
I will print the contents of `internal/metrics/kdrift.go` to check the K_drift implementation.
I will view `internal/metrics/dconst.go` to check the lexical constraint density calculation.
I will print the contents of `internal/spec/spec.go` to check how markdown files are loaded and aggregated.
I will print the contents of `internal/instrument/instrument.go` to examine the Generator interface.
I will print `internal/instrument/ollama.go` to inspect the Ollama integration and see how the Ollama request is serialized, specifically how we enforce the constraints of `think: false` and detect silent truncation.
I will run a command to view `internal/instrument/prompt.go` to inspect the system prompt templates.
I will view `docs/roadmap.md` to check what forward-looking plans are documented.
# Audit Report: `docs/architecture.md` (tumanomir v0.1)

This report presents an audit of [architecture.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/architecture.md) for the "tumanomir" Go CLI project, verifying its alignment with the specification in [requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md) and the actual implementation in the codebase.

---

## What's good

- **Structural Alignment:** The "Архітектура пакетів" (Package Architecture) section accurately reflects the physical package layout of the codebase, including `cmd/tumanomir/` and the subpackages under `internal/`.
- **Preservation of Key Invariants:** It correctly documents critical architectural constraints, such as the deterministic layer being strictly zero-network/zero-LLM, which is enforced programmatically in [nonetwork_test.go](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/nonetwork_test.go).
- **Honest Technical Debt Representation:** The document correctly calls out that report rendering (`checkResult` and `measureResult`) remains inline in [main.go](file:///home/val/wrk/projects/tumanomir/tumanomir/cmd/tumanomir/main.go) and explicitly refers to the `TODO(REQ-OUT-01)` marker for moving it to a dedicated package, matching the plans in [roadmap.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/roadmap.md).
- **Detailed Methodological Background:** It provides helpful context for orienting a new engineer or AI agent by explaining the origin of the metrics and referencing the core paper and the provenance of the ported dispersion codebase.

---

## What's bad

- **Critical Methodological Invariant Contradiction:** 
  The "Методологічні інваріанти" list contains a direct contradiction between its first and fourth bullet points:
  - The *first* bullet point claims that `D_pair` is the working metric, while `H_norm` is *"the one actually reported/gated on"* ("саме вона репортиться/гейтиться").
  - The *fourth* bullet point claims that *"only K_drift and D_pair gate the exit code — D_const and H_norm are ordinal/advisory"*.
  - *Reality check:* The Go code in [main.go](file:///home/val/wrk/projects/tumanomir/tumanomir/cmd/tumanomir/main.go#L112-L115) gates exit code 1 on `D_pair` (if it exceeds `--d-pair-max`), whereas `H_norm` is treated as strictly ordinal and is never gated. The first bullet point is factually incorrect and contradicts the implementation. This contradiction also leaked into [CLAUDE.md](file:///home/val/wrk/projects/tumanomir/tumanomir/CLAUDE.md) and [CLAUDE.uk.md](file:///home/val/wrk/projects/tumanomir/tumanomir/CLAUDE.uk.md).
- **Stale/Inaccurate Requirement Citations:**
  - The fourth bullet point cites `(REQ-CHK-06)` as the justification for why `D_const` and `H_norm` are advisory/ordinal. However, `[REQ-CHK-06]` in [requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md) is strictly about the deterministic layer's `D_const` metric and makes no mention of `H_norm` (which is covered by `[REQ-MSR-02]`).
- **CLI UX Documentation Gaps:**
  - The "CLI UX" section for the `measure` command omits the `--d-pair-max` float flag. This is a real flag parsed by [main.go](file:///home/val/wrk/projects/tumanomir/tumanomir/cmd/tumanomir/main.go#L236) that overrides the default D_pair threshold of `0.30` (from [DefaultThresholds](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/types.go#L23)).
- **D_const Implementation Discrepancy:**
  - `[REQ-CHK-03]` specifies that prose tokens are "whitespace-separated words *outside* markers". However, the implementation in [dconst.go](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/metrics/dconst.go#L39) counts all whitespace-separated words in the document (`len(bytes.Fields(doc))`), including the markers themselves, counting them twice in the denominator. [architecture.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/architecture.md) fails to document this implementation detail or the resulting distortion.
- **Omissions of Ollama-Specific Mechanisms:**
  - The document describes the Ollama instrument but fails to specify the `/api/chat` payload structure, default 5-minute timeouts, and the `(len(prompt) + 2) / 3` heuristic used to prevent silent context truncation before making network calls.

---

## What it doesn't cover

- **Skipped Verdict for Empty Requirements (`[REQ-CHK-01]`):** 
  - If a spec has zero `[REQ-*]` tags, the deterministic checker must return a `skipped` verdict, not a pass. [architecture.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/architecture.md) completely omits this behavioral rule.
- **Hanging Requirement List (`[REQ-CHK-02]`):**
  - The requirement that the `check` command output the specific requirement IDs that are untraced (`HangingIDs` inside `KDriftResult`) is not mentioned anywhere in [architecture.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/architecture.md).
- **Go AST Feature Set Definition (`[REQ-MSR-01]`):**
  - While it mentions that `D_pair` measures structural similarity, it never explains *how* the feature vectors are constructed from the AST (types, struct/interface shapes, method signatures, top-level values). A developer reading only [architecture.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/architecture.md) would not know what Go code characteristics are analyzed by [astfeat.go](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/dispersion/astfeat.go).
- **Invalid Rate Retries & Warning Gating (`[REQ-MSR-05]`):**
  - While it lists the retry and 40% threshold parameters in its invariants, it does not outline the retry loop mechanics (up to 2 retries per sample, discarding instead of padding the sample count).
- **Truncation and Done-Reason Handlers (`[REQ-MSR-06]`):**
  - It does not cover the requirement to parse and report output truncation using the Ollama backend's `done_reason` ("stop" vs. "length") to alert the user that a generation was cut off by `--num-predict`.
- **Output Specifications (`[REQ-OUT-01]`, `[REQ-OUT-02]`):**
  - It has no descriptions of how the human-readable TTY report aligns its output columns or handles exit codes programmatically based on block verdicts.
- **Non-Functional Requirements (`[REQ-NFR-01]`, `[REQ-NFR-02]`):**
  - It does not document the strict performance latency SLA (<100ms for a 1MB spec corpus) or the tool's requirement of compiling into a single static binary using Go >= 1.26 with stdlib-only dependencies in v0.1.

---

## Verdict

[architecture.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/architecture.md) is a valuable, concise starting point for orienting new developers, but it is currently **insufficient and contains critical errors**. Its description of the core gating logic contains a direct contradiction (claiming `H_norm` is gated instead of `D_pair`), it omits the vital `--d-pair-max` CLI flag, and it leaves out major specification details—such as skipped verdicts, AST feature extraction parameters, output truncation detection, and performance SLAs. It needs targeted revisions to resolve the gating contradictions and document the remaining requirements before it can serve as a reliable reference document.
