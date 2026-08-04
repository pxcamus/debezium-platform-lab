# Troubleshooting

Every preflight and deployment check has a stable identifier. When a check fails it prints
the identifier and a link straight to the section below that explains it. You should not
have to search this page.

!!! warning "Placeholder"

    The sections below are stubs, and the identifiers are provisional until the preflight
    command lands. They are listed now so the command and the documentation grow together.

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

## Exported shell variables override the environment file { #env-shadowing }

Variables already exported in your shell take precedence over the environment file, which
is loaded without overriding. A stale exported value silently changes which image tag is
deployed, or which cluster is targeted, with no warning anywhere.

<!-- TODO -->

## Wrong cluster targeted { #cluster-mismatch }

The resolved kubeconfig and context do not match the environment you selected. This is the
check most worth reading before a deployment rather than after.

<!-- TODO -->

## Image pulls failing { #image-pull }

Registry unreachable, rate-limited, or the requested tag does not exist for your CPU
architecture.

<!-- TODO -->

## Releases applied out of order { #release-ordering }

Resources were created before the operator that owns their custom resource definitions was
established. The errors name missing kinds rather than the ordering problem that caused them.

<!-- TODO -->

---

## Still stuck

Open an issue with the output of the preflight command and the deployment logs:
[github.com/pxcamus/debezium-platform-lab/issues](https://github.com/pxcamus/debezium-platform-lab/issues).
