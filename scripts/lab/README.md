# Lab machine (maintainer tooling)

Nothing here is part of deploying the platform. This is how we answer *"does this work on a
machine that has never seen the project?"* — repeatedly, which is impossible on a machine
already fixed by hand.

> **These scripts create billable cloud resources.** They bill by the hour until
> `hcloud-down.sh` runs. Roughly single-digit €/month if left running; cents per test.

## The two halves

| Script | Scope |
|---|---|
| `bootstrap.sh` | Provider-agnostic. Installs the prerequisites on any fresh Debian/Ubuntu machine — a cloud VM, a spare laptop, bare metal. This is the part that matters. |
| `hcloud-up.sh` / `hcloud-down.sh` | A thin Hetzner Cloud wrapper. Convenience only; nothing depends on Hetzner. |
| `cloud-init.yaml` | Creates one non-root user and installs nothing else, on purpose. |

## Use

```bash
export HCLOUD_TOKEN=...             # or: hcloud context create dbz-lab
export LAB_SSH_KEY=my-key           # name from 'hcloud ssh-key list'
export LAB_IDENTITY=~/.ssh/my-key   # the matching private key, if not an ssh default

scripts/lab/hcloud-up.sh            # ~40s to a bare Ubuntu box
ssh -i ~/.ssh/my-key lab@<ip>
scripts/lab/hcloud-down.sh          # deletes everything it created
```

`LAB_SSH_KEY` names the *public* key in your Hetzner project; `LAB_IDENTITY` points at the
*private* half on this machine. They are separate because ssh only offers the default
identities (`~/.ssh/id_*`) plus whatever the agent holds — a lab key stored under its own
name is never offered, ssh falls through to password auth, and `cloud-init.yaml` locks the
password. Leave `LAB_IDENTITY` unset if your lab key is an ssh default or is loaded in the
agent.

Defaults to a 4 vCPU / 8 GB amd64 box (`cpx32`) running Ubuntu 24.04 in `nbg1`. amd64 is
deliberate — the Debezium Platform release images are amd64-only. Override with `LAB_TYPE`,
`LAB_IMAGE`, `LAB_LOCATION`, `LAB_NAME`; see the header of `hcloud-up.sh`.

Hetzner retires server types per location, so a type can be *supported* somewhere and no
longer *orderable* there — the API only says so after the firewall exists and you have
confirmed the prompt. `preflight()` checks the type against `hcloud server-type list` first
and, if it is unavailable, prints the locations where it is. The check skips itself if that
listing cannot be read, so it can never become a false blocker.

## Order of operations

Run the preflight check on the **bare** box before bootstrapping it. Whatever the check
fails to report is a gap in the check, not in the machine — and a bootstrapped machine can
no longer tell you that. Bootstrap second, once you want to get on with a deployment.

If you fix something by hand over SSH, the test is void. That fix belongs in the preflight
check or in the docs; then destroy the box and start again.

## Secrets

Nothing sensitive is committed, and the scripts refuse to run rather than fall back to a
default:

- **Hetzner token** — read from `HCLOUD_TOKEN` or an active `hcloud context`. Never printed.
- **SSH key** — referenced by its *name* in your Hetzner project. `LAB_IDENTITY` is a path
  handed to `ssh -i`; no key material is read by the scripts themselves.
- **Your IP** — resolved at runtime for the firewall rule, never written down.
- **Server address** — cached under `~/.cache/dbz-lab/`, deliberately outside the repository.

## Safety

Every resource is labelled `lab=dbz-platform`, and `hcloud-down.sh` acts only on that
selector — it takes no server name. If your Hetzner project holds anything you care about,
the teardown cannot reach it. Both scripts confirm before acting unless `LAB_YES=true`.

By default the firewall exposes SSH, the Kubernetes API and HTTP/HTTPS to your current IP
only. `LAB_OPEN_HTTP=true` opens 80/443 to the internet — remember that the lab deploys no
authentication in front of the platform.
