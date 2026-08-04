# Debezium Platform Lab

Build a data pipeline that carries every change in your database — every insert, update and
delete — to wherever you need it, configured in a web UI instead of hand-written connector
plumbing.

This lab stands up the [Debezium Platform](https://debezium.io/documentation/reference/stable/operations/debezium-platform.html)
with a seeded demo database and a working pipeline, so you can see that happen in about ten
minutes and then point it at your own data.

!!! warning "This site is a skeleton"

    The structure is in place; the pages are being written. Until then the
    [repository README](https://github.com/pxcamus/debezium-platform-lab) is the
    authoritative instruction set.

## What you actually get

A running platform where you define pipelines by filling in forms:

- **Sources** — a database you want to capture from. PostgreSQL, MongoDB and SQL Server
  today; the platform's catalogue keeps growing.
- **Destinations** — where change events are delivered. An HTTP endpoint, Kafka, and other
  sinks the platform supports.
- **Transforms** — optional reshaping between the two.
- **Pipelines** — a source bound to a destination, running and observable.

Everything the lab creates through the UI is also expressible as JSON, so once a pipeline
works you can commit it and recreate it anywhere. Nothing you build here is trapped in a
click-path.

The first run delivers change events to a plain HTTP endpoint, because that makes them
immediately visible: change a row, watch the JSON arrive. Kafka is one configuration change
away when you want it — swapping the destination is the point of the platform, not a
migration.

## Do I need Kubernetes?

For now, yes — the lab deploys onto Kubernetes, locally with [Kind](https://kind.sigs.k8s.io/)
or onto a cluster you already run.

**Planned:** a host-based path that runs the same platform directly on a machine with a
container runtime, no Kubernetes involved. The pipelines, the UI and the resource
definitions are identical either way — only the thing underneath them changes — so a
pipeline built on one runs unmodified on the other. This is not available yet; the
Kubernetes path is the only one that works today.

## Where to go

| If you want to | Go to 
|---|---|
| Get it running now | [Get running](getting-started/index.md) |
| Understand what it deploys before installing | [Concepts](concepts/index.md) |
| Fix something that failed | [Troubleshooting](troubleshooting/index.md) |
| Deploy somewhere other than Kind | [Guides](guides/index.md) |
| Look up a target, variable or value | [Reference](reference/index.md) |

## Before you start

The [Get running](getting-started/index.md) path expects a machine with roughly 4 CPUs,
8 GB of RAM and 30 GB of free disk, and pulls around 20 container images the first time.
 
<!-- TODO: add the architecture diagram and a terminal recording of a full deploy. -->
<!-- TODO: the resource figures above cover today's all-in deployment. Revisit once the
     lab profile drops Kafka from the default path. -->
