# AGENTS.md

## Project Overview

Go-based automation for deploying and managing the Debezium Platform on Kubernetes. Uses **mage** (Go task runner) as the primary build system and **Taskfile** as a secondary YAML-based task runner for Kind cluster workflows.

Module path: `dbz-mage` (imported as `dbz-mage/ko/...`)

## Build & Run

```bash
# List available mage targets
mage -l

# Run a mage target
mage cluster:recreate
mage helm:all
mage helm:dbzOperator
mage helm:platform
mage helm:diff
mage data:mongo
mage data:pg
mage scenario:mongodbRs

# Taskfile (less common, mostly Kind + legacy workflows)
task deploy
task create-local-cluster
task load-postgresql-data
```

**There is no `go build` target** — the `magefile.go` uses `//go:build mage` tag so it only compiles via `mage`. The `main.go` is a standalone entry point that runs MongoDB setup directly (not the primary workflow).

**There are no test files, no lint config, no CI configs** in this repository.

## Architecture

```
main.go                  # Standalone entry point (MongoDB setup)
magefile.go              # Primary task runner (mage targets)
Taskfile.yaml            # Secondary task runner (Kind + deploy)

ko/                      # Core library (import path: dbz-mage/ko/...)
├── cluster/             # K8s cluster lifecycle (Kind, K3s)
├── runner/              # Command execution (Local, SSH via Runner interface)
├── k8s/                 # Kubernetes client-go wrapper
├── source/              # Data sources (Source interface → MongoDB, PostgreSQL)
├── automation/          # Env loading (.env), command exec helpers
├── dmp/                 # Debezium Platform HTTP API client + resource resolver
└── scenarios/           # DMP scenario management (legacy path)

deploy/
├── helmfile.yaml.gotmpl # Helmfile defining all releases
├── charts/              # Custom Helm charts (kafka-cluster, postgresql-cluster, etc.)
├── values/              # Per-component, per-environment values files
├── clusters/            # Kind cluster configs
└── images/              # Dockerfiles (kafka-connect)

resources/
├── data/                # SQL seed scripts, MongoDB JS seed scripts
└── *.json               # Standalone DMP payload definitions

ko/scenarios/
├── common/              # Shared DMP payloads (connections, destinations)
└── <scenario-name>/     # Per-scenario yaml manifest + JSON payloads
    ├── scenario.yaml
    └── payloads/
```

## Key Interfaces

- **`cluster.Cluster`** — `Recreate`, `Delete`, `UseContext` — implemented by `Kind` and `K3s`
- **`runner.Runner`** — `Run`, `Output`, `CopyFrom` — implemented by `Local` and `SSH`
- **`source.Source`** — `Wait`, `Setup`, `Populate`, `Reset`, `Close` — implemented by `MongoDB` and `PostgreSQL`
- **`dmp.Resource`** — `GetKey`, `GetFile`, `GetRefs`, `GetType` — implemented by connection/source/destination/pipeline resource types

## Environment & Configuration

**All configuration is via environment variables**, loaded from `.env` by `automation.LoadEnv()` via `godotenv`. Every mage target calls this first.

### Critical env vars

| Variable | Purpose | Default |
|---|---|---|
| `DBZ_ENV` | Deployment environment | `local` |
| `DBZ_VERSION` | Debezium helm chart version | required |
| `DBZ_NAMESPACE` | Debezium namespace | `dmp` |
| `CLUSTER_TYPE` | Cluster provider (`kind` or `k3s`) | `kind` |
| `DMP_RESOURCE_PREFIX` / `DMP_ENVIRONMENT` | Resource naming prefix | used in JSON payloads |
| `KAFKA_DMP_BOOTSTRAP_SERVERS` | Kafka bootstrap servers | used in JSON payloads |
| `MONGODB_*` / `POSTGRESQL_*` | Database connection config | see `client.go` defaults |
| `LOG_LEVEL` | Logging level | `info` |

### JSON payload environment expansion

