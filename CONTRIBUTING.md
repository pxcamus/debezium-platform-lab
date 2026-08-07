# Contributing

Thanks for your interest in this project. It automates deploying and managing the
[Debezium Platform](https://debezium.io/documentation/reference/stable/operations/debezium-platform.html)
on Kubernetes. This guide covers how to set up a local environment and how changes
land in `main`.

## Development setup

Install the tooling listed in the
[documentation prerequisites](https://lab.1int.io/getting-started/prerequisites/)
(just, helm + helmfile, kind, kubectl, Docker), then:

```bash
cp .env.example .env       # host-specific values and secrets
scripts/validate-helm.sh   # offline check that every chart still renders
```

`.env` is git-ignored. It layers over
[`deploy/environment/versions.env`](deploy/environment/versions.env), which
carries the shared version pins, so a checkout without `.env` still renders —
that is how CI runs. Put host-specific values and secrets in `.env` and version
pins in `versions.env`, never the reverse. The passwords in the Helm charts and
`.env.example` are non-secret demo values — never commit real secrets.

> **No Go, no `mage`.** The Go library, the mage task runner and the `dmp-lab`
> CLI were removed; the toolchain is now [`just`](https://just.systems/) over
> `helmfile`. This repository is charts, helmfile releases, JSON payloads, shell
> and docs. `just --list` shows every task.

## Branching and pull requests

`main` is protected. All changes — including docs and chores — land through a pull
request; direct pushes to `main` are not accepted.

```bash
git checkout -b <type>/<short-description>   # e.g. fix/otel-webhook-certs
# ... make your changes ...
git push -u origin <type>/<short-description>
gh pr create --fill
```

- Keep PRs focused and atomic — one logical change per PR, so it can be reviewed
  and reverted cleanly.
- Fill in the pull request template and make sure CI is green before merging.
- Squash-merge is preferred to keep `main` history linear:
  `gh pr merge --squash`.
- Solo maintainer? Still open a PR: it runs CI before merge, gives you a
  fresh-eyes self-review, and keeps the public history reviewable.

Suggested branch/commit prefixes: `feat/`, `fix/`, `docs/`, `chore/`, `refactor/`,
`ci/`.

## Before you open a PR

Run the same checks CI runs, locally. None of them need a cluster:

```bash
scripts/validate-helm.sh                                    # charts, gating, every environment
scripts/validate-env.sh                                     # env-var consistency
shellcheck scripts/*.sh scripts/lab/*.sh scripts/lib/*.sh
just --fmt --check --unstable                               # justfile formatting
```

CI enforces these on every PR:

- **Toolchain** ([`.github/workflows/toolchain.yaml`](.github/workflows/toolchain.yaml)) —
  shellcheck, `bash -n`, `just --fmt --check`, `just --list` and `validate-env.sh`.
  No path filter: it runs on every PR, including docs-only ones.
- **Helm validation** ([`.github/workflows/helm-validation.yaml`](.github/workflows/helm-validation.yaml)) —
  `scripts/validate-helm.sh`: `helmfile lint`, the release-gating assertions, and a
  `template` of every environment through `kubeconform`. Runs on any change under
  `deploy/`; reports a trivial pass otherwise, so the required check never blocks an
  unrelated PR.
- **Docs** ([`.github/workflows/docs.yaml`](.github/workflows/docs.yaml)) — MkDocs
  builds `docs/` and publishes to [lab.1int.io](https://lab.1int.io/).

If `just --fmt --check` fails, `just --fmt --unstable` fixes it in place.

## Conventions

See [`AGENTS.md`](AGENTS.md) for the full set:

- Values files are per-component and per-environment:
  `deploy/values/<component>/<DPL_ENV>.yaml.gotmpl`.
- Prefer adding a values layer over forking a chart.
- Version pins go in `versions.env`, secrets and host overrides in `.env`.
- **Every new release needs an explicit `installed:` gate.** Helmfile defaults it to
  `true`, so a release added without one installs for everybody — either gate it on
  `DPL_ENV` (how to configure) or on a `.Values.components` key (what to install).
  `scripts/validate-helm.sh` asserts the default install is exactly four releases and
  will fail if you add a fifth.
- Run `just --fmt --unstable` after editing the `justfile`; CI checks it.
- All DMP JSON payloads must have a `"name"` field; `${ENV_VAR}` in payloads is
  expanded at load time, and a missing variable becomes an empty string rather
  than an error.

## Reporting issues

Open a GitHub issue with enough detail to reproduce: the command you ran, your
`DPL_ENV` / `DPL_CLUSTER_TYPE`, and the relevant output. Please redact any real
secrets.
