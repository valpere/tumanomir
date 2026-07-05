# tumanomir — external documentation audit: docs/roadmap.md (agy)

> Model: Gemini 3.1 Pro (High) via agy -p. Full-codebase context (agent explored the repo
> directly via its own tools, not embedded doc text). Part of a
> 5-agent x 3-document audit round, 2026-07-04. Read-only: agent was
> explicitly instructed not to edit/create/delete any repo file;
> confirmed clean afterward via git status.

---

I will list the current permissions to identify the workspace directory and check my access.
I will list the contents of the scratch directory to locate the `tumanomir` project folder.
I will list the contents of the worktrees directory to see if the project is located there.
I will run a command to find the `tumanomir` directory in the user's home directory.
I will run a command to check the contents of `/home/val/wrk/projects/tumanomir`.
I will list the files in `/home/val/wrk/projects/tumanomir/tumanomir`.
I will display the contents of the `docs/roadmap.md` file to understand the current roadmap of the tumanomir project.
I will display the contents of the `docs/architecture.md` file to see the system design and look for deferred concerns or TODO comments mentioned in the codebase or design documentation.
I will check the files inside the `docs/` folder to see what other documentation is available.
I will print the contents of `docs/requirements.md` to see the requirement markers and check for any gaps/features that the roadmap should address.
I will read the contents of `CLAUDE.md` to get development instructions and methodology invariants.
I will check the open GitHub issues using `gh` to see if there are any tactical debt issues that we should check against the roadmap.
I will request permission to run the `gh` command to list GitHub issues for the `valpere/tumanomir` repository.
I will check the contents of the `.github` directory to see if there are any issue templates or other files that might list issues.
I will run `git log` to see recent commit messages and understand the history and current branches of the tumanomir repository.
I will run `git branch -a` to see all local and remote branches.
I will run `git status` to see if there are any untracked or modified files in the repository.
I will search for "TODO" comments in the Go codebase (`cmd/` and `internal/` directories) to identify any unfinished tasks.
I will search for "FIXME" comments in the repository.
I will display the contents of `cmd/tumanomir/main.go` to inspect the main CLI wiring and locate the `TODO(REQ-OUT-01)` markers or other inline comments.
I will view the first 120 lines of `cmd/tumanomir/main.go` to see the command-line flags, subcommand setup, and where `TODO(REQ-OUT-01)` comments might be located.
I will run a command to search for `TODO` or `FIXME` in the Go codebase (`cmd/` and `internal/` folders).
I will print lines 180 to 220 of `cmd/tumanomir/main.go` to see the second `TODO(REQ-OUT-01)` comment.
I will check if there are any `FIXME` comments in `cmd/` and `internal/`.
I will run the project's tests to verify the correctness of the current codebase.
I will check the files inside the `internal/dispersion` directory.
I will list the contents of `internal/dispersion/testdata/` to see what files are present for future dispersion tests.
I will list all test files in the Go codebase to see where coverage exists.
I will check the git log of the `test-dispersion-testdata-fixture` branch to see if there is any unmerged work or if it represents an open development effort.
I will run a command to list all local branches that are not merged into `main`.
I will check the commits on `feat-measure-dpair-max-flag` compared to `main`.
I will print the last 20 commits on `main` to understand what features have already been merged from these branches.
I will display `internal/instrument/ollama.go` to see the actual implementation of the Ollama generator backend.
I will list the files in the `internal/` directory to see the available shared packages and files.
I will display `internal/types.go` to inspect the shared types in the project.
I will print the contents of `internal/dispersion/dispersion.go` to verify if the dispersion measurement logic is Go-specific.
I will display the contents of `internal/dispersion/astfeat.go` to see the Go AST feature extraction details.
I will run a search for "TODO", "FIXME", "REQ-", or "concern" in all markdown files under the `docs/` directory to see if there are other unfinished items.
I will run the search again, excluding the `_sanity` test data subdirectory, to find relevant architectural references to deferred items, TODOs, or future work.
I will display the section in `docs/requirements.md` detailing what is out of scope to verify if there's anything missing from the roadmap.
I will extract all requirement tags (`[REQ-` style) from `docs/requirements.md` to map them against the code and roadmap.
I will view the contents of `internal/metrics/metrics_test.go` to see if the performance benchmark required by `[REQ-NFR-01]` is implemented.
I will view the English roadmap (`docs/roadmap.en.md`) to verify if it is aligned with the Ukrainian original (`docs/roadmap.md`).
I will list the files in the `docs/investigation/reports/` directory to see the external reviews.
I will view the contents of `internal/metrics/metrics.go` to see if there are any inline comments, TODOs, or signs of unfinished implementation.
I will list the files in the `internal/metrics` directory.
I will print the contents of `internal/metrics/kdrift.go` and `internal/metrics/dconst.go`.
I will view the files in `internal/spec/` to check for any unfinished work or design issues.
I will run a command to see where the `Generator` interface is defined in `internal/instrument/`.
I will display the contents of `internal/instrument/instrument.go` to see the `Generator` interface definition.
I will check the files inside the `docs/investigation` directory to understand the historical context.
I will check the exact line numbers for `checkResult` and `measureResult` structs in `cmd/tumanomir/main.go` to construct precise code links.
# Audit Report: `docs/roadmap.md` (tumanomir v0.1)

