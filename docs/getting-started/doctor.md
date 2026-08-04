# Check your machine

Before deploying anything, one command inspects the machine and reports what is missing,
what is misconfigured, and the exact command to fix each finding.

!!! warning "Placeholder"

    The preflight command is not implemented yet. Until it exists, work through the
    [prerequisites table in the README](https://github.com/pxcamus/debezium-platform-lab#prerequisites)
    by hand.

## What it checks

Roughly four families of check, each reported with a stable identifier:

- **Tooling** — container runtime, `kubectl`, `helm`, `helmfile`, `kind`, and their versions.
- **Capacity** — free memory, CPUs and disk against what the selected profile needs.
- **Host configuration** — kernel limits that a busy Kubernetes node exhausts, free ports
  for the ingress, and DNS resolution for the platform hostnames.
- **Target** — which cluster and namespace the environment currently resolves to, so you
  learn *before* deploying that your context points somewhere unexpected.

## Reading the output

Each finding carries an identifier such as `inotify-limits`, and each identifier has a
matching section in [Troubleshooting](../troubleshooting/index.md) reached by the link the
command prints. If a check fails you should never have to search for the fix.

<!-- TODO: paste real output once the command lands; the check catalog is drafted in the
     private ops notes (doctor design plan) and should be ported here as the source of
     truth for the identifiers. -->

Once every check passes, [deploy the stack](deploy.md).
