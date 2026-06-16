# TaskBridge Minimal Starter

TaskBridge is a Go intern assignment for building a cross-platform remote job runner.

This repository is a **starter skeleton**, not a completed implementation.

## What is already provided

- Go module setup
- `cmd/server` minimal runnable server
- `cmd/agent` minimal runnable agent placeholder
- shared job and agent models
- store interface
- executor interface and registry skeleton
- basic `/health` endpoint
- example job JSON files
- assignment notes

## What the candidate must build

- REST APIs for jobs and agents
- in-memory job and agent store
- agent registration and heartbeat
- job polling and assignment
- job execution and result reporting
- retry and timeout handling
- safe executors such as `http_check`, `tcp_check`, `file_exists`, `checksum`, `write_file`, and `wait`
- validation and structured errors
- tests
- improved README with demo steps

## Run server

```bash
go run ./cmd/server --addr :8080
```

Then test:

```bash
curl http://localhost:8080/health
```

## Run agent placeholder

```bash
go run ./cmd/agent --server http://localhost:8080 --id agent-dev-1
```

## Suggested implementation order

1. Implement API handlers and request DTOs.
2. Implement in-memory store using mutex-safe maps.
3. Implement agent registration and heartbeat.
4. Implement job creation and listing.
5. Implement job assignment.
6. Implement agent polling and result submission.
7. Implement executors.
8. Add retries, timeout, and cancellation.
9. Add tests.
10. Add documentation and demo commands.
