package cluster

import (
	"context"
	"dbz-mage/ko/automation"
	"dbz-mage/ko/runner"
)

type Kind struct {
	Runner runner.Runner
	Name   string
	Config string
}

func NewKindFromEnv() *Kind {
	return &Kind{
		Runner: runner.Local{},
		Name:   automation.Env("KIND_CLUSTER_NAME", "dmp"),
		Config: automation.Env("KIND_CONFIG", "deploy/clusters/kind/kind-ingress.yaml"),
	}
}

func (k *Kind) Recreate(ctx context.Context) error {
	if err := k.Delete(ctx); err != nil {
		return err
	}

	if err := k.Create(ctx); err != nil {
		return err
	}

	return k.UseContext(ctx)
}

func (k *Kind) Create(ctx context.Context) error {
	args := []string{"create", "cluster"}

	if k.Name != "" {
		args = append(args, "--name", k.Name)
	}

	if k.Config != "" {
		args = append(args, "--config", k.Config)
	}

	return k.Runner.Run(ctx, "kind", args...)
}

func (k *Kind) Delete(ctx context.Context) error {
	args := []string{"delete", "cluster"}

	if k.Name != "" {
		args = append(args, "--name", k.Name)
	}

	return k.Runner.Run(ctx, "kind", args...)
}

func (k *Kind) UseContext(ctx context.Context) error {
	return k.Runner.Run(ctx, "kubectl", "config", "use-context", "kind-"+k.Name)
}
