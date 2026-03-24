# FlowX

A lightweight, persistent run execution engine built in Go. FlowX takes a code-defined flow (an ordered list of steps), creates runs from API requests, fans them out to a pool of concurrent workers, and executes each run's steps sequentially — with automatic retries, resumability, and Slack alerting baked in.

There is no fixed use case. Define any sequence of steps, point FlowX at your database, and it will execute them reliably.

---

## Architecture

```
API Request ──► RunService ──► Queue (buffered channel) ──► Worker Pool ──► Executor ──► Steps
                    │                                                            │
               Runs (MongoDB)                                          Step Runs (MongoDB)
                                                                            │
                                                                   Slack Alerts (on failure)
```

### Core Concepts

| Concept | Description | Persisted? |
|---|---|---|
| **Flow** | A static, code-defined blueprint with an ordered list of steps | No |
| **Step** | A single unit of work with optional cleanup and execute phases | No |
| **Run** | An execution instance of a flow, created per API request | Yes (MongoDB) |
| **StepRun** | Tracks the execution state of a single step within a run | Yes (MongoDB) |

### How It Works

1. **Startup** — FlowX connects to MongoDB, spins up _N_ workers, and re-enqueues every incomplete run (`is_completed: false`) into a buffered channel.
2. **API** — A `POST /runs` request creates a new Run in MongoDB and enqueues it for processing.
3. **Workers** — Each worker is a goroutine polling the channel. Multiple runs execute in parallel across workers.
4. **Executor** — Checks the `step_runs` collection for the last recorded step of that run. If none exists, the full step list runs from scratch. If a previous run was interrupted, execution resumes from the point of failure.
5. **Step Execution** — Each step goes through two optional phases:
   - **Cleanup** — Tear down or reset state from a prior partial run.
   - **Execute** — The actual work. Receives `map[string]any` input, returns `map[string]any` output.
   
   The output of one step becomes the input of the next, forming a pipeline.
6. **Retries** — A failing step is retried up to **3 times** with a **1-minute** backoff between attempts. If all retries are exhausted, the step is marked `FAILED` and a Slack alert fires.
7. **Completion** — Once every step succeeds, the run is marked complete in MongoDB.
8. **Shutdown** — On `SIGINT`/`SIGTERM`, workers finish their current step, the HTTP server drains with a 5-second timeout, and the MongoDB connection is closed.

---

## Project Structure

```
.
├── cmd/flowx/              Entrypoint — config loading, logger, server init
├── config/                 Config struct, defaults, validation
├── errors/                 Typed application errors (Kind-based)
├── flow/                   Flow + Step definitions (code-only, not persisted)
├── http/
│   ├── handlers/           HTTP handlers (health, runs)
│   ├── middlewares/         Request logging middleware
│   ├── response/           JSON response helpers
│   └── server.go           Chi router, graceful shutdown
├── models/
│   ├── run/                Run data model (MongoDB document)
│   └── steprun/            StepRun data model (MongoDB document)
├── repositories/mongodb/   MongoDB repositories (runs, step_runs)
├── services/
│   ├── executor/           Step execution engine with retry + resume
│   ├── health/             Health check service (MongoDB ping)
│   └── run/                Run orchestrator — queue, workers, lifecycle
└── utils/
    ├── constants/          Shared constants
    ├── helpers/            Validation, time, HTTP utilities
    └── slack/              Slack webhook alerting
```

---

## Data Flow

### Run Record (`runs` collection)

```json
{
  "_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "created_at": "2026-03-22T10:00:00.000Z",
  "input": { "name": "test_user" },
  "is_completed": false,
  "completed_at": "0001-01-01T00:00:00.000Z",
  "last_step_status": false
}
```

### StepRun Record (`step_runs` collection)

Each step execution is recorded with its input, output, duration, and final state:

```json
{
  "_id": { "run_id": "a1b2c3d4-...", "step_name": "validate_input" },
  "version": 1,
  "created_at": "2026-03-22T10:00:01.000Z",
  "input": { "name": "test_user" },
  "ending": {
    "end_state": "COMPLETED",
    "reason": "",
    "ended_at": "2026-03-22T10:00:03.000Z",
    "output": { "name": "test_user", "validated_at": "2026-03-22T10:00:03Z" },
    "duration": 2
  }
}
```

---

## Defining a Flow

Flows are defined in code inside the `flow/` package. Each flow is a `Flow` struct containing an ordered list of `Step` structs. One file per flow.

