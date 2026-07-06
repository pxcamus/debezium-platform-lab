package cluster

import (
	"context"
	"dbz-mage/ko/runner"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dbz-mage/ko/automation"
)

type K3s struct {
	Runner           runner.Runner
	Host             string
	ContextName      string
	RemoteKubeconfig string
	LocalKubeconfig  string
}

func NewK3sFromEnv() (*K3s, error) {
	host, err := automation.RequiredEnv("K3S_HOST")
	if err != nil {
		return nil, err
	}

	keyPath, err := automation.RequiredEnv("K3S_SSH_KEY")
	if err != nil {
		return nil, err
	}

	sshTarget := runner.SSH{
		User:    automation.Env("K3S_SSH_USER", "ec2-user"),
		Host:    host,
		KeyPath: keyPath,
	}

	return &K3s{
		Runner:           sshTarget,
		Host:             host,
		ContextName:      automation.Env("K3S_CONTEXT", "k3s-aws"),
		RemoteKubeconfig: automation.Env("K3S_REMOTE_KUBECONFIG", "/tmp/k3s.yaml"),
		LocalKubeconfig:  automation.ExpandPath(automation.Env("K3S_LOCAL_KUBECONFIG", "~/.kube/k3s-aws.yaml")),
	}, nil
}

func (k *K3s) Recreate(ctx context.Context) error {
	if err := k.Delete(ctx); err != nil {
		return err
	}

	if err := k.Install(ctx); err != nil {
		return err
	}

	if err := k.Wait(ctx); err != nil {
		return err
	}

	return k.FetchKubeconfig(ctx)
}

func (k *K3s) Delete(ctx context.Context) error {
	const uninstallScript = "/usr/local/bin/k3s-uninstall.sh"

	exists, err := k.remoteFileExists(ctx, uninstallScript)
	if err != nil {
		return err
	}

	if !exists {
		fmt.Println("k3s is not installed")
		return nil
	}

	return k.Runner.Run(ctx, fmt.Sprintf("sudo %s", uninstallScript))
}

func (k *K3s) Install(ctx context.Context) error {
	//return k.Runner.Run(
	//	ctx,
	//	`curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --tls-san $(curl -s ifconfig.me)" sh -`,
	//)
	return k.Runner.Run(
		ctx,
		`curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --tls-san %s" sh -`, k.Host,
	)
}

func (k *K3s) Wait(ctx context.Context) error {
	return k.Runner.Run(ctx, "sudo k3s kubectl wait --for=condition=ready node --all --timeout=120s && sudo k3s kubectl get pods -A")
}

func (k *K3s) FetchKubeconfig(ctx context.Context) error {
	prepareCommand := fmt.Sprintf(
		"sudo cp /etc/rancher/k3s/k3s.yaml %[1]s && sudo chmod 644 %[1]s && sudo chown $(id -u):$(id -g) %[1]s",
		k.RemoteKubeconfig,
	)

	if err := k.Runner.Run(ctx, prepareCommand); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(k.LocalKubeconfig), 0700); err != nil {
		return err
	}

	if err := k.Runner.CopyFrom(ctx, k.RemoteKubeconfig, k.LocalKubeconfig); err != nil {
		return err
	}

	data, err := os.ReadFile(k.LocalKubeconfig)
	if err != nil {
		return err
	}

	content := string(data)
	content = strings.ReplaceAll(content, "https://127.0.0.1:6443", fmt.Sprintf("https://%s:6443", k.Host))
	content = strings.ReplaceAll(content, "https://localhost:6443", fmt.Sprintf("https://%s:6443", k.Host))
	content = strings.ReplaceAll(content, "default", k.ContextName)

	return os.WriteFile(k.LocalKubeconfig, []byte(content), 0600)
}

func (k *K3s) remoteFileExists(ctx context.Context, path string) (bool, error) {
	err := k.Runner.Run(ctx, fmt.Sprintf("test -f %s", path))
	if err == nil {
		return true, nil
	}

	if exitCode(err) == 1 {
		return false, nil
	}

	return false, err
}

func (k *K3s) UseContext(ctx context.Context) error {
	return runner.Local{}.Run(ctx, "kubectl", "config", "use-context", k.ContextName)
}

func exitCode(err error) int {
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode()
	}

	return -1
}
