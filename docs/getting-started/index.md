# Get running

One path, no choices: a local Kind cluster running the platform with a seeded PostgreSQL
source, ending in a pipeline you can watch move data. Deploying elsewhere is a
[guide](../guides/index.md), not a fork in this road.

!!! warning "Placeholder"

    This page is a skeleton. Follow the
    [README quick start](https://github.com/pxcamus/debezium-platform-lab#quick-start-local-kind)
    until it is written.

## The three steps

1. **[Check your machine](doctor.md)** — one command tells you what is missing and how to
   install it. Nothing else on this path asks you to install anything.
2. **[Deploy the stack](deploy.md)** — create the cluster and apply every release in
   dependency order.
3. **[Run your first pipeline](first-pipeline.md)** — seed the demo database, create the
   platform resources, and watch change events arrive.

## What this costs you

| | |
|---|---|
| Time | ~10 minutes warm, ~25 minutes on a cold image cache |
| Memory | 8 GB RAM free (the stack does not fit comfortably in less) |
| Disk | ~30 GB, mostly container images |
| Network | Pulls from `quay.io`, `docker.io` and `ghcr.io` |

Everything runs locally and is deleted with a single teardown command. No cloud account is
involved unless you follow one of the cloud guides.

<!-- TODO: add expected output for each step; a reader cannot tell success from a hung pod. -->
