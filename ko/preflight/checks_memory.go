package preflight

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// memoryLowGiB is where the reported total stops looking comfortable for any deployment
// shape. It is not a requirement and never blocks: what the platform actually needs depends
// on whether it runs in Kind here, on a k3s node, or directly on a host, and this command
// cannot know which numbers apply until far more is decided.
const memoryLowGiB = 7.0

const resourcesDocURL = "https://lab.1int.io/#resources-required"

// Memory reports how much memory the machine has.
//
// Detection only, deliberately. An earlier version reported "free" memory and compared it
// against a threshold, which was wrong twice over: on macOS no free figure was ever
// measured — the total was reported in its place — and a single threshold cannot describe
// requirements that vary by an order of magnitude between deployment shapes. Reporting a
// measurement that was not taken is worse than reporting nothing.
func Memory(target Target) Check {
	return Check{
		ID:    "insufficient-memory",
		Title: "Memory",
		Run: func(ctx context.Context) Result {
			return runMemory(ctx, target)
		},
	}
}

func runMemory(ctx context.Context, target Target) Result {
	total, available, err := readMemory(ctx)
	if err != nil {
		return Result{
			Status:  StatusSkip,
			Summary: "could not detect how much memory this machine has",
			Detail:  err.Error(),
		}
	}

	observed := map[string]string{
		"totalGiB": fmt.Sprintf("%.1f", total),
		"target":   string(target),
	}
	if available > 0 {
		observed["availableGiB"] = fmt.Sprintf("%.1f", available)
	}

	if !target.RunsWorkloadsHere() {
		return Result{
			Status:   StatusInfo,
			Summary:  fmt.Sprintf("%.1f GiB detected; the platform runs on the cluster, not here", total),
			Observed: observed,
		}
	}

	const caveat = "Best-effort detection. How much is actually needed depends on the " +
		"deployment type — Kind here, k3s, or directly on a host."

	status := StatusInfo
	if total < memoryLowGiB {
		status = StatusWarn
	}

	return Result{
		Status:   status,
		Summary:  fmt.Sprintf("%.1f GiB detected", total),
		Detail:   caveat,
		Link:     resourcesDocURL,
		Observed: observed,
	}
}

// readMemory returns the installed total and, where the platform reports one, the amount
// currently available. An available figure of zero means "not measured" — it is never
// substituted with the total, which is how the old macOS path came to claim a 64 GiB
// machine had 64 GiB free.
func readMemory(ctx context.Context) (totalGiB, availableGiB float64, err error) {
	if runtime.GOOS == "darwin" {
		total, err := readMemoryDarwin(ctx)
		return total, 0, err
	}
	return readMemoryProc()
}

func readMemoryProc() (totalGiB, availableGiB float64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	var availableKB, totalKB float64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, found := strings.Cut(scanner.Text(), ":")
		if !found {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			continue
		}
		kb, convErr := strconv.ParseFloat(fields[0], 64)
		if convErr != nil {
			continue
		}
		switch key {
		case "MemAvailable":
			availableKB = kb
		case "MemTotal":
			totalKB = kb
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	if totalKB == 0 {
		return 0, 0, fmt.Errorf("/proc/meminfo did not report MemTotal")
	}
	return totalKB / 1024 / 1024, availableKB / 1024 / 1024, nil
}

func readMemoryDarwin(ctx context.Context) (totalGiB float64, err error) {
	out, err := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, err
	}
	bytes, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, err
	}
	const bytesPerGiB = 1024 * 1024 * 1024
	return bytes / bytesPerGiB, nil
}
