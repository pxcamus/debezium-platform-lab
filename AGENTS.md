# AGENTS.md

## Project Overview

Helm charts and helmfile releases for deploying and managing the Debezium Platform on Kubernetes,
plus the JSON payloads that describe Debezium Platform pipelines.

**There is no application code in this repository.** The Go library, the `mage` task runner and the
`dmp-lab` CLI were removed; the toolchain is now [`just`](https://just.systems/) over `helmfile`, and
everything here is YAML, JSON, shell and documentation. Do not reintroduce Go without being asked.

## Build & Run

Every command goes through `just`, which sources `scripts/lib/env.sh` so `.env` and `versions.env`
are resolved identically everywhere. `just --list` shows the full set.

```bash
just kind-recreate    # delete + recreate the local Kind cluster, point kubectl at it
just releases         # what the helmfile would install — declarative, no cluster needed
just apply            # install/upgrade every enabled release
just hf <args>        # raw helmfile passthrough: template, diff, destroy, --selector ...

scripts/validate-helm.sh   # lint, gating assertions, render every environment (no cluster)
scripts/validate-env.sh    # env-var consistency between .env.example and the payloads
```

`just apply` always passes `--skip-diff-on-install`. This is deliberate and should not be made
conditional: helm-diff renders a release against the live cluster, so a chart carrying custom
resources whose CRDs arrive in the same run (`OpenTelemetryCollector`, `ServiceMonitor`) can never
diff on a fresh cluster. The flag only skips the diff for releases being installed for the first
time; existing releases still diff normally.

`scripts/with-env.sh` fronts a single command with the same environment, for use outside `just`.

CI is three GitHub Actions workflows:

| Workflow | Runs |
|---|---|
| `toolchain.yaml` | shellcheck, `bash -n`, `just --fmt --check`, `just --list`, `validate-env.sh` — always |
| `helm-validation.yaml` | `scripts/validate-helm.sh` — on changes under `deploy/` |
| `docs.yaml` | MkDocs build and deploy to lab.1int.io |

## Architecture

```
justfile                 # every task; recipes source scripts/lib/env.sh
deploy/
├── helmfile.yaml.gotmpl # All releases, gated by two axes (see below)
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
├── lib/env.sh           # THE env loader — sourceable, sets variables, runs nothing
├── with-env.sh          # Fronts one command with lib/env.sh resolved
├── validate-helm.sh     # Offline chart validation + gating assertions
├── validate-env.sh      # Env-var consistency checks
├── vendor-keycloak-operator.sh
└── lab/                 # Lab VM provisioning (Hetzner)
```

`data-pipelines/config/` was `ko/scenarios/` while the Go library lived in `ko/`; older commits,
issues and the published tags still use the old path.

## What installs, and why — the two axes

`installed:` **defaults to true in helmfile.** Every release therefore needs an explicit gate, and a
release added without one installs everywhere. This was false for months and produced 14 releases
where 4 were wanted.

Two orthogonal axes decide, declared at the top of `deploy/helmfile.yaml.gotmpl`:

- **`DPL_ENV` — HOW to configure.** Domains, TLS, sizing, which ingress controller. Gates releases
  that only make sense in one environment: `ingress-nginx` (local), `cert-manager`,
  `cert-manager-webhook-gandi`, `zot`, the Keycloak/SSO chain (homelab).
- **`.Values.components` — WHAT to install.** Optional component groups, supplied by
  `--state-values-file` or `--state-values-set`. **Default false.** Components:
  `postgres`, `mongodb`, `sqlserver`, `kafka`, `kafka-ui`, `kafka-connect`, `http-server`,
  `cdc-dashboard`, `apicurio`.

```bash
just apply --state-values-set components.sqlserver=true
just hf --state-values-file <file> list      # a components.yaml; no such files committed yet
```

The deciding question when adding a release: *could two setups in the same environment reasonably
differ on this?* Yes → component. No → `DPL_ENV`.

`--file` is **not** repeatable in helmfile — it is a plain string flag and the last one silently
wins, unlike `--values`, `--selector` and `--state-values-file` which are `stringArray`. Component
selection rides on state values for exactly this reason. If an example ever needs releases the base
does not define, that is `helmfiles:` sub-helmfile composition, not a second `--file`.

The base install is **four releases**: `debezium-platform`, `ingress-nginx`,
`kube-prometheus-stack`, `opentelemetry-operator`. The Debezium Operator is not a fifth — by default
it ships inside `debezium-platform` as a subchart. `scripts/validate-helm.sh` asserts this exact set,
so adding a base release is a deliberate two-file change.

### Operator packaging

The Debezium Operator ships one of two ways, never both, never neither:

| `DPL_OPERATOR_MODE` | Result |
|---|---|
| `bundled` (default) | A `debezium-platform` subchart; the standalone release is off |
| `standalone` | Its own release with an independently pinned image tag; the subchart is off |

Both switches derive from a single `$bundledOperator` in the helmfile — the release's `installed:`
and the chart's `debezium-operator.enabled` via `set:`. **Do not move that value into
`values/dmp/*.yaml.gotmpl`**: it used to live there and drifted, so aws installed two operators into
the same namespace. `validate-helm.sh` counts the rendered operator Deployments to catch a recurrence.

Note the two modes pin different versions: bundled follows the platform chart's dependency,
standalone follows `DPL_DEBEZIUM_OPERATOR_TAG`.

## Environment & Configuration

**All configuration is environment variables**, loaded by `scripts/lib/env.sh` in this order:
`deploy/environment/versions.env` first (shared, non-sensitive version pins), then `.env`
(git-ignored — secrets and host-specific overrides). Both are sourced with `set -a`, so this is
**last-wins**: `.env` beats `versions.env`, and both beat what is already exported in your shell.
Both files are optional; a checkout with only `versions.env` works, which is how CI runs.

**Version pins live in `versions.env` only** — never duplicate them into `.env` or `.env.example`.

`.env.example` is deliberately minimal: only variables with no default anywhere, or whose default is
wrong when run from a workstation rather than in-cluster. The exhaustive list is in
`docs/getting-started/deploy.md`.

### Critical env vars

| Variable | Purpose | Default |
|---|---|---|
| `DPL_ENV` | Deployment environment; selects `deploy/values/<component>/<DPL_ENV>.yaml.gotmpl` | `local` |
| `DPL_DOMAIN` | Base DNS zone; ingress hosts are `<component>.${DPL_DOMAIN}` | **none** — rendered with `requiredEnv`, so helmfile fails if unset |
| `DPL_DEBEZIUM_VERSION` | Debezium helm chart version | from `versions.env` |
| `DPL_NAMESPACE` | Debezium namespace | `dmp` |
| `DPL_OPERATOR_MODE` | `bundled` or `standalone` | `bundled` |
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

## Helm deployments

Controlled by `deploy/helmfile.yaml.gotmpl`. Releases are selectable by label — `infra=true` for the
ingress controller and monitoring; `app=<release-name>` for an individual release:

```bash
just apply --selector infra=true
just hf --selector app=debezium-platform template
```

A selector **cannot** switch on a release the helmfile has gated off — that needs state values (see
the two axes above).

Ordering is expressed with helmfile `needs:` in `<namespace>/<release>` form, not by applying
selectors in sequence. Existing edges: the Kafka and database operator chains, the identity chain,
`debezium-platform` → both monitoring releases, and `kube-prometheus-stack` → `ingress-nginx`. That
last one is conditional on `DPL_ENV=local` because ingress-nginx does not install elsewhere, and an
edge pointing at a disabled release is not reliably a no-op.

`helmDefaults` sets `wait: true`, `atomic: true`, `timeout: 120`. Two consequences:

- Releases carrying a cert-manager `Certificate` must not roll back on failure — `helm --wait` blocks
  on issuance, and if DNS-01 lags past the timeout the default rollback uninstalls an otherwise
  healthy release. The Keycloak release documents this in place.
- A chart that never becomes Ready is rolled back and disappears, so the visible symptom is often
  "nothing installed" rather than an error naming the cause. The OpenTelemetry Operator's webhook
  certificate is the canonical example — see `values/opentelemetry-operator/self-signed.yaml`.

## Conventions

- Values files are per-component and per-environment: `deploy/values/<component>/<DPL_ENV>.yaml.gotmpl`
- A values file that answers a yes/no question about the cluster, not a per-environment question, is
  named for the answer instead — `opentelemetry-operator/self-signed.yaml`, gated on `ne homelab`
- Prefer adding a values layer over forking a chart
- Version pins go in `versions.env`, secrets in `.env`, never the reverse
- Run `scripts/validate-helm.sh` before proposing chart changes; it needs no cluster
- `just --fmt --check --unstable` gates CI, so run `just --fmt --unstable` after editing the justfile
- A `just` recipe's doc string is the **last** comment line above it; multi-line comments render
  their final line in `just --list`

## Gotchas

1. **`.env` beats your shell, not the other way round.** `scripts/lib/env.sh` sources with `set -a`,
   so `DPL_ENV=homelab just hf template` silently renders `local` when `.env` says `local`. To test
   another environment, edit `.env` or bypass the loader. Variables were renamed from
   `DBZ_*`/`KIND_*`/`LAB_*`/`DMP_*` to a single `DPL_` prefix; `env.sh` warns if a legacy name is
   still exported.
2. **`DPL_DOMAIN` has no default at render time** — `deploy/values/dmp/local.yaml.gotmpl` uses
   `requiredEnv`. The convention is sslip.io in dash notation: `.env.example` ships
   `127-0-0-1.sslip.io` for Kind (which publishes 80/443 on `127.0.0.1`) and
   `scripts/lab/hcloud-up.sh` prints `<ip-with-dashes>.sslip.io` for the Hetzner lab, so neither
   needs `/etc/hosts`. `scripts/validate-helm.sh` falls back to `platform.debezium.local` — that
   value only has to render, never resolve.
3. **Optional components are gated, not commented out.** Apicurio, the CDC dashboard, Kafka Connect
   and the databases are live releases with `installed: {{ $component }}`. Enable them with state
   values; do not go looking for comment blocks, and do not delete the gate to "turn one on".
4. **Seeding and scenarios have no runner.** `resources/data/` and `data-pipelines/config/` are
   intact data with nothing in the repository to execute them, pending the `just` + JBang rebuild.
5. **`certs/` is applied by hand** (`kubectl apply -f certs/`) and is the only Kubernetes resource
   set outside both Helm and helmfile. It is homelab-only and slated to be folded into a gated release.
6. **`.env` and `.env.example` may be unreadable to tooling** that denies dotenv paths; read the
   committed version with `git show HEAD:.env.example` instead.
7. **`values/kube-prometheus-stack/` has no `hetzner.yaml.gotmpl`.** The release sets
   `missingFileHandler: Warn`, so hetzner installs with chart defaults rather than failing. Renders
   fine, but nobody chose that configuration.
