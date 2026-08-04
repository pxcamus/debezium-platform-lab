package preflight

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
)

// ContainerRuntime reports whether a container runtime is installed, running, and usable by
// the current user.
//
// The probe is "docker info", not "docker --version": the version subcommand is served by
// the client alone and succeeds against a stopped daemon and against a socket the user
// cannot open, which are two of the three ways this check fails in practice.
func ContainerRuntime() Check {
	return Check{
		ID:    "container-runtime",
		Title: "Container runtime reachable",
		Run:   runContainerRuntime,
	}
}

func runContainerRuntime(ctx context.Context) Result {
	path, err := exec.LookPath("docker")
	if err != nil {
		return dockerMissing(ctx)
	}

	out, err := exec.CommandContext(ctx, path, "info", "--format", "{{.ServerVersion}}").CombinedOutput()
	if err == nil {
		version := strings.TrimSpace(string(out))
		return Result{
			Status:   StatusPass,
			Summary:  "docker daemon reachable, server " + version,
			Observed: map[string]string{"runtime": "docker", "serverVersion": version, "path": path},
		}
	}

	detail := strings.TrimSpace(string(out))
	res := Result{
		Status:   StatusFail,
		Detail:   detail,
		Observed: map[string]string{"runtime": "docker", "path": path},
	}

	switch {
	case strings.Contains(detail, "permission denied"):
		res.Summary = "docker is installed but this user cannot open its socket"
		res.Remedy = []string{
			"sudo usermod -aG docker $USER",
			"newgrp docker   # or log out and back in — group changes need a new session",
		}
	case strings.Contains(detail, "Cannot connect to the Docker daemon"),
		strings.Contains(detail, "Is the docker daemon running"):
		res.Summary = "docker is installed but the daemon is not running"
		res.Remedy = []string{"sudo systemctl enable --now docker"}
	default:
		res.Summary = "docker is installed but 'docker info' failed"
	}

	return res
}

func dockerMissing(ctx context.Context) Result {
	observed := map[string]string{"runtime": "none"}

	remedy := []string{"sudo apt-get install -y docker.io"}
	if runtime.GOOS == "darwin" {
		remedy = []string{"brew install --cask docker   # then start Docker Desktop"}
	}

	if podman, err := exec.LookPath("podman"); err == nil {
		observed["podman"] = podman
		return Result{
			Status:  StatusWarn,
			Summary: "docker not found; podman is installed but is not the tested runtime",
			Detail: "kind can run on podman, but this project is only exercised against docker. " +
				"Expect to hit differences that are not documented here.",
			Remedy:   remedy,
			Observed: observed,
		}
	}

	return Result{
		Status:   StatusFail,
		Summary:  "no container runtime found",
		Remedy:   remedy,
		Observed: observed,
	}
}
