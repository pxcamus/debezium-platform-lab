package preflight

import (
	"context"
	"os/exec"
	"strings"
)

const dockerInstallURL = "https://docs.docker.com/get-docker/"

// ContainerRuntime reports whether a container runtime is installed, running, and usable by
// the current user.
//
// The probe is "docker info", not "docker --version": the version subcommand is served by
// the client alone and succeeds against a stopped daemon and against a socket the user
// cannot open, which are two of the three ways this check fails in practice.
func ContainerRuntime() Check {
	return Check{
		ID:    "container-runtime",
		Title: "Container runtime",
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
			Summary:  "Docker is running",
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
		res.Summary = "Docker is installed, but your user is not allowed to use it"
		res.Detail = "Adding yourself to the docker group needs a new login session to take " +
			"effect — this is why it can look like the change did nothing."
		res.Remedy = []string{
			"sudo usermod -aG docker $USER",
			"newgrp docker",
		}
	case strings.Contains(detail, "Cannot connect to the Docker daemon"),
		strings.Contains(detail, "Is the docker daemon running"):
		res.Summary = "Docker is installed, but it is not running"
		res.Detail = ""
		res.Remedy = []string{"sudo systemctl enable --now docker"}
	default:
		res.Summary = "Docker is installed, but it is not responding"
	}

	return res
}

func dockerMissing(ctx context.Context) Result {
	observed := map[string]string{"runtime": "none"}

	// Docker's own instructions rather than a package-manager command: this runs on
	// distributions with different package managers, on macOS, and behind corporate
	// installers, and guessing wrong sends someone down a path that cannot work. Docker
	// documents every one of those cases and keeps the page current.
	remedy := []string{"Install Docker: " + dockerInstallURL}

	if podman, err := exec.LookPath("podman"); err == nil {
		observed["podman"] = podman
		return Result{
			Status:  StatusWarn,
			Summary: "Docker is not installed; Podman is, but it is not what this project is tested with",
			Detail: "The deployment may work on Podman, but nothing here has been verified " +
				"against it, so any problem you hit will be undocumented.",
			Remedy:   remedy,
			Observed: observed,
		}
	}

	return Result{
		Status:  StatusFail,
		Summary: "Docker is not installed",
		Detail: "Docker runs the platform's containers. Nothing can be deployed without a " +
			"container runtime.",
		Remedy:   remedy,
		Observed: observed,
	}
}
