# Contributing

Thanks for your interest in this project. It automates deploying and managing the
[Debezium Platform](https://debezium.io/documentation/reference/stable/operations/debezium-platform.html)
on Kubernetes. This guide covers how to set up a local environment and how changes
land in `main`.

## Development setup

Install the tooling listed in the [README prerequisites](README.md#prerequisites)
(Go 1.26+, mage, helm + helmfile, kind, kubectl, Docker), then:

```bash
cp .env.example .env      # required — mage targets and validate-helm.sh load it
mage -l                   # list available targets
```

`.env` is git-ignored and **required**: mage targets call `automation.LoadEnv()`
and `scripts/validate-helm.sh` sources it, so without it required variables such
as `DBZ_VERSION` are missing. The passwords in the Helm charts and `.env.example`
are non-secret demo values — never commit real secrets.

## Branching and pull requests

`main` is protected. All changes — including docs and chores — land through a pull
request; direct pushes to `main` are not accepted.

```bash
git checkout -b <type>/<short-description>   # e.g. fix/scenario-loader-name
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

Run the same checks CI runs, locally:

```bash
gofmt -l .                 # should print nothing
go vet ./...
go build ./...
go run github.com/magefile/mage -l   # compiles the mage-tagged magefile
scripts/validate-helm.sh   # offline Helm chart + helmfile validation
```

CI enforces these on every PR:

- **Go** ([`.github/workflows/go.yaml`](.github/workflows/go.yaml)) — gofmt,
  `go vet`, `go build`, `go mod tidy` check, and a magefile compile, on any `*.go`
  / `go.mod` / `go.sum` change.
- **Helm validation** ([`.github/workflows/helm-validation.yaml`](.github/workflows/helm-validation.yaml)) —
  `helm lint` / `template` and `helmfile lint` / `template` through `kubeconform`,
  on any change under `deploy/`.

## Coding conventions

These match the existing codebase (see [`CLAUDE.md`](CLAUDE.md) for the full set):

- **`NewFromEnv()`** factory pattern: read env vars, build config, call `New()`.
- Env access via `automation.Env(name, fallback)` /
  `automation.RequiredEnv(name)`; paths via `automation.ExpandPath(path)`.
- Structured logging with `log/slog` via `slog.Default()`.
- Wrap errors with `fmt.Errorf("...: %w", err)`.
- No inline comments — only Go doc comments on exported symbols.
- All DMP JSON payloads must have a `"name"` field; `${ENV_VAR}` in payloads is
  expanded at load time.

## Reporting issues

Open a GitHub issue with enough detail to reproduce: the mage target or command
you ran, your `DBZ_ENV` / `CLUSTER_TYPE`, and the relevant log output (set
`LOG_LEVEL=debug` for more). Please redact any real secrets.
