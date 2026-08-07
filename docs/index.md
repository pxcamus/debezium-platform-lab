# Debezium Platform Lab

Build a data pipeline that carries every change in your database — every insert, update and
delete — to wherever you need it, configured in a web UI instead of hand-written connector
plumbing.

This lab stands up the [Debezium Platform](https://debezium.io/documentation/reference/stable/operations/debezium-platform.html), 
either empty, ready for you to explore and configure, or with a seeded demo database and a working pipeline, all with a couple simple commands.

!!! info "Community project"

    Not affiliated with, or endorsed by, the Debezium project or Red Hat.

## What you actually get

A running platform where you define pipelines by filling in forms:

- **Sources** — a database you want to capture from. PostgreSQL, MongoDB, Oracle, SQL Server, and more;
  the platform's catalogue keeps growing.
- **Destinations** — where change events are delivered. An HTTP endpoint, Kafka, and other
  sinks the platform supports.
- **Transforms** — optional reshaping between the two.
- **Pipelines** — a source bound to a destination, running and observable.

Everything the lab creates through the UI is also expressible as JSON, so once a pipeline
 works, you can commit it and recreate it anywhere. Nothing you build here is trapped in a
click-path.

The default is the platform plus monitoring — Prometheus, Grafana, the otel operator, ingress-nginx, 
ready for you to explore and configure. You can also start with a seeded demo database and a working pipeline delivering change events to a plain HTTP endpoint, 
because that makes them immediately visible: change a row, watch the JSON arrive. Kafka is one configuration change
away when you want it — swapping the destination is the point of the platform, not a
migration.

## Do I need Kubernetes?

For now, yes — the lab deploys onto Kubernetes, locally with [Kind](https://kind.sigs.k8s.io/)
or onto a cluster you already run. [just](https://just.systems/man/en) recipes controlled by environment variables
abstract you from raw helm/kubernetes commands:

```shell
just kind-recreate && just apply
```

and your platform is up-and-running with monitoring and observability built in with sensible defaults.

**Planned:** a host-based path — the same pipelines on a machine with a container runtime,
no Kubernetes involved.

The work that makes this possible is the Debezium team's: Debezium
3.7.0.Alpha1 added a [native distribution module for Debezium
Server](https://debezium.io/blog/2026/07/30/debezium-3-7-alpha1-released/): Debezium Server
can now be built as a native executable, with a faster startup and a smaller memory footprint
than the JVM distribution.

To stay informed, subscribe to the [Debezium blog](https://debezium.io/blog/).

## Where to go

| If you want to             | Go to                                              |
|----------------------------|----------------------------------------------------|
| Get it running now         | [Get running](getting-started/index.md)            |
| Fix something that failed  | [Troubleshooting](troubleshooting/index.md)        |
| Deploy a more complex flow | [Guides](guides/index.md)                          |

## Before you start { #resources-required }

How much resources you need depends on where the platform runs.

| Deployment | Runs on | Needs                                                            |
|---|---|------------------------------------------------------------------|
| **Kind** | your machine | roughly 2 CPUs, 4 GB RAM, 10 GB free disk                        |
| **k3s** | a server you connect to | the same, on that server — your own machine only needs the tools |
| **Existing cluster** | wherever it already runs | capacity on the cluster; nothing in particular here              |

