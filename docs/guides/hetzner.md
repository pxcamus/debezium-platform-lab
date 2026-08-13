# Provision a Hetzner host

Nothing in the platform depends on [Hetzner](https://hetzner.com), and nothing on the deployment path calls these
scripts. Everything here is a convenience: an EC2 instance, a spare laptop or bare metal
works just as well, and the k3s guide is written for any of them.

The material here is provided as-is to help you quickly get setup on Hetzner, a European cloud provider known for 
being significantly discounted vs US hyperscalers.

!!! warning "These scripts create billable resources"

    A server and a firewall, billing by the hour until `hcloud-down.sh` runs. Ranging from
    single-digit €/month if left running, to ~€85 for the SSO stack. `hcloud-up.sh` prints what it is
    about to create and asks before doing it.

## TL;DR

From nothing to a k3s cluster your workstation can drive. The one step that is not a
command is the API token: it comes from the Cloud Console, under
**Security → API tokens → Generate API token**, with **Read & Write**.

```shell
# hcloud CLI — or a distro package, or a release binary from hetznercloud/cli
brew install hcloud

# authenticate; prompts once for the token, then remembers it
hcloud context create dbz-lab

# SSH key: generated locally, only the public half is uploaded
ssh-keygen -t ed25519 -f ~/.ssh/dmp-demo -C dmp-demo
hcloud ssh-key create --name dmp-demo --public-key-from-file ~/.ssh/dmp-demo.pub

# tell the scripts about it
cat >> .env <<'EOF'
DPL_LAB_SSH_KEY=dmp-demo
DPL_LAB_IDENTITY=~/.ssh/dmp-demo
DPL_LAB_TYPE=cpx42
DPL_LAB_OPEN_HTTP=true
EOF

# create the machine — about forty seconds
scripts/with-env.sh scripts/lab/hcloud-up.sh

# install k3s on it
source ~/.cache/dbz-lab/dbz-lab.env
ssh -i "${DPL_LAB_IDENTITY}" "${DPL_LAB_SSH}" 'bash -s' < scripts/lab/k3s-install.sh

# point this workstation at the cluster
just k3s-kubeconfig
export KUBECONFIG=~/.kube/dmp-demo.yaml

# destroy everything — it bills until you do
scripts/lab/hcloud-down.sh
```

Then continue with [Deploy to k3s](k3s.md#configure). The rest of this page explains those
commands and what each variable is for — including `DPL_LAB_TYPE` and `DPL_LAB_OPEN_HTTP`,
which are fixed when the machine is created and cost a rebuild to change.

## Prerequisites

The `hcloud` CLI, which is deliberately not in the
[project prerequisites](../getting-started/prerequisites.md) — nothing on the deployment
path uses it. Install it from [hetznercloud/cli](https://github.com/hetznercloud/cli/blob/main/docs/tutorials/setup-hcloud-cli.md).

### The API token

`HCLOUD_TOKEN` is a Hetzner Cloud API token. It authorizes the CLI against exactly one
project rather than your account, so a token for a scratch project cannot reach anything
else you run there. It also makes deleting billable resources easy and safe.

Create it in the Cloud Console: select the project, then **Security → API tokens →
Generate API token**, with **Read & Write** — the scripts create and delete servers and
firewalls. The value is displayed once and never again.

Either export it, or hand it over once and let the CLI keep it:

```shell
hcloud context create dbz-lab
```

`hcloud-up.sh` accepts either, and reads no credential from this repository.

### The SSH key

Two halves, configured separately. Generate a pair and upload the public half to the
project:

```shell
ssh-keygen -t ed25519 -f ~/.ssh/dmp-demo -C dmp-demo
hcloud ssh-key create --name dmp-demo --public-key-from-file ~/.ssh/dmp-demo.pub
```
`DPL_LAB_SSH_KEY` is the *name* in the project (`dmp-demo` above), which is what Hetzner
injects into the new machine; `hcloud ssh-key list` shows what you have.
`DPL_LAB_IDENTITY` is the path to the *private* half, passed to `ssh -i`.

### What goes in `.env`

Everything this page needs, in one block:

```shell
HCLOUD_TOKEN=<from the console>     # or run `hcloud context create` instead
DPL_LAB_SSH_KEY=dmp-demo            # the key's NAME in your Hetzner project
DPL_LAB_IDENTITY=~/.ssh/dmp-demo    # the matching PRIVATE key on this machine, no .pub
DPL_LAB_TYPE=cpx42                  # the cpx32 default is 8 GB — too tight for SSO
DPL_LAB_OPEN_HTTP=true              # 80/443 from anywhere, not just your address
```

`.env` is git-ignored, so the token is safe there.

## Create the machine

!!! note "`.env` does not reach these scripts on its own"

    Unlike the `just` recipes, `hcloud-up.sh` does not source
    [`scripts/lib/env.sh`](https://github.com/pxcamus/debezium-platform-lab/blob/main/scripts/lib/env.sh)
    — it reads the environment as it finds it. To keep the values in `.env` instead of
    exporting them by hand, front the script with the loader:

```shell
scripts/with-env.sh scripts/lab/hcloud-up.sh
```

!!! warning "Billable resources"

    A CPX42 machine is currently advertised at €85/month.



About forty seconds later you have a bare Ubuntu box with one non-root user (`lab`) and
nothing else installed — no docker, no kubectl, no helm. That is deliberate: a bare machine
is the only honest answer to "what does this project actually require", and a bootstrapped
one can no longer tell you.

The address is cached outside the repository, so you never have to paste an IP:

```shell
source ~/.cache/dbz-lab/dbz-lab.env
ssh -i "${DPL_LAB_IDENTITY}" "${DPL_LAB_SSH}"
```

### What it creates

| Resource | Detail                                                                                   |
|---|------------------------------------------------------------------------------------------|
| Server | `cpx32` by **default** — Ubuntu 24.04, amd64, in `nbg1`                                  |
| Firewall | ssh (22) and the Kubernetes API (6443) from your current IP only                         |
| | http/https (80, 443) from your current IP, or the internet with `DPL_LAB_OPEN_HTTP=true` |

The amd64 is used because images for the platform are amd64-only -- the only multi-platform tag is `nightly`.

Every resource carries the label `lab=dbz-platform`, and `hcloud-down.sh` acts only on that
selector — it takes no server name. If your project holds anything you care about, the
teardown cannot reach it.

### Settings

| Variable | Purpose | Default                |
|---|---|------------------------|
| `DPL_LAB_SSH_KEY` | key name in your Hetzner project | **required**           |
| `DPL_LAB_IDENTITY` | local private key to connect with | ssh's own defaults     |
| `DPL_LAB_NAME` | server name; also names the state file | `dbz-lab`              |
| `DPL_LAB_TYPE` | server type | `cpx32` (4 vCPU, 8 GB) |
| `DPL_LAB_IMAGE` | OS image | `ubuntu-24.04`         |
| `DPL_LAB_LOCATION` | datacenter | `nbg1`                 |
| `DPL_LAB_OPEN_HTTP` | expose 80/443 to the internet | `false`                |
| `DPL_LAB_YES` | skip the confirmation prompt | `false`                |

**`DPL_LAB_OPEN_HTTP=true` is required for single sign-on.** The default firewall admits
80/443 from your own address only, which will break the moment
an OIDC redirect has to come back through a browser on another network.

**`DPL_LAB_TYPE=cpx42` for the SSO example.** The `cpx32` default is 8 GB, which is enough
for the platform and monitoring but tight once Kafka and the identity chain are running —
see the sizing note in [Deploy to k3s](k3s.md#provision-a-host).

!!! tip "Server types are retired per location"

    A type can be *supported* in a location and no longer *orderable* there — `cpx31` is
    the standing example in `nbg1`. `hcloud-up.sh` checks against
    `hcloud server-type list` first and, if the type is unavailable, prints the locations
    where it is. The check skips itself if that listing cannot be read, so it can never
    become a false blocker.

## Next: install k3s

The machine is bare. Continue with [Install k3s](k3s.md#install-k3s), which is
provider-agnostic from here on:

```shell
source ~/.cache/dbz-lab/dbz-lab.env
ssh -i "${DPL_LAB_IDENTITY}" "${DPL_LAB_SSH}" 'bash -s' < scripts/lab/k3s-install.sh
```

Then copy the kubeconfig back. On this path that needs no new variables at all —
`hcloud-up.sh` already recorded the address and the key it used, so sourcing its state file
supplies both:

```shell
source ~/.cache/dbz-lab/dbz-lab.env
just k3s-kubeconfig
export KUBECONFIG=~/.kube/dmp-demo.yaml
```

That last line is what actually points `kubectl` and `helmfile` at the new cluster —
nothing in this repository sets `KUBECONFIG` for you, so `just apply` would otherwise
target whatever context you were already on. The recipe prints the exact path it wrote, and
the export belongs in every shell you deploy from.

Worth doing it this way rather than writing the address into `.env`: rebuilding the machine
changes the address, the state file follows, and a hand-copied one would not.

Bringing your own machine instead means setting the `DPL_K3S_*` block yourself — see
[Point kubectl at it](k3s.md#point-kubectl-at-it).

There is also `scripts/lab/bootstrap.sh`, which installs a *workstation* toolchain — helm,
helmfile, just, docker and Kind. A machine that only runs k3s does not need it; helmfile
drives the cluster from your laptop over the fetched kubeconfig.

## Tear down

```shell
scripts/lab/hcloud-down.sh
```

Deletes every resource labelled `lab=dbz-platform` and nothing else. Both scripts confirm
before acting unless `DPL_LAB_YES=true`.

