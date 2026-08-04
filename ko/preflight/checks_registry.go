package preflight

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var platformRegistries = []string{
	"registry-1.docker.io",
	"quay.io",
	"ghcr.io",
	"registry.k8s.io",
}

const registryProbeTimeout = 10 * time.Second

// ImagePull reports whether the platform's image registries are reachable and whether this
// machine's CPU architecture is one the published images support.
//
// Reachability treats 401 and 403 as success. A registry requiring authentication answers
// an unauthenticated /v2/ request with 401, and three of the four registries here do
// exactly that — only registry.k8s.io allows anonymous access and returns 200. Testing for
// 200 alone reports three healthy registries as unreachable.
func ImagePull() Check {
	return Check{
		ID:    "image-pull",
		Title: "Registries reachable and architecture supported",
		Run:   runImagePull,
	}
}

func runImagePull(ctx context.Context) Result {
	// Checked here because an unsupported architecture surfaces as a pull failure, but it
	// is worth its own identifier: every other finding on this page is remediable, and this
	// one can only be answered by using a different machine.
	if runtime.GOARCH != "amd64" {
		return Result{
			Status:  StatusFail,
			Summary: fmt.Sprintf("architecture %s is not supported by the platform images", runtime.GOARCH),
			Detail: "The Debezium Platform release images are published for amd64 only; just the " +
				"nightly tag is multi-arch. Pulls fail with manifest errors that do not mention " +
				"architecture.",
			Remedy:   []string{"Deploy from an amd64 machine, or pin the platform to the nightly tag."},
			Observed: map[string]string{"arch": runtime.GOARCH, "os": runtime.GOOS},
		}
	}

	results := probeRegistries(ctx)

	observed := map[string]string{"arch": runtime.GOARCH}
	var unreachable []string
	for host, outcome := range results {
		observed[host] = outcome.describe()
		if !outcome.reachable {
			unreachable = append(unreachable, host)
		}
	}
	sort.Strings(unreachable)

	if len(unreachable) > 0 {
		return Result{
			Status:  StatusWarn,
			Summary: fmt.Sprintf("%d of %d registries unreachable: %s", len(unreachable), len(platformRegistries), strings.Join(unreachable, ", ")),
			Detail: "Image pulls will fail during deployment. A proxy or egress filter is the " +
				"usual cause on a corporate network.",
			Remedy: []string{
				"Check HTTPS egress to the listed hosts, including any proxy configuration.",
			},
			Observed: observed,
		}
	}

	return Result{
		Status:   StatusPass,
		Summary:  fmt.Sprintf("all %d registries reachable, architecture amd64", len(platformRegistries)),
		Observed: observed,
	}
}

type probeOutcome struct {
	reachable bool
	status    int
	err       error
}

func (o probeOutcome) describe() string {
	if o.err != nil {
		return "unreachable: " + o.err.Error()
	}
	return fmt.Sprintf("HTTP %d", o.status)
}

func probeRegistries(ctx context.Context) map[string]probeOutcome {
	client := &http.Client{Timeout: registryProbeTimeout}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make(map[string]probeOutcome, len(platformRegistries))
	)

	for _, host := range platformRegistries {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			outcome := probeRegistry(ctx, client, host)
			mu.Lock()
			results[host] = outcome
			mu.Unlock()
		}(host)
	}
	wg.Wait()

	return results
}

func probeRegistry(ctx context.Context, client *http.Client, host string) probeOutcome {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+"/v2/", nil)
	if err != nil {
		return probeOutcome{err: err}
	}

	resp, err := client.Do(req)
	if err != nil {
		return probeOutcome{err: err}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusUnauthorized, http.StatusForbidden:
		return probeOutcome{reachable: true, status: resp.StatusCode}
	default:
		return probeOutcome{status: resp.StatusCode}
	}
}
