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

// memoryFloorGiB is the amount of memory below which the deployment is expected to fail.
//
// Expressed in GiB, and compared against *available* rather than total, because both
// details are load-bearing. A Hetzner cpx32 is sold as "8 GB" and reports 7.6 GiB total,
// 7.2 GiB available: a naive "at least 8 GB" test fails the machine this project uses as
// its own reference. Total is also the wrong quantity — it says nothing about what the
// deployment can actually have.
//
// The value is provisional. It is set below the reference machine on the evidence that the
// reference machine works, not on a measurement of what the stack peaks at; one instrumented
// deployment should replace it with a real number.
const memoryFloorGiB = 7.0

const bytesPerGiB = 1024 * 1024 * 1024

// Memory reports whether the machine has enough usable RAM for the platform.
func Memory() Check {
	return Check{
		ID:    "insufficient-memory",
		Title: "Sufficient memory",
		Run:   runMemory,
	}
}

func runMemory(ctx context.Context) Result {
	available, total, err := readMemory(ctx)
	if err != nil {
		return Result{
			Status:  StatusSkip,
			Summary: "could not determine available memory",
			Detail:  err.Error(),
		}
	}

	observed := map[string]string{
		"availableGiB": fmt.Sprintf("%.1f", available),
		"totalGiB":     fmt.Sprintf("%.1f", total),
		"floorGiB":     fmt.Sprintf("%.1f", memoryFloorGiB),
	}

	if available < memoryFloorGiB {
		return Result{
			Status: StatusFail,
			Summary: fmt.Sprintf("%.1f GiB available, need at least %.1f GiB",
				available, memoryFloorGiB),
			Detail: "Below this the Kafka broker is typically killed part-way through startup, " +
				"which surfaces as an unrelated-looking timeout elsewhere in the stack.",
			Remedy: []string{
				"Close other workloads, or move to a machine with more memory.",
			},
			Observed: observed,
		}
	}

	return Result{
		Status:   StatusPass,
		Summary:  fmt.Sprintf("%.1f GiB available of %.1f GiB total", available, total),
		Observed: observed,
	}
}

func readMemory(ctx context.Context) (availableGiB, totalGiB float64, err error) {
	if runtime.GOOS == "darwin" {
		return readMemoryDarwin(ctx)
	}
	return readMemoryProc()
}

func readMemoryProc() (availableGiB, totalGiB float64, err error) {
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
	return availableKB / 1024 / 1024, totalKB / 1024 / 1024, nil
}

func readMemoryDarwin(ctx context.Context) (availableGiB, totalGiB float64, err error) {
	out, err := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, 0, err
	}
	bytes, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, 0, err
	}
	total := bytes / bytesPerGiB
	return total, total, nil
}