JSON payload files use `${ENV_VAR}` syntax — `os.ExpandEnv()` replaces these at load time. This applies to files in `ko/scenarios/common/` and `ko/scenarios/<name>/payloads/`.

## DMP Resource Management Flow

The `dmp.ResourceResolver` (in `ko/dmp/scenarios_common.go`) manages Debezium Platform resources idempotently:

1. **`EnsureCommonConnections`** — Loads JSON from `ko/scenarios/common/connections/<key>.json`, validates via DMP API, creates if not exists
2. **`EnsureCommonDestinations`** — Loads from `ko/scenarios/common/destinations/<key>.json`, links to resolved connection ID
3. **`EnsureScenarioSources`** — Loads from `ko/scenarios/<scenario>/payloads/`, injects connection ref
4. **`EnsureScenarioPipelines`** — Loads from `ko/scenarios/<scenario>/payloads/`, injects source + destination refs

All operations do FindByName first — if a resource with the deterministic name already exists, it is reused rather than recreated.

### Resource naming convention

Resources get deterministic names derived from their JSON payload's `name` field (with env var expansion). The pattern is typically: `${DMP_RESOURCE_PREFIX}-${DMP_ENVIRONMENT}-<resource-type>`.

## Helm Deployments

Controlled by `deploy/helmfile.yaml.gotmpl`. Deployments are ordered by `needs` dependencies and selectable by labels:

```bash
helmfile --file deploy/helmfile.yaml.gotmpl --selector app=strimzi-cluster-operator apply
helmfile --file deploy/helmfile.yaml.gotmpl --selector infra=true apply
```

Comment-out blocks in the helmfile (like `kafka-connect`, `cdc-dashboard`, `apicurio-registry`) indicate components that are optional or retired.

The `mage helm:all` applies releases in dependency order using separate helmfile invocations with different selectors.

## Code Patterns & Conventions

- **`NewFromEnv()`** is the standard factory pattern — reads env vars, builds config, calls `New()`
- **`automation.Env(name, fallback)`** provides env vars with defaults; `RequiredEnv(name)` fails if missing
- **`automation.ExpandPath(path)`** handles `~/` prefix and `${VAR}` expansion for paths
- **Structured logging** uses `log/slog` via `slog.Default()`; each component has a `logger()` method that falls back to the default logger
- **Error wrapping** uses `fmt.Errorf("...: %w", err)` consistently
- **No comments in code** unless they are Go doc comments on exported functions

## Gotchas

1. **Module name mismatch**: The Go module is `dbz-mage` but the directory is `oneint-k8s-mgt`. Imports use `dbz-mage/ko/...`.
2. **Two task systems**: Both `magefile.go` and `Taskfile.yaml` exist. The magefile is the primary tool; Taskfile is mostly legacy for Kind-based local development.
3. **Hardcoded DMP base URL**: `ko/dmp/http_client.go` hardcodes `http://platform.debezium.local`. Override by changing the `BaseURL` field on the client struct.
4. **Mage build tag**: The magefile has `//go:build mage` — it won't compile with regular `go build`. Use `mage` to run targets.
5. **No idempotent apply for pipelines**: The `scenario:mongodbRs` mage target goes through `mongoRsBasic()` which uses `dmp.ResourceResolver` — idempotent. But the `ko/scenarios/service.go` path does NOT do FindByName (marked as TODO), so repeated calls there would create duplicates.
6. **JSON payloads expect `name` field**: All DMP JSON payloads MUST have a `"name"` key, or the loader will error.
7. **Commented-out code**: Both `magefile.go` and `helmfile.yaml.gotmpl` contain commented-out sections for retired/optional components (Apicurio, CDC dashboard, Kafka Connect). Don't uncomment without understanding the dependencies.
8. **`.env` file required**: Both mage targets and Taskfile tasks call `automation.LoadEnv()` which loads `.env`. Without it, required env vars like `DBZ_VERSION` will be missing.
