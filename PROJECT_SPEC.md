# Job Scheduler Service Specification

## Purpose

Build a lightweight Go backend service that loads job definitions from a directory, schedules one-time and recurring work, runs configured commands, and records execution outcomes. It keeps scheduling state in memory and reconstructs that state from the definition files when it starts.

## Confirmed requirements

- **R1 — Directory-backed jobs:** Read JSON job definitions from a configuration directory at startup.
- **R2 — Live changes:** Add a job when its file appears, apply updates when a file changes, and remove a job when its file is deleted, without restarting the service.
- **R3 — One-time schedules:** Accept a future timestamp in ISO 8601/RFC 3339 form.
- **R4 — Recurring schedules:** Accept five-field minute-resolution cron expressions and six-field second-resolution expressions, including `*/5 * * * *` and `*/10 * * * * *`.
- **R5 — Extensible task types:** Execute `execute_command` tasks now and keep task dispatch separable so future task types can be added without changing scheduler logic.
- **R6 — Concurrent execution:** Permit overlapping scheduled jobs to execute concurrently.
- **R7 — Execution log:** Record each execution's job ID, execution time, and `SUCCESS` or `FAILURE` status.
- **R8 — In-memory state:** Do not require a database; rebuild active schedules from files at startup.
- **R9 — Go backend:** Implement the service in Go.
- **R10 — Parse outcome logging:** Report whether each job definition was parsed/validated successfully or failed, through the Logger service.
- **R11 — Command output logging:** Include the command's combined standard output and standard error in the execution log record.
- **R12 — In-code cron and parallel execution:** Implement five/six-field cron parsing and matching in this project; dispatch overlapping scheduled executions concurrently using goroutines.
- **R13 — Sub-minute frequency:** Support seconds in cron schedules so a job can repeat every ten seconds.

## Input format

Each `.json` file defines one job:

```json
{
  "job_id": "report-001",
  "description": "Generate the daily sales report.",
  "schedule": "0 1 * * *",
  "task": {
    "type": "execute_command",
    "command": "python scripts/generate_sales_report.py"
  }
}
```

The `schedule` is either a five-field cron expression, a six-field cron expression with seconds first, or an RFC 3339 timestamp. Five-field expressions fire at second zero; six-field expressions can schedule sub-minute frequencies. The file name is not the job identity; `job_id` is.

## Service interface

- Run as a foreground CLI application with flags for the jobs directory, execution log path, worker limit, and command timeout.
- The CLI starts the services, prints a readiness message after the initial directory scan, and remains running until interrupted. Startup errors and shutdown are reported on stderr.
- Provide a `validate` command (or equivalent CLI mode) to parse and validate the configured job files without scheduling or executing them.
- Parse and execution outcomes are appended as JSON Lines to the configured log file.

## Service responsibilities and data flow

1. **Scheduler Manager service:** Watch the job directory, perform the startup scan, and coordinate job additions, updates, and removals.
2. **Parser service:** Parse and validate each JSON definition; when the Scheduler signals a run, dispatch the configured task to the task handler. Keep task dispatch extensible for future types.
3. **Scheduler service:** Register jobs and select the scheduling strategy. Implement five- and six-field cron parsing and second-level matching in this project for recurring schedules, and use Go timers for one-time RFC 3339 timestamps. Replacing or removing a job cancels future triggers.
4. **Logger service:** Receive parse/validation and execution results, write structured diagnostics to stderr, and append parse and execution records to the JSONL log. Each record identifies its event kind, time, and success/failure status; execution records also include the job ID and combined command output.

The manager sends file changes to the parser, then supplies valid job definitions to the scheduler. Parser reports parse success/failure to Logger. At a trigger, the scheduler launches the parser/task dispatcher in a goroutine, bounded by the configured worker limit, then sends the outcome to Logger.

## Assumptions to confirm during plan review

- Cron schedules use UTC for predictable behavior across hosts.
- The in-code cron parser supports numeric five- and six-field expressions with wildcards, lists, ranges, and step values. In six-field expressions, seconds come first. Named month/day aliases are out of scope.
- A one-time job whose timestamp is already past is loaded but not run immediately.
- Invalid or unsupported job files are logged and skipped. If an update is invalid, keep the last valid in-memory version; if a job is newly invalid, do not schedule it.
- Deleting or changing a job cancels future triggers. A command already running is allowed to finish.
- Command definitions are trusted operator input. `execute_command` invokes a shell and therefore can run arbitrary commands.
- Job files are in one flat directory, use `.json`, and contain unique `job_id` values. A duplicate ID is rejected and reported.
- Concurrent commands are capped at a small configurable limit and wait for a worker slot; the initial default will be four.
- Commands have a configurable timeout, initially five minutes, after which they fail and are logged.
- Log entries use UTC RFC 3339 timestamps and append to the configured file; standard service diagnostics go to stderr. Execution log records include all captured command stdout and stderr, but not the command string.
- No network interface is required; job management is done through files and CLI options.

## Out of scope for the first version

- Persistent database or recovery of execution history beyond the append-only log.
- Retries, dependencies between jobs, distributed scheduling, job history UI, or task types beyond `execute_command`.
- Network APIs for job management or status.

## Open decisions / risks

- The challenge brief says the time limit is 75 minutes, but its listed round timeline reaches 90+ minutes. This plan uses 75 minutes as the stated limit.
- Confirm whether cron should use UTC or host-local time, and whether the proposed worker limit and command timeout are acceptable.
- Confirm whether “interval,” “timestamp-based,” and “one-time” refer to two strategies (recurring interval/cron and one-time timestamp) or three distinct schedule forms.
- Shell-command definitions are powerful and must only be writable by trusted operators.
- Capturing full command output in memory until the command exits can use substantial memory for very verbose commands; a streaming or output-size policy may be needed if that becomes a practical concern.
- File watcher events may arrive in bursts or while a file is partially written; implementation should debounce and validate complete JSON before replacing a valid job.