This report presents a structured critique of the [roadmap.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/roadmap.md) (and its English translation [roadmap.en.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/roadmap.en.md)) for the **tumanomir** Go CLI project. It evaluates the roadmap's strengths, weaknesses, coverage gaps against the system architecture, and overall utility.

---

## What's good

- **Logical Horizon Ordering**: The division of items into *Near-term*, *Mid-term*, and *Exploratory* horizons is sensible and well-justified:
  - **Near-term** items focus on architectural cleanup and interface stabilization. Extracting the inline rendering logic from [cmd/tumanomir/main.go](file:///home/val/wrk/projects/tumanomir/tumanomir/cmd/tumanomir/main.go) into a dedicated [internal/report](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/report) package is a natural precursor to building `.tumanomir.yaml` and the `gate` command without duplicating code.
  - **Mid-term** items address core empirical goals, such as baseline calibration (`tumanomir calibrate`) and addressing statistical variance (Bootstrap CI). These properly wait for a stable base of usage history.
  - **Exploratory** items contain high-uncertainty or long-term design ideas (RFLP graph databases, LLM-based traceability extraction, alternative API providers, and multi-language projections) that have not been scoped.
- **Actionable Item Scoping**: Each of the 8 roadmap items is concrete. The descriptions clearly state the engineering problem being solved, explicitly identify the affected codebase files/symbols (e.g. referencing `TODO(REQ-OUT-01)` in [cmd/tumanomir/main.go](file:///home/val/wrk/projects/tumanomir/tumanomir/cmd/tumanomir/main.go)), and note any dependencies (e.g., explaining why the `gate` subcommand depends on package `report`).
- **Respected Task Boundaries**: The roadmap respects the distinction between long-term product features and tactical debt. It correctly leaves minor bug fixes, CLI flag adjustments, HTTP client timeouts (like those in [internal/instrument/ollama.go](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/instrument/ollama.go)), and test coverage improvements to GitHub issues or local development branches.

---

## What's bad

- **Neo4j / RFLP Architectural Contradiction**: Item 5 (*"RFLP-граф (Neo4j) для повного D_const"*) suggests replacing the lexical constraint-density scanner with a full graph-based representation in Neo4j. This introduces a major, unacknowledged architectural conflict with the project's strict non-functional constraint `[REQ-NFR-02]` in [requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md), which mandates a *"Single static binary, stdlib-only"* design. Integrating a Neo4j driver or expecting users to run an external Neo4j database contradicts the project's goal of a lightweight, zero-dependency pre-commit hook.
- **Vague Configuration Strategy for Other Instruments**: Item 7 (*"Інші прилади"*) outlines adding OpenAI and Anthropic API backends to complement Ollama. However, the roadmap fails to address how credentials/API keys, custom endpoints, and secrets will be configured (e.g., whether they will be managed in `.tumanomir.yaml`, environment variables, or new command-line flags).
- **Manual English Translation Synchronization**: The codebase maintains two identical roadmaps ([roadmap.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/roadmap.md) as the source of truth, and [roadmap.en.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/roadmap.en.md) for accessibility). This manual duplication introduces a risk of synchronization drift as the project evolves, which could be automated or mitigated by keeping documentation in a single unified format.

---

## What it doesn't cover

- **Missing Performance Benchmark (`[REQ-NFR-01] / [PHY-NFR-01]`)**: The project specifications in [requirements.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/requirements.md) mandate a deterministic-layer performance target:
  > *`[REQ-NFR-01] check on a 1 MB spec corpus must complete in under 100 ms... -> [PHY-NFR-01] benchmark in internal/metrics`*
  
  The benchmark is entirely missing from the codebase test suite ([internal/metrics/metrics_test.go](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/metrics/metrics_test.go)). While this is a test gap, the fact that a core physical requirement remains unverified and unmentioned as an open item in the roadmap is a notable omission.
- **Untested Stochastic Engine ([internal/dispersion](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/dispersion))**: Although [CLAUDE.md](file:///home/val/wrk/projects/tumanomir/tumanomir/CLAUDE.md) states that test fixtures are present in `testdata/`, the core package [internal/dispersion](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/dispersion) contains zero Go test files. Given that the math behind [dispersion.Analyze](file:///home/val/wrk/projects/tumanomir/tumanomir/internal/dispersion/dispersion.go#L12-L44) (AST cosine similarity clustering, Shannon entropy) is the most mathematically complex part of the codebase, leaving the package completely untested on the main branch is a major risk that should be explicitly scheduled for resolution.
- **Error Handling and API Resilience**: While the requirements define rules for truncation (`[REQ-MSR-06]`) and invalid retries (`[REQ-MSR-05]`), the roadmap does not schedule any improvements for network recovery, HTTP request retries, or rate-limiting handling. When querying remote API providers (like OpenAI or Anthropic in Item 7), robust error handling will be critical to prevent network noise from skewing the `D_pair` stochastic measurements.

---

## Verdict

In its current state, [docs/roadmap.md](file:///home/val/wrk/projects/tumanomir/tumanomir/docs/roadmap.md) is a highly useful and structured planning document. It establishes a logical development sequence and keeps the project focused on feature development rather than tactical bugs. However, it requires refinement in its exploratory section to resolve the architectural conflict between introducing Neo4j and maintaining the zero-dependency, stdlib-only constraints of the tool. Once these architectural contradictions are addressed and the missing benchmark requirement is accounted for, the roadmap will be fully trustworthy.
