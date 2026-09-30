# Implementation Plan

## Goal

Make the existing Go job scheduler repository easier to maintain and present as production quality while preserving its current observable behavior. The user has set no time limit (“Take your time”), so work will be bounded by completion and review rather than a fixed timebox.

## Requirements (confirmed)

- **R1 — Behavior preservation:** retain current CLI modes, flags, defaults, messages, exit behavior, JSON job schema, cron syntax/time semantics, one-time schedules, file watching, concurrency, timeouts, and JSONL log fields and contents.
- **R2 — Maintainability:** improve code organization, naming, comments, error handling, and lifecycle clarity only where behavior stays equivalent.
- **R3 — Production presentation:** make root documentation and repository metadata clear and consistent with what the program actually does; remove stale or duplicated project notes when their replacement is accurate.
- **R4 — Reviewability:** keep the diff focused, do not add dependencies or new capabilities, and leave unrelated untracked user files untouched.
- **R5 — Evidence:** use existing or characterization tests and build/static checks to compare behavior before and after cleanup. Report commands run and results.

## Assumptions

- “Production like” means readable, consistent, documented, and operationally clear code and repository contents; it does not imply deployment infrastructure, a network API, persistence, security sandboxing, or new features.
- Existing behavior is the contract, including quirks discovered in the current implementation. Potential bug fixes that change accepted inputs, scheduling, outputs, or failure modes are out of scope and will be recorded as follow-up decisions.
- Preserve `main` and the current remote configuration. Do not push changes unless separately requested.
- Keep current package layout and CLI entry points unless a demonstrably behavior-neutral refactor requires otherwise.

## Stack and architecture

- Go module and existing dependencies remain unchanged.
- Retain the current CLI with manager, parser, scheduler, cron expression, task execution, model, and logging packages.
- Preserve the current in-process, goroutine-based execution model and filesystem watcher.
- No new external services, frameworks, dependencies, or runtime configuration.

## API / Interface and data model

- Freeze the current command-line interface, defaults, readiness/error output, and exit codes as observed at baseline.
- Freeze job JSON fields and supported schedule forms as observed at baseline.
- Freeze JSONL record schema, event content, command output capture, and append behavior as observed at baseline.
- Avoid exported API or data model changes; where internal refactoring is needed, keep observable behavior equivalent.

## Implementation steps

1. Record repository status and current behavior from source/docs; inspect all tracked files and identify stale, inconsistent, or redundant material.
2. Establish a baseline with the current build and existing tests; add focused characterization coverage for important current behavior where missing.
3. Make small, behavior-neutral code cleanup changes: formatting, clearer names/comments, cohesive helpers, explicit resource/lifecycle handling, and consistent error context.
4. Reconcile README, implementation notes, and examples with actual current behavior; remove or consolidate stale documentation only when the replacement preserves useful project history and instructions.
5. Review the complete diff against R1–R5, run verification, and summarize any discovered behavior quirks as open follow-up items rather than changing them.

## Test strategy

- Capture baseline behavior with existing tests and command observations before refactoring.
- Add characterization tests only for current contracts that lack coverage; avoid tests that assert proposed new behavior.
- After each meaningful change, run relevant existing tests; at completion run formatting, `go test ./...`, and `go build ./...`.
- Compare CLI usage/output, validation behavior, scheduling and watcher behavior, command output capture, and JSONL records before and after wherever practical.
- No new behavior is accepted merely because tests pass; inspect the diff and confirm each change is behavior-neutral.

## Logging and health

- Preserve runtime log event schema and content exactly.
- Preserve startup/readiness, diagnostic destinations, shutdown handling, and process exit behavior.
- Improve only source-level clarity and documentation around the existing logging and lifecycle contracts.

## Run instructions

- Preserve the current documented commands for validation and running the CLI.
- Include the exact build/test commands used for verification in the completion summary.
- Do not change job examples or their schedules as part of cleanup.

## Risks and open decisions

- “Production like” has no prescribed checklist; this plan limits it to code quality, accurate docs, and repository hygiene while preserving behavior.
- The implementation may contain bugs or ambiguous behavior. Correcting those would conflict with the no-behavior-change requirement and needs a separate request.
- Removing stale notes can remove useful history; inspect each candidate before consolidating it.
- Baseline checks may reveal existing failures. Record them and distinguish them from cleanup regressions.
