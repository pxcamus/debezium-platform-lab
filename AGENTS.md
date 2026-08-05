# AGENTS.md

## Project Overview

Helm charts and helmfile releases for deploying and managing the Debezium Platform on Kubernetes,
plus the JSON payloads that describe Debezium Platform pipelines.

**There is no application code in this repository.** The Go library, the `mage` task runner and the
`dmp-lab` CLI were removed while the toolchain moves to [`just`](https://just.systems/); everything
here is now YAML, JSON, shell and documentation. Do not reintroduce Go without being asked.

## Build & Run

```bash
# Apply releases in dependency order (what `mage helm:all` used to run)
export HELMFILE="helmfile --file deploy/helmfile.yaml.gotmpl"
$HELMFILE --selector app=strimzi-cluster-operator    apply
$HELMFILE --selector app=cnpg-operator               apply
$HELMFILE --selector app=mongodb-community-operator  apply
$HELMFILE --selector infra=true                      apply --skip-diff-on-install
$HELMFILE --selector app=debezium-operator           apply
$HELMFILE --selector app=debezium-platform           apply

# Preview pending changes
helmfile --file deploy/helmfile.yaml.gotmpl diff

# Validate every chart without a cluster
scripts/validate-helm.sh
```

`scripts/with-env.sh` wraps a command with `.env` + `versions.env` resolved, which is what the mage
targets did before running a child process.

CI is two GitHub Actions workflows: `helm-validation.yaml` (chart lint/template/kubeconform on
changes under `deploy/`) and `docs.yaml` (MkDocs build and deploy to lab.1int.io).

## Architecture

```
deploy/
├── helmfile.yaml.gotmpl # All releases, ordered by dependency, selectable by label
├── charts/              # Custom Helm charts (kafka-cluster, postgresql-cluster, mssql, ...)
├── values/              # Per-component, per-environment values files
├── clusters/            # Kind cluster configs
├── environment/         # versions.env — shared version pins
└── images/              # Dockerfiles (kafka-connect)

data-pipelines/config/   # DMP scenario manifests + JSON payloads (data, not code)
├── common/              # Shared DMP payloads (connections, destinations, transforms)
└── <scenario-name>/
    ├── scenario.yaml
    └── payloads/

resources/
├── data/                # SQL seed scripts, MongoDB JS seed scripts
└── *.json               # Standalone DMP payload definitions

certs/                   # Optional homelab TLS manifests (ClusterIssuers + Certificate)
docs/                    # MkDocs sources for lab.1int.io
scripts/
├── validate-helm.sh     # Offline chart validation
├── with-env.sh          # Runs a command with .env + versions.env resolved
├── vendor-keycloak-operator.sh
└── lab/                 # Lab VM provisioning
```

`data-pipelines/config/` was `ko/scenarios/` while the Go library lived in `ko/`; older commits,
issues and the published tags still use the old path.

## Environment & Configuration

**All configuration is via environment variables**, layered and first-wins: `.env` (git-ignored —
secrets and host-specific overrides) is read first, then `deploy/environment/versions.env` (shared,
non-sensitive version pins) fills the gaps. Both files are optional; a checkout with only
`versions.env` works, which is how CI runs. **Version pins live in `versions.env` only** — never
duplicate them into `.env` or `.env.example`.

`.env.example` is deliberately minimal: it carries only variables that have no default anywhere, or
whose default is wrong when run from a workstation rather than in-cluster. The exhaustive list
belongs in `docs/reference/environment.md`.

### Critical env vars

| Variable | Purpose | Default |
|---|---|---|
| `DPL_ENV` | Deployment environment; selects `deploy/values/<component>/<DPL_ENV>.yaml.gotmpl` | `local` |
| `DPL_DOMAIN` | Base DNS zone; ingress hosts are `<component>.${DPL_DOMAIN}` | **none** — rendered with `requiredEnv`, so helmfile fails if unset |
| `DPL_DEBEZIUM_VERSION` | Debezium helm chart version | from `versions.env` |
| `DPL_NAMESPACE` | Debezium namespace | `dmp` |
| `DPL_CLUSTER_TYPE` | Cluster provider (`kind` or `k3s`) | `kind` |
| `DPL_DMP_RESOURCE_PREFIX` / `DPL_DMP_ENVIRONMENT` | Resource naming prefix | used in JSON payloads |
| `DPL_DMP_KAFKA_BOOTSTRAP_SERVERS` | Kafka bootstrap servers | used in JSON payloads |

Known `DPL_ENV` values: `local`, `homelab`, `aws`, `hetzner`.

### JSON payload environment expansion

DMP payload files use `${ENV_VAR}` syntax, expanded from the environment when a payload is loaded.
A missing variable expands to an empty string rather than raising an error, so an unset value
produces a malformed payload rather than a failure — this applies to `data-pipelines/config/common/`
and `data-pipelines/config/<name>/payloads/`.

### Resource naming convention

DMP resources are named deterministically from the payload's `name` field after expansion, typically
`${DPL_DMP_RESOURCE_PREFIX}-${DPL_DMP_ENVIRONMENT}-<resource-type>`. The convention exists so a resource can
be found by name and reused rather than duplicated. Every payload MUST have a `"name"` key.

## Helm Deployments

Controlled by `deploy/helmfile.yaml.gotmpl`. Releases are selectable by label — `infra=true` for
operators, databases and ingress; `app=<release-name>` for an individual release:

```bash
helmfile --file deploy/helmfile.yaml.gotmpl --selector app=strimzi-cluster-operator apply
helmfile --file deploy/helmfile.yaml.gotmpl --selector infra=true apply
```

Helm has no `needs:`, so operator-then-custom-resource ordering is expressed by applying separate
selectors in sequence (see Build & Run above), not by a single `apply`.

Releases carrying a cert-manager `Certificate` must not roll back on failure: `helm --wait` blocks
on issuance, and if DNS-01 lags past the timeout the default rollback would uninstall an otherwise
healthy release. The Keycloak release documents this in place.

## Conventions

- Values files are per-component and per-environment: `deploy/values/<component>/<DPL_ENV>.yaml.gotmpl`
- Prefer adding a values layer over forking a chart
- Version pins go in `versions.env`, secrets in `.env`, never the reverse
- Run `scripts/validate-helm.sh` before proposing chart changes

## Gotchas

1. **Exported shell variables beat both env files.** Loading is non-overriding, so a stale exported `DPL_*` in your shell silently wins over `.env` and `versions.env`. Variables were renamed from `DBZ_*`/`KIND_*`/`LAB_*`/`DMP_*` to a single `DPL_` prefix; `scripts/lib/env.sh` warns if a legacy name is still exported.
2. **`DPL_DOMAIN` has no default at render time** despite what older docs claimed — `deploy/values/dmp/local.yaml.gotmpl` uses `requiredEnv`. The convention is sslip.io in dash notation: `.env.example` ships `127-0-0-1.sslip.io` for Kind (which publishes 80/443 on `127.0.0.1`) and `scripts/lab/hcloud-up.sh` prints `<ip-with-dashes>.sslip.io` for the Hetzner lab, so neither needs `/etc/hosts`. Note `scripts/validate-helm.sh` still falls back to `platform.debezium.local` — that value only has to render, never resolve.
3. **Commented-out releases**: `helmfile.yaml.gotmpl` contains disabled blocks for optional or retired components (Apicurio, CDC dashboard, Kafka Connect). Don't enable without understanding the dependency chain.
4. **Seeding and scenarios have no runner.** `resources/data/` and `data-pipelines/config/` are intact data with nothing in the repository to execute them, pending the `just` + JBang rebuild.
5. **`certs/` is applied by hand** (`kubectl apply -f certs/`) and is the only Kubernetes resource set outside both Helm and helmfile. It is homelab-only and slated to be folded into a gated release.
6. **`.env` and `.env.example` may be unreadable to tooling** that denies dotenv paths; read the committed version with `git show HEAD:.env.example` instead.
