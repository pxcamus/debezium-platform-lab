# debezium-platform-lab

Reproducible, environment-agnostic automation for running the [Debezium Platform](https://debezium.io/documentation/reference/stable/operations/debezium-platform.html) on Kubernetes — the same [**helm**](https://helm.sh/) charts and [**helmfile**](https://helmfile.readthedocs.io/) releases take you from a local [Kind](https://kind.sigs.k8s.io/) cluster to AWS or a self-hosted k3s box.

[![Helm validation](https://github.com/pxcamus/debezium-platform-lab/actions/workflows/helm-validation.yaml/badge.svg)](https://github.com/pxcamus/debezium-platform-lab/actions/workflows/helm-validation.yaml)

> A community project. Not affiliated with, or endorsed by, the Debezium project or Red Hat.

It provisions a change-data-capture (CDC) stack — web UI to configure Kafka (Strimzi), PostgreSQL (CloudNativePG), MongoDB, and more to come, the Debezium Operator and the Debezium Platform with monitoring.

Defaults target a local Kind cluster. Passwords in the Helm charts and `.env.example` are non-secret demo values.

**📖 Documentation: [lab.1int.io](https://lab.1int.io/)** — start with [Get running](https://lab.1int.io/getting-started/) for the guided walkthrough, or [Troubleshooting](https://lab.1int.io/troubleshooting/) when something fails.

> **Rebuild in progress.** The Go task runner (`mage`) and the `dmp-lab` CLI were removed; the
> toolchain is now [`just`](https://just.systems/) over `helmfile`, and deployment is unaffected —
> it was always `helmfile` underneath.

---

## Quick start (local Kind)

```bash
git clone https://github.com/pxcamus/debezium-platform-lab.git
cd debezium-platform-lab

cp .env.example .env
just kind-recreate
just apply
```

That gives you the platform, an ingress controller and monitoring — nothing else until you ask
for it. `just releases` lists what will be installed without touching a cluster, and `just --list`
shows every recipe.

Full walkthrough — tool installation, what each command does, what healthy output looks like,
and what to do when it doesn't:

- **[Prerequisites](https://lab.1int.io/getting-started/prerequisites/)** — the five tools you need
- **[Deploy the stack](https://lab.1int.io/getting-started/deploy/)** — configure, create, apply, verify
- **[Troubleshooting](https://lab.1int.io/troubleshooting/)** — when it doesn't

---

## Blog series

A series on [debezium.io](https://debezium.io/blog/) walks through this repository. Each post is
written against a tag — check it out before following along, because the interface moves.

| Post | Published | Written against |
|---|---|---|
| [Running the Debezium Platform on AWS: PostgreSQL to Amazon Kinesis](https://debezium.io/blog/2026/07/24/debezium-platform-on-aws-postgres-to-kinesis/) | 2026-07-24 | `v0.1.0` |
| Single Sign-On for the Debezium Platform, part 1 — forthcoming | — | `v0.3.0` |

> The AWS post was written against `v0.1.0`, when the task runner was `mage` — its
> `mage cluster:recreate` and `mage helm:platform` have no equivalent on `main`, which uses `just`.
> Run `git checkout v0.1.0` to follow it.

Parts 2 (Entra ID brokering) and 3 (RBAC groundwork) are forthcoming.

