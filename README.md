# FlowX

A lightweight, configurable workflow execution engine built in Go. FlowX picks up workflow records from MongoDB, fans them out to a pool of concurrent workers, and runs each workflow's tasks sequentially — with automatic retries, resumability, and Slack alerting baked in.

There is no fixed use case. Define any sequence of tasks, point FlowX at your database, and it will execute them reliably.

---

## Architecture

```
MongoDB ──► Queue (buffered channel) ──► Worker Pool ──► Processor ──► Tasks
                                                              │
                                                         Task Logs (MongoDB)
                                                              │
                                                     Slack Alerts (on failure)
```

### How It Works

1. **Startup** — FlowX connects to MongoDB, spins up _N_ workers, and loads every incomplete workflow (`is_completed: false`) into a buffered channel.
2. **Workers** — Each worker is a goroutine polling the channel. When a workflow arrives, the worker hands it to the Processor.
3. **Processor** — Checks the `tasklogs` collection for the last recorded task of that workflow. If none exists, the full task list runs from scratch. If a previous run was interrupted, execution resumes from the point of failure.
4. **Task Execution** — Each task goes through two optional phases:
   - **Cleanup** — Tear down or reset state from a prior partial run.
   - **Execute** — The actual work. Receives `map[string]any` input, returns `map[string]any` output.
   
   The output of one task becomes the input of the next, forming a pipeline.
5. **Retries** — A failing task is retried up to **3 times** with a **1-minute** backoff between attempts. If all retries are exhausted, the task is marked `FAILED` and a Slack alert fires.
6. **Completion** — Once every task succeeds, the workflow record is marked complete in MongoDB.
7. **Shutdown** — On `SIGINT`/`SIGTERM`, workers finish their current task, the HTTP server drains with a 5-second timeout, and the MongoDB connection is closed.

---

## Project Structure

```
.
├── cmd/flowx/              Entrypoint — config loading, logger, server init
├── config/                 Config struct, defaults, validation
├── errors/                 Typed application errors (Kind-based)
├── http/
│   ├── handlers/           HTTP handlers (health check)
│   ├── middlewares/         Request logging middleware
│   ├── response/           JSON response helpers
│   └── server.go           Chi router, graceful shutdown
├── models/
│   ├── tasklog/            TaskLog data model
│   └── workflow/           Workflow + Task data models
├── notifications/
│   ├── alerter.go          Alerter interface + factory
│   ├── slack.go            Slack webhook implementation
│   └── discard.go          No-op alerter for dev mode
├── repositories/mongodb/   MongoDB repositories (workflows, tasklogs)
├── services/
│   ├── health/             Health check service (MongoDB ping)
│   ├── processor/          Task execution engine with retry + resume
│   ├── queue/              Buffered channel queue + worker pool
│   └── workflow/           Workflow definition service
└── utils/
    ├── constants/          Shared constants
    └── helpers/            Validation, time, HTTP utilities
```

---

## Data Flow

### Workflow Record (`workflows` collection)

```json
{
  "_id": "order-12345",
  "created_at": "2026-03-22T10:00:00Z",
  "input": { "order_id": "12345", "user_email": "user@example.com" },
  "is_completed": false,
  "completed_at": "",
  "last_task_status": false
}
```

Insert a document like this into MongoDB and FlowX will pick it up on the next startup (or enqueue it via code at runtime).

### Task Log Record (`tasklogs` collection)

Each task execution is recorded with its input, output, duration, and final state:

```json
{
  "_id": { "workflow_id": "order-12345", "task_name": "validate-payment" },
  "version": 1,
  "created_at": "2026-03-22T10:00:01Z",
  "input": { "order_id": "12345" },
  "ending": {
    "end_state": "COMPLETED",
    "reason": "",
    "ended_at": "2026-03-22T10:00:03Z",
    "output": { "payment_valid": true },
    "duration": 2
  }
}
```

---

## Defining a Workflow

A workflow is an ordered list of `Task` structs. Each task has a `Name`, optional `Cleanup` function, and optional `Execute` function.

```go
package workflow

import (
    "context"
    models "flowx/models/workflow"
)

func (s *WorkflowService) GetMyWorkflow() models.Workflow {
    return models.Workflow{
        Name: "order-processing",
        Tasks: []models.Task{
            {
                Name: "validate-payment",
                Execute: func(ctx context.Context, input map[string]any) (map[string]any, error) {
                    orderID := input["order_id"].(string)
                    // ... validate payment logic ...
                    return map[string]any{"order_id": orderID, "payment_valid": true}, nil
                },
            },
            {
                Name: "reserve-inventory",
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
                Name: "send-confirmation",
                Execute: func(ctx context.Context, input map[string]any) (map[string]any, error) {
                    // ... send email/notification ...
                    return map[string]any{"notified": true}, nil
                },
            },
        },
    }
}
```

Then wire it up in `cmd/flowx/main.go`:

```go
workflow := workflowSvc.GetMyWorkflow()
processor := processor.NewProcessor(logger, tasklogRepo, workflow)
```

That's it. FlowX handles execution, retries, logging, and alerting.

---

## Use Cases

FlowX is a general-purpose sequential workflow runner. Some examples of what you can plug in:

| Workflow | Tasks |
|---|---|
| **Order Processing** | Validate payment, reserve inventory, generate invoice, send confirmation |
| **User Onboarding** | Create account, provision resources, send welcome email, schedule follow-up |
| **Data Pipeline** | Fetch from API, transform records, load into warehouse, notify stakeholders |
| **CI/CD Post-Deploy** | Run smoke tests, warm caches, toggle feature flags, post to Slack |
| **Media Processing** | Download asset, transcode video, generate thumbnails, upload to CDN |
| **Report Generation** | Query databases, aggregate metrics, render PDF, email recipients |

Each task receives the previous task's output as input, so data flows naturally through the pipeline without external orchestration.

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
| `queue.size` | Max workflows that can be buffered before producers block |
| `queue.workers` | Number of goroutines consuming from the queue |
| `is_prod_mode` | Enables Slack alerts; disables config printing on boot |
| `slack.send_alert_in_dev` | Force Slack alerts even when `is_prod_mode` is false |

---

## Resumability

If FlowX crashes or restarts mid-workflow, it does **not** start over. On the next boot:

1. All workflows where `is_completed: false` are loaded from MongoDB.
2. For each workflow, the processor queries `tasklogs` for the last recorded task.
3. If the last task was `COMPLETED`, execution resumes from the **next** task using that task's output.
4. If the last task was `FAILED` (or started but never finished), execution resumes from **that same task** using its original input — so the `Cleanup` phase can undo partial work before re-executing.

This makes FlowX safe to run in environments where processes may be killed at any time.

---

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/{prefix}/v1/health` | Returns `200` if MongoDB is reachable, `503` otherwise |

Default: `GET /flowx/v1/health`

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
make build        Build binary to bin/
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

