# debezium-platform-lab

Reproducible, environment-agnostic automation for running the [Debezium Platform](https://debezium.io/documentation/reference/stable/operations/debezium-platform.html) on Kubernetes — the same [**helm**](https://helm.sh/) charts and [**helmfile**](https://helmfile.readthedocs.io/) releases take you from a local [Kind](https://kind.sigs.k8s.io/) cluster to AWS or a self-hosted k3s box.

[![Helm validation](https://github.com/pxcamus/debezium-platform-lab/actions/workflows/helm-validation.yaml/badge.svg)](https://github.com/pxcamus/debezium-platform-lab/actions/workflows/helm-validation.yaml)

> A community project. Not affiliated with, or endorsed by, the Debezium project or Red Hat.

It provisions a change-data-capture (CDC) stack — Kafka (Strimzi), PostgreSQL (CloudNativePG), MongoDB, optionally SQL Server, the Debezium Operator and the Debezium Platform — and ships the JSON payloads that define connections, sources, destinations and pipelines for the Debezium Platform API.

Defaults target a local Kind cluster. Passwords in the Helm charts and `.env.example` are non-secret demo values.

**📖 Documentation: [lab.1int.io](https://lab.1int.io/)** — start with [Get running](https://lab.1int.io/getting-started/) for the guided walkthrough, or [Troubleshooting](https://lab.1int.io/troubleshooting/) when something fails. The quick start below is the condensed version of the same path.

> **Rebuild in progress.** The Go task runner (`mage`) and the `dmp-lab` CLI were removed while the
> toolchain moves to [`just`](https://just.systems/). Deployment is unaffected — it was always
> `helmfile` underneath, and the commands below are what the mage targets ran. Database seeding and
> scenario application had no other implementation and are temporarily unavailable; the SQL and
> JavaScript seed scripts (`resources/data/`) and the DMP payloads (`data-pipelines/config/`) are untouched
> and still describe what those steps did.

---

## Prerequisites

| Tool | Purpose |
|---|---|
| [helmfile](https://helmfile.readthedocs.io/en/latest/#installation) + [helm](https://helm.sh/) | chart orchestration (helm-diff plugin recommended) |
| [kind](https://kind.sigs.k8s.io/) | local Kubernetes cluster |
| [kubectl](https://kubernetes.io/docs/tasks/tools/) | cluster access |
| [Docker](https://www.docker.com/) | container runtime for Kind |

Optional: [kubeconform](https://github.com/yannh/kubeconform) (used by CI/helm validation).

---

## Quick start (local Kind)

For the step-by-step version — what each command does, what healthy output looks like, and what to do when it doesn't — see [Get running](https://lab.1int.io/getting-started/).

```bash
# 1. Configure environment
cp .env.example .env
#    Ensure DPL_ENV=local and DPL_CLUSTER_TYPE=kind in .env

# 2. Create the local cluster
kind create cluster --name dmp --config deploy/clusters/kind/kind-ingress.yaml

# 3. Deploy the stack, ordered by dependency. Each operator is applied and
#    established before the resources that depend on its CRDs.
export HELMFILE="helmfile --file deploy/helmfile.yaml.gotmpl"
$HELMFILE --selector app=strimzi-cluster-operator    apply
$HELMFILE --selector app=cnpg-operator               apply
$HELMFILE --selector app=mongodb-community-operator  apply
$HELMFILE --selector infra=true                      apply --skip-diff-on-install
$HELMFILE --selector app=debezium-operator           apply
$HELMFILE --selector app=debezium-platform           apply
```

Environment variables are read from `.env`, so run these from the repository root with the file in
place — `scripts/with-env.sh` wraps a command with the same resolution if you prefer.

Seeding the demo databases and creating the platform's pipelines were `mage data:*` and
`mage scenario:*`; both are pending the toolchain rebuild.

---

## Known gaps

- **Seeding and scenarios have no runner right now.** `resources/data/` (PostgreSQL SQL, MongoDB JS) and `data-pipelines/config/` (per-scenario `scenario.yaml` + JSON payloads) are intact, but nothing in the repository executes them since the Go code was removed.
- **Only the MongoDB replica-set scenario was ever wired up.** The `postgres-basic` and `sqlserver-basic` directories under `data-pipelines/config/` contain payloads that were never exposed as targets.
- **SQL Server requires amd64.** Microsoft ships no arm64 SQL Server image (and Azure SQL Edge, the historical arm64 stand-in, was retired 2025-09-30). The `mssql` release is not part of the sequence above — apply it explicitly (`helmfile --file deploy/helmfile.yaml.gotmpl --selector app=mssql apply`) — so this only affects SQL Server work. Use an amd64 cluster (`DPL_CLUSTER_TYPE=k3s` on a cloud box) for it.
- **Debezium Platform release images are amd64-only** (`platform-conductor` / `platform-stage` version tags, checked 2026-07); only the `nightly` tag is multi-arch. `deploy/environment/versions.env` pins `nightly` for this reason. Everything else on the default path — Strimzi operator and Kafka, MongoDB operator/server, CloudNativePG and PostgreSQL, ingress-nginx, the Debezium Operator — publishes amd64+arm64.

---

## Configuration

**All configuration is via environment variables**, loaded from two layered files: `.env` (git-ignored — secrets and host-specific overrides) takes precedence, and [`deploy/environment/versions.env`](deploy/environment/versions.env) (shared, non-sensitive version pins) is loaded as a fallback. `scripts/with-env.sh` and `scripts/validate-helm.sh` both resolve this way, so **version pins live in `versions.env` only** — `.env` doesn't duplicate them and a CI checkout can run from `versions.env` alone.

[`.env.example`](.env.example) is the minimum needed for the local Kind demo: variables that have no default anywhere, or whose default is wrong when you run from your own machine. The full set is documented in [`docs/reference/environment.md`](docs/reference/environment.md).

| Variable | Purpose | Default |
|---|---|---|
| `DPL_DEBEZIUM_VERSION` | Debezium Helm chart version | **required** (from `versions.env`) |
| `DPL_ENV` | Deployment environment; selects `deploy/values/<component>/<DPL_ENV>.yaml.gotmpl` | `local` |
| `DPL_DOMAIN` | Base DNS zone; every ingress host is `<component>.${DPL_DOMAIN}` (e.g. `dmp.`, `apicurio.`, `kafbat.`, `registry.`) | **required** — rendered with `requiredEnv` |
| `DPL_NAMESPACE` | Debezium Platform namespace | `dmp` |
| `DPL_CLUSTER_TYPE` | Cluster provider: `kind` or `k3s` | `kind` |
| `DPL_DMP_RESOURCE_PREFIX` / `DPL_DMP_ENVIRONMENT` | Prefix for deterministic DMP resource names | — |
| `DPL_DMP_KAFKA_BOOTSTRAP_SERVERS` | Kafka bootstrap for DMP payloads | — |

Known `DPL_ENV` values in this repo: `local`, `homelab` (self-hosted k3s + public TLS), `aws`, `hetzner`. Each has a matching values file under `deploy/values/<component>/`.

DMP JSON payloads use `${ENV_VAR}` syntax, expanded from the environment when the payload is loaded.

---

## Common tasks

All releases are selectable by label:

```bash
helmfile --file deploy/helmfile.yaml.gotmpl --selector infra=true apply
helmfile --file deploy/helmfile.yaml.gotmpl --selector app=debezium-platform apply
helmfile --file deploy/helmfile.yaml.gotmpl --selector app=debezium-operator destroy
helmfile --file deploy/helmfile.yaml.gotmpl diff              # preview pending changes
```

Labels in use include `infra=true` (operators, databases, ingress) and `app=<release-name>` for every
individual release — see [`deploy/helmfile.yaml.gotmpl`](deploy/helmfile.yaml.gotmpl).

---

## Validate Helm charts (no cluster required)

```bash
scripts/validate-helm.sh
```

Runs `helm lint` / `helm template` per chart and `helmfile lint` / `template` piped through `kubeconform`. This is also enforced in CI ([`.github/workflows/helm-validation.yaml`](.github/workflows/helm-validation.yaml)) on changes under `deploy/`.

---

## Repository layout

```
deploy/
├── helmfile.yaml.gotmpl # All Helm releases, ordered by dependency
├── charts/              # Custom charts (kafka-cluster, postgresql-cluster, mssql, ...)
├── values/              # Per-component, per-environment values
├── clusters/            # Kind cluster configs
└── environment/         # versions.env — shared version pins

data-pipelines/config/   # DMP scenario manifests + JSON payloads
├── common/              # Shared connections, destinations, transforms
└── <scenario>/          # Per-scenario scenario.yaml + payloads/

resources/               # SQL / MongoDB seed scripts, standalone DMP payloads
certs/                   # Optional TLS (see below)
docs/                    # MkDocs sources for lab.1int.io
scripts/
├── validate-helm.sh     # Offline chart validation
├── with-env.sh          # Runs a command with .env + versions.env resolved
└── lab/                 # Lab VM provisioning
```

---

## Optional: HTTPS via cert-manager + Gandi (homelab only)

The files under [`certs/`](certs/) and the `cert-manager-webhook-gandi` release in the helmfile expose platform services over HTTPS using Let's Encrypt with a DNS-01 challenge solved through the [Gandi](https://www.gandi.net/) DNS API. **This is entirely optional and specific to a self-hosted homelab (`DPL_ENV=homelab`, wildcard domain `*.example.com` — replace with your own); a local Kind demo does not need it.**

The webhook release is gated on `DPL_ENV=homelab`, so it is not installed for `local`. To use it in your own environment:

1. Set your ACME registration email in `certs/letsencrypt-*-clusterissuer.yaml` (currently a placeholder).
2. Provide your Gandi Personal Access Token. It is **not committed** — create the Secret out of band:
   ```bash
   kubectl -n cert-manager create secret generic gandi-credentials \
     --from-literal=pat="$GANDI_PAT"
   ```
   (`deploy/values/cert-manager-webhook-gandi/homelab.yaml` also carries a `gandiPat` placeholder for the webhook chart itself.)
3. Adjust the domains in `certs/*.yaml` to your own, then apply the ClusterIssuers and Certificate:
   ```bash
   kubectl apply -f certs/
   ```

---

## Notes & gotchas

- **`DPL_DOMAIN` has no default at render time.** The values templates use `requiredEnv "DPL_DOMAIN"`, so helmfile fails outright if it is unset. `.env.example` defaults it to `127-0-0-1.sslip.io`, which resolves to `127.0.0.1` where the Kind node publishes ports 80 and 443 — no `/etc/hosts` entries needed. If your resolver drops public answers that point at loopback (DNS rebinding protection) or you are offline, `dig +short dmp.127-0-0-1.sslip.io` returns nothing useful; set a private zone and map the hosts by hand instead. `scripts/lab/hcloud-up.sh` prints the equivalent `<ip>.sslip.io` value for the Hetzner lab.
- **Exported shell variables win over `.env`.** Both files are loaded without overriding what is already in the environment, so a stale exported `DPL_*` silently beats the file. Every variable is prefixed `DPL_` (`env | grep DPL_` shows the lot); the old `DBZ_*`, `KIND_*`, `LAB_*` and `DMP_*` names are gone, and `scripts/lib/env.sh` warns if one is still exported in your shell.
- **Idempotent DMP resources:** the payloads are named deterministically (`${DPL_DMP_RESOURCE_PREFIX}-${DPL_DMP_ENVIRONMENT}-<type>`) so a resource can be found by name and reused rather than duplicated.
- **Commented-out releases:** `helmfile.yaml.gotmpl` contains disabled blocks for optional/retired components (Apicurio, CDC dashboard, Kafka Connect). Don't enable without checking the dependency chain.
