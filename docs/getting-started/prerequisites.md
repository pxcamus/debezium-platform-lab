# Prerequisites

Five tools and a container runtime. Everything else the lab needs — charts, operators,
images — is pulled during the deployment.

| Tool | Purpose |
|---|---|
| [just](https://just.systems/man/en/packages.html) | task runner; every command on this path is a `just` recipe |
| [helmfile](https://helmfile.readthedocs.io/en/latest/#installation) + [helm](https://helm.sh/) | chart orchestration |
| [kind](https://kind.sigs.k8s.io/) | local Kubernetes cluster |
| [kubectl](https://kubernetes.io/docs/tasks/tools/) | cluster access |
| [Docker](https://www.docker.com/) | container runtime Kind runs on |

The [helm-diff](https://github.com/databus23/helm-diff) plugin is worth installing —
helmfile uses it to show what an apply will change before it changes it:

```shell
helm plugin install https://github.com/databus23/helm-diff
```

Optional: [kubeconform](https://github.com/yannh/kubeconform), used by `scripts/validate-helm.sh`
and CI to check rendered manifests against the Kubernetes schemas. You only need it if you
plan to run that validation locally.

Machine capacity is on the [previous page](index.md#what-this-costs-you).

## Check what you have

```shell
just --version && helmfile --version && kind --version && kubectl version --client && docker version --format '{{.Server.Version}}'
```

Example of output:
```shell
just 1.58.0
helmfile version v1.7.3
kind version 0.32.0
Client Version: v1.36.3
Kustomize Version: v5.8.1
29.6.2
```
??? note "Why does `kubectl version` mention Kustomize?"

    `kubectl` embeds Kustomize as a built-in library, so `kubectl version` reports the vendored Kustomize version
    alongside its own — nothing here uses or installs it separately.

Once this command works, [deploy the stack](deploy.md).
