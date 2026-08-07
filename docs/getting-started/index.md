# Get running

The default deployment gives you a local Kind cluster running the platform plus monitoring — Prometheus, Grafana, the otel operator, ingress-nginx. 
More comprehensive examples and deployments on Kubernetes infrastructure are documented in the [guide](../guides/index.md).

!!! warning "Partly written"

    Steps 1 and 2 are written. Step 3 — the first pipeline — is still a skeleton, because
    the seeding and scenario runners are being rebuilt.

## The three steps

1. **[Prerequisites](prerequisites.md)** — five tools and a container runtime. Nothing else
   on this path asks you to install anything.
2. **[Deploy the stack](deploy.md)** — two commands: create the cluster, apply the releases.
3. **[Run your first pipeline](first-pipeline.md)** — connect to your sources and destinations, create your first 
   data pipeline, and watch change events arrive.

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
