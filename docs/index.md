# Debezium Platform Lab

Stand up a complete change-data-capture stack on Kubernetes — Kafka, a source database,
the Debezium Operator and the [Debezium Platform](https://debezium.io/documentation/reference/stable/operations/debezium-platform.html) —
and watch a pipeline move rows, on a laptop or on a cluster you own.

!!! warning "This site is a skeleton"

    The structure is in place; the pages are being written. Until then the
    [repository README](https://github.com/pxcamus/debezium-platform-lab) is the
    authoritative instruction set.

## What you get

A single Kubernetes namespace running the platform end to end: Strimzi-managed Kafka,
PostgreSQL (CloudNativePG) or MongoDB seeded with demo data, the Debezium Operator, and
the Debezium Platform's conductor and stage components behind an ingress. From there you
create a connection, a source, a destination and a pipeline, and see change events flow.

The same automation targets a local [Kind](https://kind.sigs.k8s.io/) cluster, a
self-hosted k3s box, or AWS — the environment is a variable, not a fork.

## Where to go

| If you want to | Go to |
|---|---|
| Get it running now | [Get running](getting-started/index.md) |
| Understand what it deploys before installing | [Concepts](concepts/index.md) |
| Fix something that failed | [Troubleshooting](troubleshooting/index.md) |
| Deploy somewhere other than Kind | [Guides](guides/index.md) |
| Look up a target, variable or value | [Reference](reference/index.md) |

## Before you start

The [Get running](getting-started/index.md) path expects a machine with roughly 4 CPUs,
8 GB of RAM and 30 GB of free disk, and pulls around 20 container images the first time.
Budget ten minutes on a warm cache and rather more on a cold one.

<!-- TODO: add the architecture diagram and a terminal recording of a full deploy. -->
