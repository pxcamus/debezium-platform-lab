# Deploy the stack

Create the cluster, then apply every Helm release in dependency order: infrastructure
first, then the Debezium Operator, then the Debezium Platform.

!!! warning "Placeholder"

    Use the [README quick start](https://github.com/pxcamus/debezium-platform-lab#quick-start-local-kind)
    until this page is written.

## Configure

A single environment file drives every deployment — which environment you are targeting,
which cluster type to create, the DNS zone the ingresses are published under, and the
database credentials. Copy the template, adjust nothing for a local run, and continue.

## Create the cluster

Kind reads a cluster definition that maps ports 80 and 443 to the host so the ingress is
reachable from your browser.

## Apply the releases

Releases are labelled and applied in waves, because the operators must be running and
their custom resource definitions established before the resources that use them are
created. Applying everything at once fails in ways that look like bugs but are ordering.

## Verify

The deployment is finished when every workload is ready *and* the platform API answers.
Both are worth checking — a running pod is not a working platform.

<!-- TODO: expected output per wave; how long each wave typically takes; the teardown
     command; and what a partial failure looks like. -->

Next: [run your first pipeline](first-pipeline.md). If something failed, see
[Troubleshooting](../troubleshooting/index.md).
