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
go mod tidy          # resolve dependencies (aws-sdk-go-v2, cobra, yaml, ...)
go build -o driftdetect .
```

Requires Go 1.25+ (dependencies are pinned in `go.mod` / `go.sum`; `go mod tidy`
only needs network if those drift).

> The Terraform working directory `.terraform/` (downloaded provider binaries,
> including a ~700 MB `terraform-provider-aws` executable) is gitignored — never
> commit it, or pushes to GitHub will be rejected for exceeding the 100 MB file
> limit.

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

Results are available in more than just the terminal: full per-finding detail
(expected vs. actual, attribute, message) is in the `--format json` output, and
saved scans (`--save`) are browsable in the web dashboard and via the `history`
/ `report` commands below.

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

## Example: detect drift on an S3 bucket

This repository ships a `main.tf` that provisions a demo bucket
(`drift-demo-<account-id>`) with versioning, SSE (AES256), and a public-access
block. Drift is detected **passively** — no `terraform plan`/`apply` — by
comparing Terraform **state** against the live AWS API.

1. Create the infrastructure and the state file (credentials come from the AWS
   default credential chain, `~/.aws/credentials`, profile `default`):

   ```bash
   terraform init
   terraform apply
   ```

2. Scan for drift. The detector reads `./terraform.tfstate` (overridable via
   `state.source` in `config.yaml` or `--state`) and queries AWS directly:

   ```bash
   driftdetect scan
   ```

   Right after `apply` the bucket matches state exactly — **zero drift**:

   ```
   Resources scanned: 1   Drift detected: 0
   Summary:
     Missing resources in cloud (deleted): 0
     Extra resources in cloud (orphaned):   0
     Attribute changes (modified):          0
     Tag changes:                           0
     Total findings:                        0
   No drift detected. Infrastructure matches Terraform state.
   ```

3. Introduce drift in the AWS console — e.g. add a tag `test = ahmed` to the
   bucket — then scan again:

   ```
   Scan 3fed2261ded94ba3  @ 2026-07-15 19:28:30  (780 ms)
   Resources scanned: 1   Drift detected: 1
   Summary:
     Missing resources in cloud (deleted): 0
     Extra resources in cloud (orphaned):   0
     Attribute changes (modified):          0
     Tag changes:                           1
     Total findings:                        1
   TYPE           NAME        ID                       DRIFT       DETAIL            EXPECTED -> ACTUAL
   aws_s3_bucket  drift_demo  drift-demo-796973496507  tag_change  tag "test" added  <nil> -> ahmed
   ```

   The summary reports, per category:
   - **Missing resources in cloud** (`deleted`) — declared in state but gone from AWS.
   - **Extra resources in cloud** (`orphaned`) — in AWS but not declared in state
     (only counted when `drift.detect_orphans: true`).
   - **Attribute changes** (`modified`) — a compared attribute changed value.
   - **Tag changes** — a tag was added, removed, or changed.
   - **Total findings** — the total number of drift items.

> Note: the demo bucket is compared by **tags** only. Versioning and encryption
> are configured through dedicated `aws_s3_bucket_*` resources that this build
> does not yet fetch, so their changes are not reported as drift unless the S3
> fetcher is extended. Resource types without a registered fetcher (e.g.
> `aws_s3_bucket_versioning`, `aws_s3_bucket_server_side_encryption_configuration`,
> `aws_s3_bucket_public_access_block`) are ignored, not reported as deletions.

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
