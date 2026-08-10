# Lab machine (maintainer tooling)

Nothing here is part of deploying the platform. This is how we answer *"does this work on a
machine that has never seen the project?"* — repeatedly, which is impossible on a machine
already fixed by hand.

> **`hcloud-up.sh` creates billable cloud resources.** They bill by the hour until
> `hcloud-down.sh` runs. Roughly single-digit €/month if left running; cents per test.

## The pieces

| File | Scope |
|---|---|
| `bootstrap.sh` | Installs the *workstation* toolchain — docker, kubectl, helm, helmfile, kind, just. Debian/Ubuntu only; it shells out to `apt-get`. |
| `k3s-install.sh` | Installs k3s on a *cluster host*, configured for `DPL_ENV=public`. Needs only curl and systemd, so any distribution works. |
| `hcloud-up.sh` / `hcloud-down.sh` | A thin Hetzner Cloud wrapper. Convenience only; nothing depends on Hetzner. |
| `cloud-init.yaml` | Creates one non-root user and installs nothing else, on purpose. |

`bootstrap.sh` and `k3s-install.sh` are not a sequence. The first sets up a workstation (it
installs Kind, which a k3s host has no use for); the second sets up a cluster host. A
machine that only runs k3s needs the second alone — helmfile drives it from your laptop
over the fetched kubeconfig.

## Documentation

The how-to lives in the guides, so it stays in one place:

- [**Provision a Hetzner host**](https://lab.1int.io/guides/hetzner/) — the `hcloud` CLI,
  the API token, the SSH key pair, every `DPL_LAB_*` setting, and teardown.
  ([source](../../docs/guides/hetzner.md))
- [**Deploy to k3s**](https://lab.1int.io/guides/k3s/) — `k3s-install.sh`, fetching the
  kubeconfig, and what changes between the `local` and `public` profiles.
  ([source](../../docs/guides/k3s.md))

Each script's own header documents its variables and the reasoning behind its defaults.

## Bootstrapping a workstation

The one piece no guide covers, because it is not on the deployment path. `bootstrap.sh`
never reads stdin, so piping it over ssh is safe and leaves nothing behind:

```bash
ssh -i ~/.ssh/my-key lab@<ip> 'bash -s' < scripts/lab/bootstrap.sh
```

`hcloud-up.sh` caches the address, so you need not paste an IP. The state file is named
after `DPL_LAB_NAME`:

```bash
source ~/.cache/dbz-lab/dbz-lab.env
ssh -i "${DPL_LAB_IDENTITY}" "${DPL_LAB_SSH}" 'bash -s' < scripts/lab/bootstrap.sh
```

**Reconnect afterwards.** It adds `lab` to the `docker` group, which does not apply to the
session that ran it — a `docker` command in that same session still fails.

## Order of operations

Note what the **bare** box is missing before you bootstrap it — that list is the real
prerequisite list, and a bootstrapped machine can no longer tell you what it needed.

If you fix something by hand over SSH, the test is void. That fix belongs in a script or in
the guides; then destroy the box and start again.

## Secrets

Nothing sensitive is committed, and the scripts refuse to run rather than fall back to a
default:

- **Hetzner token** — read from `HCLOUD_TOKEN` or an active `hcloud context`. Never printed.
- **SSH key** — referenced by its *name* in your Hetzner project. `DPL_LAB_IDENTITY` is a
  path handed to `ssh -i`; no key material is read by the scripts themselves.
- **Your IP** — resolved at runtime for the firewall rule, never written down.
- **Server address** — cached under `~/.cache/dbz-lab/`, deliberately outside the repository.

Every resource is labelled `lab=dbz-platform`, and `hcloud-down.sh` acts only on that
selector — it takes no server name, so it cannot reach anything else in your project.
