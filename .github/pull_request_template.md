<!-- Keep PRs focused: one logical change per PR. -->

## What & why

<!-- What does this change and why? Link any related issue: Closes #123 -->

## Type of change

- [ ] Bug fix (non-breaking)
- [ ] New feature (non-breaking)
- [ ] Breaking change (behavior/interface/values changes existing users must react to)
- [ ] Docs / chore / CI only

## How was this tested?

<!-- Commands run, cluster type (kind/k3s), DBZ_ENV, scenario, etc. -->

## Checklist

- [ ] `gofmt -l .` is clean, `go vet ./...` and `go build ./...` pass
- [ ] `go run github.com/magefile/mage -l` compiles (if magefile/targets changed)
- [ ] `scripts/validate-helm.sh` passes (if anything under `deploy/` changed)
- [ ] No real secrets committed (demo values only)
- [ ] Breaking changes are called out above and documented (README / values)
