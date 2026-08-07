# Troubleshooting

Every deployment check has a stable identifier. When a check fails it prints the identifier
and a link straight to the section below that explains it. You should not have to search
this page.

!!! warning "Placeholder"

    The sections below are stubs and the identifiers are provisional. They are listed now
    so the checks and the documentation grow together.

## The contract

A heading here carries an explicit anchor equal to the check identifier — `inotify-limits`
becomes `#inotify-limits`. That is what lets the command link to it. Renaming a check means
renaming its anchor, and the build fails if a link no longer resolves, so the two cannot
drift apart silently.

<!-- TODO: add a CI check asserting every check ID in the registry has an anchor here. -->

---

## Container runtime not reachable { #container-runtime }

The runtime is not installed, not started, or your user cannot talk to its socket.

<!-- TODO -->

## Required tools missing or too old { #tooling-versions }

`kubectl`, `helm`, `helmfile` or `kind` is absent, or its version is below the minimum the
charts require.

<!-- TODO -->

## Not enough memory { #insufficient-memory }

The stack needs about 8 GB of free RAM. Below that, pods are evicted or the Kafka broker
is killed part-way through startup, which usually surfaces as an unrelated-looking timeout.

<!-- TODO -->

## Kernel inotify limits exhausted { #inotify-limits }

A Kubernetes node with many pods exhausts the default `fs.inotify` watch and instance
limits. Symptoms are misleading: containers fail to start with "too many open files", and
the control plane becomes intermittently unresponsive.

<!-- TODO -->

## Ports 80 or 443 already in use { #ingress-ports-busy }

The cluster maps the ingress to the host's HTTP ports. Another local service holding them
makes the platform unreachable even though every pod is healthy.

<!-- TODO -->

## Platform hostnames do not resolve { #dns-resolution }

Ingress hosts are derived from the configured DNS zone. If those names do not resolve to
the cluster, the deployment succeeds and nothing is reachable.

<!-- TODO -->

## Wrong cluster targeted { #cluster-mismatch }

The resolved kubeconfig and context do not match the environment you selected. This is the
check most worth reading before a deployment rather than after.

<!-- TODO -->

## Image pulls failing { #image-pull }

Registry unreachable, rate-limited, or the requested tag does not exist for your CPU
architecture.

The architecture case is the one that looks like something else. `deploy/environment/versions.env`
pins `DPL_IMAGE_TAG` and `DPL_IMAGE_TAG_CONDUCTOR` to `nightly` because the
`platform-conductor` and `platform-stage` *version* tags are published amd64-only — only
`nightly` is multi-arch. On Apple Silicon or any arm64 machine, changing those pins to a
release version gives you a pull failure or an `exec format error` that never mentions the
tag you changed. If you have overridden either variable in `.env`, put it back.

Microsoft publishes no arm64 SQL Server image at all, so the `sqlserver` component needs an
amd64 cluster regardless of tags.

<!-- TODO -->

## Releases applied out of order { #release-ordering }

Resources were created before the operator that owns their custom resource definitions was
established. The errors name missing kinds rather than the ordering problem that caused them.

<!-- TODO -->

---

## Still stuck

Open an issue at
[github.com/pxcamus/debezium-platform-lab/issues](https://github.com/pxcamus/debezium-platform-lab/issues)
with enough to reproduce:

```shell
just releases          # what the helmfile intends to install
just hf list           # the same, including the releases that are gated off
kubectl get pods -A
```

Plus the failing command's output, your `DPL_ENV`, and `env | grep DPL_`. Redact real
secrets.
