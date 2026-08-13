# Deploy to k3s

Running the stack on a single Linux machine — a cloud VM, a spare box under a desk, bare
metal — rather than a Kind cluster: what changes, and what stays identical.

Almost everything stays identical. The helmfile, the charts and the `just` recipes are the
same; what moves is the *profile* — `DPL_ENV=public` instead of `local` — and a profile only
decides **how** things are configured, not **what** is installed. Real DNS, real
certificates and a different ingress controller are profile concerns. The release list is
not.

This page is provider-agnostic: bring any Linux machine and it applies unchanged.

!!! tip "If you have no machine to hand"

    The repository carries throwaway scripts that create and destroy a Hetzner Cloud VM
    for exactly this — see [Provision a Hetzner host](hetzner.md). It is a convenience,
    not a dependency; skip it entirely if you already have a host.

## What actually differs

| | Kind (`local`) | k3s (`public`) |
|---|---|---|
| Ingress controller | `ingress-nginx`, installed by the helmfile | Traefik, already bundled with k3s |
| `ingressClassName` | `nginx` | `traefik` |
| Storage class | `standard` | `local-path` |
| Hostnames | `*.127-0-0-1.sslip.io`, resolving to loopback | a zone you control, with real A records |
| TLS | none — plain HTTP | cert-manager issuing from Let's Encrypt |
| Reachability | your machine only | ports 80/443 open to the internet |

Two of those are worth dwelling on, because they are where a Kind-shaped habit breaks.