```go
// flow/order_processing.go
package flow

import (
    "context"
)

var OrderProcessing = Flow{
    Name: "order_processing",
    Steps: []Step{
        {
            Name: "validate_payment",
            Execute: func(ctx context.Context, input map[string]any) (map[string]any, error) {
                orderID := input["order_id"].(string)
                // ... validate payment logic ...
                return map[string]any{"order_id": orderID, "payment_valid": true}, nil
            },
        },
        {
            Name: "reserve_inventory",
            Cleanup: func(ctx context.Context, input map[string]any) error {
                // release any previously reserved stock before retrying
                return nil
            },
            Execute: func(ctx context.Context, input map[string]any) (map[string]any, error) {
                // ... reserve stock ...
                return map[string]any{"reserved": true}, nil
            },
        },
        {
            Name: "send_confirmation",
            Execute: func(ctx context.Context, input map[string]any) (map[string]any, error) {
                // ... send email/notification ...
                return map[string]any{"notified": true}, nil
            },
        },
    },
}
```

Then wire it up in `cmd/flowx/main.go`:

```go
import "flowx/flow"

exec := executor.NewExecutor(logger, stepRunRepo, flow.OrderProcessing)
```

That's it. FlowX handles execution, retries, logging, and alerting.

---

## Use Cases

FlowX is a general-purpose sequential step runner. Some examples of what you can plug in:

| Flow | Steps |
|---|---|
| **Order Processing** | Validate payment, reserve inventory, generate invoice, send confirmation |
| **User Onboarding** | Create account, provision resources, send welcome email, schedule follow-up |
| **Data Pipeline** | Fetch from API, transform records, load into warehouse, notify stakeholders |
| **CI/CD Post-Deploy** | Run smoke tests, warm caches, toggle feature flags, post to Slack |
| **Media Processing** | Download asset, transcode video, generate thumbnails, upload to CDN |
| **Report Generation** | Query databases, aggregate metrics, render PDF, email recipients |

Each step receives the previous step's output as input, so data flows naturally through the pipeline without external orchestration.

---

## Configuration

FlowX uses a YAML config file with sensible defaults. Pass a custom file with `-c`:

```bash
./flowx -c config.yml
```

### Default Configuration

```yaml
application: "flowx"

logger:
  encoding: "logfmt"   # logfmt or json
  level: "debug"        # debug, info, warn, error

listen: ":3625"
prefix: "/flowx"
is_prod_mode: false

mongo:
  uri: "mongodb://localhost:27017"

queue:
  size: 50              # buffered channel capacity
  workers: 5            # number of concurrent worker goroutines

slack:
  webhook_url: "https://hooks.slack.com/services/your/webhook/url"
  send_alert_in_dev: false
```

| Key | Description |
|---|---|
| `queue.size` | Max runs that can be buffered before producers block |
| `queue.workers` | Number of goroutines consuming from the queue |
| `is_prod_mode` | Enables Slack alerts; disables config printing on boot |
| `slack.send_alert_in_dev` | Force Slack alerts even when `is_prod_mode` is false |

---

## Resumability

If FlowX crashes or restarts mid-run, it does **not** start over. On the next boot:

1. All runs where `is_completed: false` are loaded from MongoDB.
2. For each run, the executor queries `step_runs` for the last recorded step.
3. If the last step was `COMPLETED`, execution resumes from the **next** step using that step's output.
4. If the last step was `FAILED` (or started but never finished), execution resumes from **that same step** using its original input — so the `Cleanup` phase can undo partial work before re-executing.

This makes FlowX safe to run in environments where processes may be killed at any time.

---

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/{prefix}/v1/health` | Returns `200` if MongoDB is reachable, `503` otherwise |
| `POST` | `/{prefix}/v1/runs` | Creates a new run and enqueues it for execution |

### Create a Run

```bash
curl -X POST http://localhost:3625/flowx/v1/runs \
  -H "Content-Type: application/json" \
  -d '{"name": "test_user"}'
```

**Response** (`201 Created`):

```json
{
  "message": "Run Created Successfully!",
  "run_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

---

## Running

### Local

```bash
# build and run
make run

# or run directly without building
make dev
```

### Docker

```bash
# build image
make docker

# build and run on port 3625
make docker-run
```

### Prerequisites

- Go 1.26+
- MongoDB instance
- (Optional) Slack incoming webhook URL

---

## Make Targets

```
make build        Build binary to .bin/
make run          Build and run with config.yml
make dev          Run via go run (no binary)
make test         Run tests with race detector
make test-cover   Run tests + HTML coverage report
make lint         Run go vet + gofmt check
make fmt          Format all Go files
make tidy         Run go mod tidy
make docker       Build Docker image
make docker-run   Build and run container
make clean        Remove build artifacts
```

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
