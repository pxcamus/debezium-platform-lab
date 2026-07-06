package runner

import (
	"context"
	"dbz-mage/ko/automation"
	"fmt"
)

type Local struct{}

func (Local) Run(ctx context.Context, command string, args ...string) error {
	return automation.Run(ctx, command, args...)
}

func (Local) Output(ctx context.Context, command string, args ...string) ([]byte, error) {
	return automation.Output(ctx, command, args...)
}

func (Local) CopyFrom(ctx context.Context, remotePath string, localPath string) error {
	return fmt.Errorf("copy from is not supported for local runner: %s -> %s", remotePath, localPath)
}