**The helmfile installs no ingress controller.** Traefik is one, and k3s ships it as part
of the distribution — it is already running in `kube-system` before you apply anything. So
[`deploy/helmfile.yaml.gotmpl`](https://github.com/pxcamus/debezium-platform-lab/blob/main/deploy/helmfile.yaml.gotmpl)
gates the `ingress-nginx` release to `local`, which is the profile that genuinely has no
controller: Kind ships nothing. What changes for you is the name — public values files that
create an Ingress ask for the `traefik` class instead of `nginx`.

**The storage class name changes.** Kind's default is `standard`; k3s's is `local-path`.
Both are backed by the same Rancher local-path provisioner, so the behaviour is the same and
only the name moves — which is exactly why it is easy to miss. A PVC naming `standard` on
k3s stays `Pending` forever with no error beyond "waiting for a volume to be created".

## Provision a host

Any Linux machine running systemd, with a public IPv4 address and ports 22, 80, 443 and
6443 reachable. k3s is not tied to a distribution — Debian, Ubuntu, Fedora, RHEL and
openSUSE all work.

Two things need attention on RPM-based distributions, and neither applies on Debian or
Ubuntu.

**SELinux is handled for you.** The upstream k3s installer at `get.k3s.io` — which is what
both the helper script and the plain `curl` command [below](#install-k3s) end up running —
pulls in the `k3s-selinux` policy when SELinux is enforcing.

**firewalld is not.** Left running, it drops pod-to-pod and pod-to-service traffic, and the
symptom never mentions a firewall: in-cluster DNS lookups time out, readiness probes fail,
and pods restart in a loop. `scripts/lab/k3s-install.sh` detects it and prints the fix but
changes nothing itself — on a disposable host, stop it; otherwise open the cluster and
service CIDRs.

**Sizing.** 8 GB runs the platform and monitoring comfortably — that is what the
[Kind walkthrough](../getting-started/deploy.md) uses. The components that change the
answer are Kafka and SSO: Strimzi and a broker, then Keycloak with its own CloudNativePG
database and two `oauth2-proxy` instances. Two of those are JVMs, and they are what make 
8 GB tight rather than comfortable. Budget 16 GB for the full single sign-on example.

Ports 80 and 443 must be reachable from anywhere, not just from your own address, as soon
as single sign-on is involved: an OIDC redirect comes back through whatever browser the
user is sitting at. Cloud firewalls that default to "my IP only" are the usual culprit.

## Install k3s

```shell
ssh <user>@<ip> 'bash -s' < scripts/lab/k3s-install.sh
```

A throwaway helper, and provider-agnostic — it runs on whatever machine you point it at,
and needs only curl and systemd. It takes the `stable` channel unless you pin a release
with `DPL_K3S_VERSION=v1.33.4+k3s1`, which is worth doing whenever the deployment has to be
reproducible later.

The one flag that matters is `--tls-san <public-ip>`, which the script derives for you.
Without it the API server certificate covers only `127.0.0.1` and the private address, the
kubeconfig you copy to your workstation fails verification, and the obvious next move is
`--insecure-skip-tls-verify` — which is how a demo cluster ends up unverified for its whole
life.

Installing by hand instead is one line, and `INSTALL_K3S_VERSION` is the equivalent pin:

```shell
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--tls-san <public-ip>" sh -
```

Keep Traefik either way.

## Point kubectl at it

k3s writes a kubeconfig on the node whose server address is `127.0.0.1`, so it has to be
rewritten in flight. Fill in the `DPL_K3S_*` variables in `.env` and one recipe does it:

```shell
just k3s-kubeconfig
```

| Variable | Purpose | Default |
|---|---|---|
| `DPL_K3S_HOST` | ssh target, and the address written into the kubeconfig | **required**, or `DPL_LAB_IP` |
| `DPL_K3S_SSH_USER` | user on that machine | `root` |
| `DPL_K3S_SSH_KEY` | private key for that user | `DPL_LAB_IDENTITY`, else ssh's own |
| `DPL_K3S_REMOTE_KUBECONFIG` | where k3s writes it on the node | `/etc/rancher/k3s/k3s.yaml` |
| `DPL_K3S_LOCAL_KUBECONFIG` | where to copy it locally | `~/.kube/<context>.yaml` |
| `DPL_K3S_CONTEXT` | context name after the copy | `dmp-demo` |

This is the only step that has to know anything about your machine. Everything after it is
`kubectl` and `helmfile`, so bring whatever you like — this VM, an EC2 instance, bare
metal — and the six values above are the whole interface.

The two `DPL_LAB_*` fallbacks exist so the Hetzner path needs no duplication: those scripts
write the address and the key they used to `~/.cache/dbz-lab/<name>.env`, and sourcing that
file before the recipe supplies both. An explicit `DPL_K3S_*` always takes precedence, so a
k3s box that is not the lab box behaves as you would expect. Ignore them entirely if you
brought your own machine.

!!! warning "`DPL_K3S_HOST` has to satisfy kubectl too, not just ssh"

    It is passed to `ssh` *and* substituted into the kubeconfig as the API server
    address. An alias defined only in `~/.ssh/config` connects happily and then yields a
    kubeconfig nothing can resolve. Use an IP or a real DNS name, and make sure the API
    certificate covers it — `--tls-san`, which
    [`k3s-install.sh`](#install-k3s) derives from the detected public address.

The recipe is a convenience, not a dependency. By hand it is:

```shell
ssh <user>@<ip> 'sudo cat /etc/rancher/k3s/k3s.yaml' \
  | sed 's#127.0.0.1#<ip>#' > ~/.kube/dmp-demo.yaml
chmod 600 ~/.kube/dmp-demo.yaml
kubectl --kubeconfig ~/.kube/dmp-demo.yaml config rename-context default dmp-demo
```

Piping through `sed` avoids both the `sed -i` incompatibility between macOS and Linux and
any need to loosen the file mode on the node.

Neither form sets `KUBECONFIG`. `just apply` targets whatever context you are already on,
so export it in the shell you deploy from:

```shell
export KUBECONFIG=~/.kube/dmp-demo.yaml
```

!!! note "`just cluster-recreate` is Kind-only, deliberately"

    A Kind cluster is disposable by construction, so recreating it costs nothing. A k3s
    cluster is a machine you brought, and the first act of a recreate would be to
    uninstall it — via `/usr/local/bin/k3s-uninstall.sh`, which only exists if k3s came
    from `get.k3s.io` at all. The recipe refuses and points at `just k3s-kubeconfig`.

## Configure

```shell
DPL_ENV=public
DPL_CLUSTER_TYPE=k3s
DPL_DOMAIN=demo.example.com
DPL_ACME_EMAIL=you@example.com
```

`DPL_DOMAIN` has to be a zone you can create records in. sslip.io does not work here:
Let's Encrypt will not issue for it, and DNS-01 validation needs to write a TXT record into
a zone you control.

Point both records at the host — the wildcard is single-level and does **not** cover the
name it hangs off:

```
demo.example.com      A   <ip>
*.demo.example.com    A   <ip>
```

## Certificates

The `public` profile installs cert-manager and expects a `ClusterIssuer` named
`letsencrypt-prod`. Kubernetes itself has no certificate concept beyond a
`kubernetes.io/tls` Secret and an `Ingress.tls` stanza naming it; cert-manager is what fills
the gap, and the issuer is what knows how to prove you own the zone.

Proving that is provider-specific, so it sits on the component axis rather than the profile
axis. Wildcards require DNS-01, and cert-manager ships built-in solvers for only a handful
of providers — Route53, CloudDNS, Cloudflare and a few others. Anything else needs an
out-of-tree webhook. This repository carries one for Gandi behind a component flag:

```yaml
components:
  dns01-gandi: true
```

with `DPL_GANDI_PAT` set. If your DNS lives elsewhere, leave the flag off and create your
own `ClusterIssuer` named `letsencrypt-prod`; nothing else changes, because every
`Certificate` in the repository references the issuer by name alone.

!!! tip "Use the staging endpoint while iterating"

    Let's Encrypt allows five *identical* certificate requests per week, and rebuilding
    the cluster asks for the same pair of names every time. Point `acme.server` at
    `https://acme-staging-v02.api.letsencrypt.org/directory` until the deployment is
    settled, then switch. Staging certificates are untrusted by browsers, which is the
    point — they cost nothing.

## Deploy

Unchanged from the local flow:

```shell
just releases
just apply
```

Grafana is deliberately not published on this profile. Its Ingress would need a TLS Secret
from another namespace, and a static admin password would be the one unauthenticated host
on the domain. Reach it directly instead:

```shell
kubectl -n monitoring port-forward svc/kube-prometheus-stack-grafana 3000:80
```

For putting authentication in front of the platform itself, see
[Single sign-on](sso.md).

## Persisting data across restarts

`local-path` volumes are directories under `/var/lib/rancher/k3s/storage` on the node. They
survive pod restarts, node reboots and `helmfile destroy`, because a PVC outlives the
release that created it.

Two consequences:

- **They are not backed up and not replicated.** One machine, one disk. This is a lab, not
  a durable home for anything you care about.
- **A stale PVC can outlive its chart.** If a database comes back with data you expected to
  be gone, the volume was reused. `kubectl get pvc -A` shows what survived; deleting the
  PVC is what actually resets it.

Disk fills quietly. Prometheus is configured for 20Gi with 15 days of retention, which is
the largest single consumer.

## Tear down

```shell
sudo /usr/local/bin/k3s-uninstall.sh   # on the node
```

Delete the DNS records too. A dangling A record pointing at a recycled cloud address is
worth more to somebody else than it is to you.

If the host itself was disposable, destroying it is simpler than uninstalling anything —
see [Tear down](hetzner.md#tear-down) for the Hetzner case.
