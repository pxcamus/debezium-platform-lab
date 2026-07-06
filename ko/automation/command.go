package automation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func Run(ctx context.Context, command string, args ...string) error {
	fmt.Printf("Running: %s %v\n", command, args)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func Output(ctx context.Context, command string, args ...string) ([]byte, error) {
	fmt.Printf("Running: %s %v\n", command, args)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Output()
}
