# Implementation Plan

## Goal

Build a runnable Go job scheduler backend in the stated 75-minute challenge timebox. See `PROJECT_SPEC.md` for the complete behavior and assumptions.

## Requirements (confirmed)

1. **R1** Load JSON job definitions from a directory at startup.
2. **R2** Apply file additions, updates, and deletions while running.
3. **R3** Schedule one-time RFC 3339/ISO 8601 timestamps.
4. **R4** Schedule five-field minute-resolution and six-field second-resolution cron expressions.
5. **R5** Execute `execute_command` through an extensible task interface.
6. **R6** Run overlapping jobs concurrently.
7. **R7** Log each execution's job ID, UTC execution time, and success/failure.
8. **R8** Keep scheduling state in memory and rebuild it from files at startup.
9. **R9** Implement the backend in Go.
10. **R10** Log whether each job definition parsed/validated successfully or failed.
11. **R11** Include combined command stdout and stderr in each execution log record.
12. **R12** Implement five/six-field cron parsing/matching in this project and run overlapping jobs concurrently with goroutines.
13. **R13** Support sub-minute cron frequency, including every ten seconds.

## Assumptions

- Cron uses UTC and supports numeric five- or six-field expressions with `*`, comma lists, ranges, and `/step`; seconds are the first field in six-field expressions. Past one-time schedules do not run on startup.
- Invalid additions are skipped; invalid updates leave the last valid schedule in place.
- Deletion/update cancels future executions but does not kill an already-running command.
- Jobs are flat `.json` files with unique IDs; definitions are trusted operator input.
- Use a configurable worker limit (default four) and command timeout (default five minutes).
- Keep management local to the job-definition directory and CLI; no network API in v1.
- The document's explicit 75-minute limit is T; the timeline extending beyond 90 minutes is inconsistent.

## Stack and architecture

- Go 1.25.5 and Go modules; module download check succeeded.
- `github.com/fsnotify/fsnotify` watches the job directory. Five- and six-field cron parsing and second-level matching are implemented in this project; one-time jobs use Go timers.
- One foreground CLI process with separate Scheduler Manager, Parser, Scheduler, and Logger services, plus an extensible task dispatcher.
- Scheduler Manager performs an initial scan and watches the flat job directory. It passes file changes to Parser for complete-file parsing and validation, then registers valid definitions with Scheduler. It debounces writes; invalid updates preserve the last valid job.
- Scheduler owns registered jobs, evaluates in-code cron expressions at second boundaries, and uses Go timers for one-time timestamps. It cancels future triggers on update/removal and invokes the parser's task dispatch path in a goroutine when a job fires.
- Parser handles JSON parsing/validation and dispatches execution to a task handler selected by task type. Execution is bounded by a configurable worker pool and command timeout.
- Logger receives parse/validation and execution outcomes, writes service diagnostics to stderr, and appends parse and execution records to JSONL. Each record carries its event kind, time, and status; execution events carry job ID.

## API / Interface and data model

- CLI flags: `-config-dir`, `-log-file`, `-max-concurrency`, and `-command-timeout`.
- CLI modes: `run` starts the services and stays active until interrupted; `validate` parses and validates all configured job files without executing them.
- On successful initial scan, `run` prints a readiness message; fatal startup errors return non-zero.
- Job JSON: `job_id`, `description`, `schedule`, `task.type`, and `task.command`.
- JSONL events: `event_type` (`parse` or `execution`), `time` (UTC RFC 3339), and `status` (`SUCCESS` or `FAILURE`); execution events include `job_id`, full combined command output, and optional error detail. Parse events may include file and job ID.

## Implementation steps

1. **Vertical slice:** initialize the Go module, define the job model, load one JSON file, execute one due `execute_command` job, append its result to JSONL, and run the service end to end.
2. Implement five- and six-field cron parsing (numeric wildcards, lists, ranges, and steps), second-level matching, one-time timestamps, and schedule replacement/removal.
3. Add the filesystem watcher, startup scan, debouncing, update/delete behavior, and validation of duplicate IDs and malformed files.
4. Add bounded concurrent execution, command timeouts, and consistent success/failure logging, including combined stdout/stderr for every command execution.
5. Add CLI `run` and `validate` modes, graceful shutdown, readiness output, and useful startup/watcher/execution diagnostics.
6. Add deterministic tests for parsing and parse logs, schedule selection, file add/update/delete, concurrency, command success/failure/timeout, and CLI validation/readiness behavior.
7. Run formatting, all tests, and the actual service; exercise a temporary job through file creation, update, and deletion; review requirements and record evidence.

## Test strategy

- Validate valid/invalid JSON, schedule classification, duplicate IDs, in-code five/six-field cron parsing/matching, and one-time past/future handling.
- Use temporary directories and short test schedules to verify startup loading and watcher add/update/delete behavior without relying on long real cron intervals.
- Verify command success, non-zero exit, timeout, concurrent overlap, and one JSONL record per run.
- Verify CLI validation output, readiness on startup, and non-zero exit behavior for fatal startup errors.
- Run `gofmt`, `go test ./...`, then launch the service and exercise a real temporary job and file change.

## Logging and health

- Append one JSON object per parse or execution outcome to the configured JSONL file with event type, UTC time, and status; execution events include job ID, full combined command output, and concise error details on failures. Do not include the command string.
- Write startup, watcher, validation, and shutdown diagnostics to stderr.
- Treat successful initial scan plus the CLI readiness message as the service health signal; process failure returns a non-zero exit status.
- Do not log secrets; command strings may contain sensitive values, so exclude them from diagnostics.

## Run instructions

- Install dependencies: `go mod tidy`
- Test: `go test ./...`
- Validate definitions: `go run . validate -config-dir ./jobs`
- Start: `go run . run -config-dir ./jobs -log-file ./logs/executions.jsonl`

## Risks and open decisions

- The stated 75-minute challenge duration conflicts with a timeline that extends beyond 90 minutes; this plan follows 75 minutes.
- Confirmed architecture is a CLI process with Scheduler Manager, Parser, Scheduler, and Logger services; their initial boundaries are internal Go interfaces/components in one process.
- Confirm UTC versus local cron time, worker limit, command timeout, and whether the requested strategy terms mean two or three schedule forms. Current assumptions are UTC, four workers, five minutes, and two forms: recurring interval/cron and one-time timestamp.
- Shell execution runs arbitrary commands and assumes trusted, access-controlled job files.
- Full command output is captured until process exit, so very large output can consume memory; revisit streaming/bounds if workload needs it.
- Watchers can report partial writes; debounce and parse-before-replace are required to avoid losing a valid schedule.
- A process crash loses in-memory schedule state and any active command; definitions are restored on restart, while the log is append-only.
