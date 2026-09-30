# Job Scheduler CLI

A small Go scheduler that rebuilds in-memory schedules from JSON files, watches the job directory for changes, executes configured commands, and appends parse and execution results to a JSONL log.

## Structure

- `internal/manager`: startup scan, directory watcher, and job add/update/remove coordination.
- `internal/parser`: JSON validation and task-type dispatch.
- `internal/scheduler`: one-time and recurring cron scheduling plus concurrent execution limits.
- `internal/cronexpr`: in-code five/six-field cron parsing and second-level matching (numeric wildcards, lists, ranges, and steps).
- `internal/task`: task handlers; `execute_command` is implemented.
- `internal/logging`: parse and execution JSONL records, including combined command output.
- `jobs/sample.json`: safe recurring example that prints a short line every five minutes.

## Run

Validate the job files:

```sh
go run . validate -config-dir ./jobs
```

Start the scheduler:

```sh
go run . run -config-dir ./jobs -log-file ./logs/executions.jsonl
```

Override the concurrency and timeout defaults if needed:

```sh
go run . run -config-dir ./jobs -log-file ./logs/executions.jsonl -max-concurrency 4 -command-timeout 5m
```

Stop with Ctrl-C. Each log line is a JSON object. Parse events contain the file, job ID when available, and status. Execution events contain the job ID, completion time, status, combined stdout/stderr in `output`, and an error message on failure. The command string itself is not logged.

Job definitions use either a five-field cron expression, a six-field cron expression with seconds first, or an RFC 3339 timestamp in `schedule`. Five-field expressions run at second zero; six-field expressions support sub-minute schedules such as `*/10 * * * * *`. The in-code parser supports numeric wildcards, lists, ranges, and steps; cron runs in UTC. Job files are trusted input because `execute_command` invokes `sh -c`.
