package runner

import "context"

type Runner interface {
	Run(ctx context.Context, command string, args ...string) error
	Output(ctx context.Context, command string, args ...string) ([]byte, error)
	CopyFrom(ctx context.Context, remotePath string, localPath string) error
}
