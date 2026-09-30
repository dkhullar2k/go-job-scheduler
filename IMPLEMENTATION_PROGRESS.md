# Progress

## Current phase

Phase 4 — core features. Initial CLI vertical slice is implemented and manually exercised.

## Done

- Added Go CLI modes for `run` and `validate`.
- Added Scheduler Manager, Parser, Scheduler, task handler, and Logger packages.
- Added directory startup scan and live JSON file watching with debounced updates.
- Added cron and one-time RFC 3339 scheduling, bounded command concurrency, and command timeouts.
- Added parse outcome logging and execution records with combined stdout/stderr output.
- Implemented an in-project numeric five/six-field cron parser and second matcher; cron expressions support wildcards, lists, ranges, and steps.
- Scheduler dispatches cron and one-time jobs through goroutines and limits active command executions with a worker semaphore.
- Removed the external cron module; filesystem watching remains the only direct third-party dependency.
- Added a harmless recurring example job and run instructions.
- `go build ./...` completed successfully.
- `go run . validate -config-dir ./jobs` reported `Validated 1 job definition(s).`
- Manual runtime check ran a one-time shell command and produced an execution JSONL record with `SUCCESS` and output `captured scheduler output\n`.
- Manual cron runtime check started two `* * * * *` jobs at one minute boundary; both logged successful output. Their completion times were about 3 ms apart after each command slept for four seconds, showing parallel execution.
- Interactive terminal run printed the readiness message, executed a one-time demo job, logged its successful command output, and shut down cleanly on Ctrl-C.
- Updated the Hello World job to six-field `*/10 * * * * *`, for execution every ten seconds.
- Removed the third-party cron module; only the filesystem watcher is external.
- Manual cron run started two `* * * * *` jobs together; both completed successfully with output logged, and completion timestamps were about 3 ms apart after each command slept for four seconds.

## In progress

- Clarify whether timestamp-based and one-time schedules are distinct strategies.

## Remaining

- Add/complete automated tests if requested or authorized.
- Clarify whether timestamp-based and one-time schedules are distinct strategies.
- Commit the completed application files if approved; the local commit attempt was rejected by the approval prompt.

## Tests

Automated test suite not run. Build, CLI validation, and the command-output logging path were checked manually.

## Runtime validation

PASS — Started the CLI with a temporary one-time job, observed readiness, allowed the command to run, and inspected JSONL. The execution record contained `job_id: output-check`, `status: SUCCESS`, and `output: "captured scheduler output\\n"`.
PASS — Interactive run of `/private/tmp/job-scheduler-terminal-demo-20260930` loaded `terminal-demo`, recorded `SUCCESS` with output `Terminal demo: command ran successfully\n`, then stopped cleanly on Ctrl-C.

## Known issues

- Command output is captured in memory until the command exits; very verbose commands can use substantial memory.
- The scheduler currently treats recurring cron/interval schedules and one-time RFC 3339 timestamps as two schedule forms; the strategy wording may intend three.

## Next step

Resolve the schedule-strategy distinction and decide whether to authorize the local commit.
