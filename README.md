# debezium-platform-lab

Reproducible, environment-agnostic automation for running the [Debezium Platform](https://debezium.io/documentation/reference/stable/operations/debezium-platform.html) on Kubernetes — the same [**helm**](https://helm.sh/) charts and [**helmfile**](https://helmfile.readthedocs.io/) releases take you from a local [Kind](https://kind.sigs.k8s.io/) cluster to AWS or a self-hosted k3s box.

[![Helm validation](https://github.com/pxcamus/debezium-platform-lab/actions/workflows/helm-validation.yaml/badge.svg)](https://github.com/pxcamus/debezium-platform-lab/actions/workflows/helm-validation.yaml)

> A community project. Not affiliated with, or endorsed by, the Debezium project or Red Hat.

It provisions a change-data-capture (CDC) stack — Kafka (Strimzi), PostgreSQL (CloudNativePG), MongoDB, and more to come, the Debezium Operator and the Debezium Platform — and ships the JSON payloads that define connections, sources, destinations and pipelines for the Debezium Platform API.

Defaults target a local Kind cluster. Passwords in the Helm charts and `.env.example` are non-secret demo values.

**📖 Documentation: [lab.1int.io](https://lab.1int.io/)** — start with [Get running](https://lab.1int.io/getting-started/) for the guided walkthrough, or [Troubleshooting](https://lab.1int.io/troubleshooting/) when something fails.

> **Rebuild in progress.** The Go task runner (`mage`) and the `dmp-lab` CLI were removed; the
> toolchain is now [`just`](https://just.systems/) over `helmfile`, and deployment is unaffected —
> it was always `helmfile` underneath.

---

## Quick start (local Kind)

```bash
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

