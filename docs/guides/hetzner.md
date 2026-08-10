# Provision a Hetzner host

A disposable Hetzner Cloud VM, created and destroyed by two scripts. It exists so
[Deploy to k3s](k3s.md) can start from a machine that has never seen this project — which
is impossible on a machine you have already fixed by hand.

Nothing in the platform depends on Hetzner, and nothing on the deployment path calls these
scripts. Everything here is a convenience: an EC2 instance, a spare laptop or bare metal
works just as well, and the k3s guide is written for any of them.

!!! warning "These scripts create billable resources"

    A server and a firewall, billing by the hour until `hcloud-down.sh` runs. Roughly
    single-digit €/month if left running, cents per test. `hcloud-up.sh` prints what it is
    about to create and asks before doing it.

## Prerequisites

The `hcloud` CLI, which is deliberately **not** in the
[project prerequisites](../getting-started/prerequisites.md) — nothing on the deployment
path uses it. Install it from [hetznercloud/cli](https://github.com/hetznercloud/cli):
`brew install hcloud`, a package on most distributions, or a release binary.

### The API token

`HCLOUD_TOKEN` is a Hetzner Cloud API token. It authorises the CLI against exactly one
*project* rather than your account, so a token for a scratch project cannot reach anything
else you run there.

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

Both are needed because ssh offers only the default identities (`~/.ssh/id_*`) plus
whatever the agent holds. A key stored under its own name is never offered, ssh falls
through to password authentication, and the image locks the password — so the script waits
five minutes and reports a timeout on a machine that is in fact perfectly healthy. Leave
`DPL_LAB_IDENTITY` unset only if the key is an ssh default or already loaded in your agent.

## Create the machine

```shell
export HCLOUD_TOKEN=...
export DPL_LAB_SSH_KEY=dmp-demo
export DPL_LAB_IDENTITY=~/.ssh/dmp-demo
DPL_LAB_TYPE=cpx41 DPL_LAB_OPEN_HTTP=true scripts/lab/hcloud-up.sh
```

!!! note "`.env` does not reach these scripts on its own"

    Unlike the `just` recipes, `hcloud-up.sh` does not source
    [`scripts/lib/env.sh`](https://github.com/pxcamus/debezium-platform-lab/blob/main/scripts/lib/env.sh)
    — it reads the environment as it finds it. To keep the values in `.env` instead of
    exporting them by hand, front the script with the loader:

    ```shell
    scripts/with-env.sh scripts/lab/hcloud-up.sh
    ```

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

| Resource | Detail |
|---|---|
| Server | `cpx32` by default — Ubuntu 24.04, amd64, in `nbg1` |
| Firewall | ssh (22) and the Kubernetes API (6443) from your current IP only |
| | http/https (80, 443) from your current IP, or the internet with `DPL_LAB_OPEN_HTTP=true` |

amd64 is deliberate: the Debezium Platform release images are amd64-only, so an arm64 box
fails for reasons unrelated to whatever you are testing.

Every resource carries the label `lab=dbz-platform`, and `hcloud-down.sh` acts only on that
selector — it takes no server name. If your project holds anything you care about, the
teardown cannot reach it.

### Settings

| Variable | Purpose | Default |
|---|---|---|
| `DPL_LAB_SSH_KEY` | key name in your Hetzner project | **required** |
| `DPL_LAB_IDENTITY` | local private key to connect with | ssh's own defaults |
| `DPL_LAB_NAME` | server name; also names the state file | `dbz-lab` |
| `DPL_LAB_TYPE` | server type | `cpx32` (4 vCPU, 8 GB) |
| `DPL_LAB_IMAGE` | OS image | `ubuntu-24.04` |
| `DPL_LAB_LOCATION` | datacenter | `nbg1` |
| `DPL_LAB_OPEN_HTTP` | expose 80/443 to the internet | `false` |
| `DPL_LAB_YES` | skip the confirmation prompt | `false` |

**`DPL_LAB_OPEN_HTTP=true` is required for single sign-on.** The default firewall admits
80/443 from your own address only, which is fine for a private trial and useless the moment
an OIDC redirect has to come back through a browser on another network.

**`DPL_LAB_TYPE=cpx41` for the SSO example.** The `cpx32` default is 8 GB, which is enough
for the platform and monitoring but tight once Kafka and the identity chain are running —
see the sizing note in [Deploy to k3s](k3s.md#provision-a-host).

!!! tip "Server types are retired per location"

    A type can be *supported* in a location and no longer *orderable* there — `cpx31` is
    the standing example in `nbg1`. The API only says so after the firewall exists and you
    have answered the confirmation, so `hcloud-up.sh` checks against
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

There is also `scripts/lab/bootstrap.sh`, which installs a *workstation* toolchain — helm,
helmfile, just, docker and Kind. A machine that only runs k3s does not need it; helmfile
drives the cluster from your laptop over the fetched kubeconfig.

## Tear down

```shell
scripts/lab/hcloud-down.sh
```

Deletes every resource labelled `lab=dbz-platform` and nothing else. Both scripts confirm
before acting unless `DPL_LAB_YES=true`.

Delete the DNS records too, if you created any. A dangling A record pointing at a recycled
cloud address is worth more to somebody else than it is to you.

!!! note "If you fixed something by hand, the test is void"

    That fix belongs in `scripts/lab/bootstrap.sh`, in `scripts/lab/k3s-install.sh` or in
    these pages. Then destroy the box and start again — that is the entire point of it
    being disposable.
