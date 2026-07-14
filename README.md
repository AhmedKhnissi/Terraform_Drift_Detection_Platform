# driftdetect — Terraform Drift Detection Platform

`driftdetect` continuously compares your **Terraform state** against the **actual
cloud infrastructure** and reports configuration drift — without ever running
`terraform plan` or `terraform apply`. It reads state passively, queries cloud
provider APIs directly, normalizes both into a common model, and surfaces
differences via a **CLI**, **JSON** output, and a **web dashboard**.

This build ships the **AWS** provider (EC2, S3, RDS, IAM) with an extensible
provider interface so additional clouds and resource types drop in cleanly.

## Features

- **Passive drift detection** — no Terraform binary, no plan/apply.
- **Multi-source state** — local `.tfstate` files or an S3 remote backend
  (`s3://bucket/key`).
- **Cloud-agnostic core** — a `CloudProvider` / `ResourceFetcher` interface; AWS
  implemented now, Azure/GCP follow behind the same contract.
- **Drift categories** — `deleted`, `modified` (attributes), `tag_change`
  (added/removed/changed), and optional `orphaned` detection.
- **Per-type attribute rules** — only meaningful attributes are compared, so
  computed/read-only fields don't cause false positives.
- **On-demand & scheduled scans** — one-shot `scan` or recurring `schedule` /
  `serve --schedule`.
- **History & dashboard** — results persist to SQLite and are browsable via a
  simple web UI and JSON API.
- **Single static binary** — pure-Go dependencies (no cgo), runs on Windows,
  Linux, and macOS.

## Install

```bash
go build -o driftdetect .
```

(Requires Go 1.22+.)

## Configure

Copy and edit the example config:

```bash
cp config.example.yaml config.yaml
```

```yaml
aws:
  region: us-east-1
  profile: default          # optional shared profile
state:
  source: ./terraform.tfstate   # or s3://my-bucket/terraform.tfstate
drift:
  compare_attributes: true
  compare_tags: true
  detect_orphans: false
schedule:
  spec: ""                   # e.g. "*/15 * * * *"
storage:
  path: ./driftdetect.db
web:
  addr: ":8080"
```

AWS credentials are resolved through the standard AWS credential chain (env
vars, `~/.aws/credentials`, IAM role, …).

## Usage

### Scan on demand

```bash
# Human-readable table
driftdetect scan --state ./terraform.tfstate

# JSON to stdout
driftdetect scan --format json

# Save the result to history for the dashboard
driftdetect scan --save

# JSON to a file
driftdetect scan --out report.json
```

### View history & reports

```bash
driftdetect history
driftdetect report --scan <scan-id>
driftdetect report --scan <scan-id> --format json
```

### Web dashboard

```bash
driftdetect serve
# open http://localhost:8080
```

With a built-in scheduler and on-demand trigger:

```bash
driftdetect serve --schedule "*/15 * * * *"
# POST /api/scan triggers an on-demand scan from the dashboard
```

### Scheduled daemon

```bash
# Every 15 minutes
driftdetect schedule --every 15m

# Raw cron
driftdetect schedule --spec "0 * * * *"
```

## Architecture

```
  Terraform state ──▶ state (parse + source) ──▶ expected ResourceState
                                                        │
  Cloud APIs ───────▶ cloud/aws (ResourceFetcher) ──▶ actual ResourceState
                                                        │
                                              drift.Compare ──▶ DriftReport
                                                        │
                                    ┌───────────────────┼───────────────────┐
                              output (CLI/JSON)    storage (SQLite)    web (dashboard + API)
```

- `internal/model` — cloud-agnostic `ResourceState` and `DriftReport`.
- `internal/state` — Terraform state parsing (local + S3) into `ResourceState`.
- `internal/cloud` + `internal/cloud/aws` — provider interface + AWS fetchers.
- `internal/drift` — comparator + per-type attribute rules.
- `internal/engine` — orchestrates a full scan (timing + scan id).
- `internal/storage` — SQLite persistence.
- `internal/schedule` — cron scheduler.
- `internal/web` — dashboard + JSON API (stdlib `net/http`, embedded templates).
- `cmd/driftdetect` — CLI commands.

## Extending

### Add a new AWS resource type

1. Create a fetcher in `internal/cloud/aws/` (e.g. `elb.go`) implementing
   `ResourceFetcher.Fetch(ctx, expected) (model.ResourceState, error)`, returning
   `cloud.ErrNotFound` when the resource is gone.
2. Register it in `buildRegistry` (`registry.go`) keyed by its Terraform type
   (e.g. `"aws_lb"`).
3. Add the comparable attributes to `driftAttributes` in `internal/drift/rules.go`.

No changes to the comparator, parser, storage, or CLI are required.

### Add a new cloud (Azure, GCP, …)

Implement `cloud.CloudProvider` (and internal `ResourceFetcher`s) the same way
`internal/cloud/aws` does, then wire it into the command layer in
`cmd/driftdetect/app.go`.

## Limitations (first build)

- AWS only; Azure/GCP are future work behind the same interface.
- No authentication on the web dashboard (assume trusted network / localhost).
- No remediation, alerting, or retention policies for stored scans.
