package runner

import (
	"context"
	"fmt"

	"dbz-mage/ko/automation"
)

type SSH struct {
	User    string
	Host    string
	KeyPath string
}

func (s SSH) Address() string {
	return fmt.Sprintf("%s@%s", s.User, s.Host)
}

func (s SSH) Run(ctx context.Context, command string, args ...string) error {
	sshArgs := []string{"-i", automation.ExpandPath(s.KeyPath), s.Address(), command}
	return automation.Run(ctx, "ssh", sshArgs...)
}

func (s SSH) Output(ctx context.Context, command string, args ...string) ([]byte, error) {
	sshArgs := []string{"-i", automation.ExpandPath(s.KeyPath), s.Address(), command}
	return automation.Output(ctx, "ssh", sshArgs...)
}

func (s SSH) CopyFrom(ctx context.Context, remotePath string, localPath string) error {
	source := fmt.Sprintf("%s:%s", s.Address(), remotePath)
	return automation.Run(ctx, "scp", "-i", automation.ExpandPath(s.KeyPath), source, automation.ExpandPath(localPath))
}
