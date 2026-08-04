package preflight

import (
	"fmt"
	"strings"

	"dbz-mage/ko/automation"
)

// Target is where the platform is going to be deployed. Several checks are only meaningful
// for one of them: an arm64 laptop cannot run the release images itself, but it can drive a
// deployment onto an amd64 cluster perfectly well, and reporting that as a failure turns a
// working setup into a dead end.
type Target string

const (
	// TargetLocal deploys a Kind cluster onto this machine. The workloads run here, so this
	// machine's architecture, memory, kernel limits and free ports all matter.
	TargetLocal Target = "local"
	// TargetRemote deploys onto a cluster this machine only talks to. The workloads run
	// elsewhere, so host capacity is informational and kubeconfig resolution is critical.
	TargetRemote Target = "remote"
)

// ParseTarget converts a flag value into a Target.
func ParseTarget(s string) (Target, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(TargetLocal), "kind":
		return TargetLocal, nil
	case string(TargetRemote), "k3s":
		return TargetRemote, nil
	default:
		return "", fmt.Errorf("unknown target %q: expected %q or %q", s, TargetLocal, TargetRemote)
	}
}

// DetectTarget infers the target from the environment, mirroring how CLUSTER_TYPE already
// selects a cluster provider elsewhere in this project so the two cannot disagree.
func DetectTarget() Target {
	if strings.EqualFold(automation.Env("CLUSTER_TYPE", "kind"), "k3s") {
		return TargetRemote
	}
	return TargetLocal
}

// RunsWorkloadsHere reports whether the platform's containers will execute on this machine.
// Checks about host capacity should consult this rather than testing the target directly,
// so adding a third target does not mean revisiting every check.
func (t Target) RunsWorkloadsHere() bool {
	return t == TargetLocal
}
