package preflight

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Kind's own recommendation, from the page linked below. A bare Ubuntu 24.04 box ships well
// under both — 128 instances and roughly 60k watches — so this check fires on a stock
// machine rather than being a theoretical concern.
//
// A recommendation and not a requirement: how many watches a cluster consumes depends on how
// many pods end up on the node, which is not known until the deployment is chosen. Whether
// the stock limits are actually too low for a given install is unanswerable here, so this
// reports the numbers against Kind's guidance and leaves the judgement with the operator.
const (
	recommendedInotifyInstances = 512
	recommendedInotifyWatches   = 524288
)

// kindInotifyDocURL documents the symptom and the fix. Raising a sysctl persistently differs
// between distributions, so Kind's page owns the instructions rather than this command
// printing commands that are right for systemd and wrong elsewhere.
const kindInotifyDocURL = "https://kind.sigs.k8s.io/docs/user/known-issues/#pod-errors-due-to-too-many-open-files"

// InotifyLimits reports whether the kernel's inotify limits are high enough for a Kubernetes
// node's worth of pods.
//
// Only meaningful when the cluster runs on this machine: for a remote target the limits that
// matter are the cluster node's, which this command cannot see and should not guess at.
func InotifyLimits(target Target) Check {
	return Check{
		ID:    "inotify-limits",
		Title: "Kernel file watches",
		Run: func(ctx context.Context) Result {
			return runInotifyLimits(target)
		},
	}
}

func runInotifyLimits(target Target) Result {
	if !target.RunsWorkloadsHere() {
		return Result{
			Status:  StatusInfo,
			Summary: "not a limit here — the cluster does not run on this machine",
		}
	}

	if runtime.GOOS != "linux" {
		return Result{
			Status:  StatusSkip,
			Summary: fmt.Sprintf("not checked on %s", runtime.GOOS),
			Detail: "Docker runs the cluster inside its own Linux virtual machine, and that " +
				"machine's limits are set by Docker rather than by this one.",
		}
	}

	instances, errInstances := readSysctl("/proc/sys/fs/inotify/max_user_instances")
	watches, errWatches := readSysctl("/proc/sys/fs/inotify/max_user_watches")
	if errInstances != nil || errWatches != nil {
		return Result{
			Status:  StatusSkip,
			Summary: "could not read inotify limits",
		}
	}

	observed := map[string]string{
		"max_user_instances":   strconv.Itoa(instances),
		"max_user_watches":     strconv.Itoa(watches),
		"recommendedInstances": strconv.Itoa(recommendedInotifyInstances),
		"recommendedWatches":   strconv.Itoa(recommendedInotifyWatches),
	}

	// Same shape as the tooling list: the measured value, then what is recommended beside it,
	// so the two are compared by the reader rather than by a verdict in the summary.
	items := []string{
		inotifyItem("fs.inotify.max_user_instances", instances, recommendedInotifyInstances),
		inotifyItem("fs.inotify.max_user_watches", watches, recommendedInotifyWatches),
	}

	low := instances < recommendedInotifyInstances || watches < recommendedInotifyWatches
	if !low {
		return Result{
			Status:   StatusPass,
			Summary:  "at or above Kind's recommended limits",
			Observed: observed,
		}
	}

	return Result{
		Status:  StatusWarn,
		Summary: "below Kind's recommended limits",
		Items:   items,
		Detail: "Stock limits on most distributions are lower than Kind suggests for a node " +
			"running many pods. Whether it matters depends on how large the deployment ends " +
			"up being; if pods later fail to start, this is the first thing to raise.",
		Link:     kindInotifyDocURL,
		Observed: observed,
	}
}

func inotifyItem(name string, got, recommended int) string {
	return fmt.Sprintf("- %-30s %d (recommended: %d)", name, got, recommended)
}

func readSysctl(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(raw)))
}
