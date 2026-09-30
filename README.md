# Go Job Scheduler

A foreground Go CLI that loads job definitions from JSON files, watches the directory for changes, runs scheduled shell commands, and appends parse and execution results to a JSON Lines log. Schedules are held in memory and rebuilt from the job files at startup.

## Requirements

- Go 1.23 or newer
- A POSIX shell (`sh`) for `execute_command` jobs

## Build and validate

Run the automated checks:

```sh
go test ./...
go build ./...
```

Validate job files without starting the scheduler:

```sh
go run . validate -config-dir ./jobs
```

## Run

```sh
go run . run -config-dir ./jobs -log-file ./logs/executions.jsonl
```

The CLI accepts these flags in both modes:

| Flag | Default | Description |
| --- | --- | --- |
| `-config-dir` | `./jobs` | Directory of JSON job definitions |
| `-log-file` | `./logs/executions.jsonl` | Append-only JSONL event log |
| `-max-concurrency` | `4` | Maximum number of commands running at once |
| `-command-timeout` | `5m` | Maximum runtime for a command |

The `run` mode prints a readiness message after its initial scan and watches the directory for additions, updates, and removals. Stop it with Ctrl-C or SIGTERM. Service diagnostics go to stderr.

## Job definitions

Each `.json` file defines one job. The job ID is the identity; the filename is not:

```json
{
  "job_id": "hello-world",
  "description": "Print a greeting",
  "schedule": "*/10 * * * * *",
  "task": {
    "type": "execute_command",
    "command": "echo Hello World"
  }
}
```

The scheduler accepts five-field cron expressions (minute resolution), six-field cron expressions (seconds first), and RFC 3339 timestamps for one-time runs. Cron supports numeric wildcards, lists, ranges, and steps; it evaluates in UTC. Five-field schedules run at second zero. Past one-time timestamps are not run immediately. Invalid definitions are logged; an invalid update leaves the last valid schedule registered.

The `jobs` directory contains a five-minute sample and a ten-second Hello World example. Commands execute through `sh -c`, so only use job files from trusted sources.

## Logs

Each line in the configured log file is a JSON object. Parse events record the file, job ID when available, and validation status. Execution events record the job ID, completion time, status, combined stdout and stderr, and an error message on failure. The command string is not included. The log file is opened with owner-only read/write permissions when newly created.

## Packages

- `internal/manager`: startup scan, file watcher, and job registration coordination.
- `internal/parser`: strict JSON parsing, validation, and task dispatch.
- `internal/scheduler`: cron and one-time scheduling, goroutine execution, concurrency limit, and timeouts.
- `internal/cronexpr`: in-code five- and six-field cron parser and matcher.
- `internal/task`: task handlers, including `execute_command`.
- `internal/logging`: parse and execution JSONL records.
