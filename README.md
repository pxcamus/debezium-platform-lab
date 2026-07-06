# debezium-platform-lab

Go-based automation for deploying and managing the [Debezium Platform](https://debezium.io/documentation/reference/stable/operations/debezium-platform.html) on Kubernetes.

It provisions a full change-data-capture (CDC) stack — Kafka (Strimzi), PostgreSQL (CloudNativePG), MongoDB, SQL Server, the Debezium Operator and the Debezium Platform — seeds demo databases, and drives the Debezium Platform HTTP API to create connections, sources, destinations and pipelines idempotently.

The primary task runner is [**mage**](https://magefile.org/) (Go); Helm releases are orchestrated with [**helmfile**](https://helmfile.readthedocs.io/).

> **Status:** internal automation shared for reference. Defaults target a local [Kind](https://kind.sigs.k8s.io/) cluster. Passwords in the Helm charts and `.env.example` are non-secret demo values.

---

## Prerequisites

| Tool | Purpose |
|---|---|
| [Go](https://go.dev/) 1.26+ | build/run mage targets |
| [mage](https://magefile.org/#installation) | primary task runner |
| [helmfile](https://helmfile.readthedocs.io/en/latest/#installation) + [helm](https://helm.sh/) | chart orchestration (helm-diff plugin recommended) |
| [kind](https://kind.sigs.k8s.io/) | local Kubernetes cluster |
| [kubectl](https://kubernetes.io/docs/tasks/tools/) | cluster access |
| [Docker](https://www.docker.com/) | container runtime for Kind |

Optional: [Task](https://taskfile.dev/) (legacy Kind workflow), [kubeconform](https://github.com/yannh/kubeconform) (used by CI/helm validation).

---

## Quick start (local Kind)

```bash
# 1. Configure environment
cp .env.example .env
#    Ensure DBZ_ENV=local and CLUSTER_TYPE=kind in .env

# 2. Create the local cluster
kind create cluster --name dmp --config deploy/clusters/kind/kind-ingress.yaml

# 3. Deploy the full stack (infra → operator → platform), ordered by dependency
mage helm:all

# 4. Seed the demo databases
mage data:pg       # PostgreSQL ecommerce schema + data
mage data:mongo    # MongoDB ecommerce collections + data

# 5. Create Debezium Platform resources for a scenario
mage scenario:mongodbRs
```

List every available target with:

```bash
mage -l
```

---

## Configuration

**All configuration is via environment variables**, loaded from `.env` (git-ignored). Start from [`.env.example`](.env.example), which documents every variable. The most important ones:

| Variable | Purpose | Default |
|---|---|---|
| `DBZ_VERSION` | Debezium Helm chart version | **required** |
| `DBZ_ENV` | Deployment environment; selects `deploy/values/<component>/<DBZ_ENV>.yaml` | `local` |
| `DBZ_NAMESPACE` | Debezium Platform namespace | `dmp` |
| `CLUSTER_TYPE` | Cluster provider: `kind` or `k3s` | `kind` |
| `DMP_RESOURCE_PREFIX` / `DMP_ENVIRONMENT` | Prefix for deterministic DMP resource names | — |
| `KAFKA_DMP_BOOTSTRAP_SERVERS` | Kafka bootstrap for DMP payloads | — |
| `MONGODB_*` / `POSTGRESQL_*` / `SQLSERVER_*` | DB connection config | see `.env.example` |
| `LOG_LEVEL` | `debug` \| `info` \| `warn` \| `error` | `info` |

Known `DBZ_ENV` values in this repo: `local`, `erebor` (homelab), `aws`, `hetzner`. Each has a matching values file under `deploy/values/<component>/`.

DMP JSON payloads use `${ENV_VAR}` syntax that is expanded from the environment at load time.

---

## Common tasks

```bash
# Cluster (remote k3s lifecycle; requires CLUSTER_TYPE=k3s + K3S_* vars)
mage cluster:recreate

# Helm
mage helm:all            # apply everything in dependency order
mage helm:infra          # infra releases only (label infra=true)
mage helm:dbzOperator    # Debezium Operator only
mage helm:platform       # Debezium Platform only
mage helm:diff           # preview pending changes
mage helm:platformDestroy / mage helm:allDestroy

# Data seeding
mage data:mongo / mage data:resetMongo
mage data:pg    / mage data:resetPg

# DMP scenarios
mage scenario:mongodbRs   # MongoDB replica-set → Kafka
mage scenario:all         # every wired scenario
```

> **Note:** only the MongoDB replica-set scenario (`scenario:mongodbRs`) is wired up today, so `scenario:all` currently runs just that one. The `postgres-basic` and `sqlserver-basic` directories under `ko/scenarios/` contain payloads but are not yet exposed as targets.

Helm releases are selectable by label directly, too:

```bash
helmfile --file deploy/helmfile.yaml.gotmpl --selector infra=true apply
helmfile --file deploy/helmfile.yaml.gotmpl --selector app=debezium-platform apply
```

---

## Validate Helm charts (no cluster required)

```bash
scripts/validate-helm.sh
```

Runs `helm lint` / `helm template` per chart and `helmfile lint` / `template` piped through `kubeconform`. This is also enforced in CI ([`.github/workflows/helm-validation.yaml`](.github/workflows/helm-validation.yaml)) on changes under `deploy/`.

---

## Repository layout

```
main.go                  # Standalone entry point (MongoDB setup)
magefile.go              # Primary task runner (mage targets)
Taskfile.yaml            # Secondary task runner (legacy Kind workflow)

ko/                      # Core Go library (module: dbz-mage, imports dbz-mage/ko/...)
├── cluster/             # K8s cluster lifecycle (Kind, K3s)
├── runner/              # Command execution (Local, SSH)
├── source/              # Data sources (MongoDB, PostgreSQL)
├── automation/          # Env loading (.env), exec helpers
├── dmp/                 # Debezium Platform HTTP API client + resource resolver
└── scenarios/           # DMP scenario manifests + JSON payloads
    ├── common/          # Shared connections, destinations, transforms
    └── <scenario>/      # Per-scenario scenario.yaml + payloads/

deploy/
├── helmfile.yaml.gotmpl # All Helm releases, ordered by dependency
├── charts/              # Custom charts (kafka-cluster, postgresql-cluster, mssql, ...)
├── values/              # Per-component, per-environment values
├── clusters/            # Kind cluster configs
└── environment/         # versions.env — shared version pins

resources/               # SQL / MongoDB seed scripts, standalone DMP payloads
certs/                   # Optional TLS (see below)
scripts/validate-helm.sh # Offline chart validation
```

`collections/` (Posting HTTP collections for the DMP API) is auxiliary/reference material. `examples/`, `docs/`, `thorin/` are external symlinks and are git-ignored.

---

## Optional: HTTPS via cert-manager + Gandi (homelab only)

The files under [`certs/`](certs/) and the `cert-manager-webhook-gandi` release in the helmfile expose platform services over HTTPS using Let's Encrypt with a DNS-01 challenge solved through the [Gandi](https://www.gandi.net/) DNS API. **This is entirely optional and specific to a private homelab (`DBZ_ENV=erebor`, domain `*.erebor.1int.io`); a local Kind demo does not need it.**

The webhook release is gated on `DBZ_ENV=erebor`, so it is not installed for `local`. To use it in your own environment:

1. Set your ACME registration email in `certs/letsencrypt-*-clusterissuer.yaml` (currently a placeholder).
2. Provide your Gandi Personal Access Token. It is **not committed** — create the Secret out of band:
   ```bash
   kubectl -n cert-manager create secret generic gandi-credentials \
     --from-literal=pat="$GANDI_PAT"
   ```
   (`deploy/values/cert-manager-webhook-gandi/erebor.yaml` also carries a `gandiPat` placeholder for the webhook chart itself.)
3. Adjust the domains in `certs/*.yaml` to your own, then apply the ClusterIssuers and Certificate:
   ```bash
   kubectl apply -f certs/
   ```

---

## Notes & gotchas

- **Module name:** the Go module is `dbz-mage` (unchanged by the repo name). Imports use `dbz-mage/ko/...`.
- **No `go build` target:** `magefile.go` uses the `//go:build mage` tag and only compiles via `mage`. `main.go` is a separate standalone entry point.
- **Two task runners:** `magefile.go` is primary and current; `Taskfile.yaml` is a legacy Kind workflow and some of its paths predate the `deploy/` reorg.
- **DMP base URL:** `ko/dmp/http_client.go` defaults to `http://platform.debezium.local`; override via `DMP_BASE_URL`.
- **Idempotent DMP resources:** the resolver does find-by-name before create, so re-running a scenario reuses existing resources.
- **Commented-out releases:** `helmfile.yaml.gotmpl` and `magefile.go` contain disabled blocks for optional/retired components (Apicurio, CDC dashboard, Kafka Connect). Don't enable without checking the dependency chain.
