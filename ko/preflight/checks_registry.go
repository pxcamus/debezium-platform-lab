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
func ImagePull(target Target) Check {
	return Check{
		ID:    "image-pull",
		Title: "Image downloads",
		Run: func(ctx context.Context) Result {
			return runImagePull(ctx, target)
		},
	}
}

func runImagePull(ctx context.Context, target Target) Result {
	observed := map[string]string{
		"arch":   runtime.GOARCH,
		"os":     runtime.GOOS,
		"target": string(target),
	}

	var unreachable []string
	for host, outcome := range probeRegistries(ctx) {
		observed[host] = outcome.describe()
		if !outcome.reachable {
			unreachable = append(unreachable, host)
		}
	}
	sort.Strings(unreachable)

	// Reported together with reachability rather than short-circuiting ahead of it: an
	// operator on an unsupported architecture still needs to know whether their network can
	// reach the registries, because the remote path depends on it and the local one does not.
	archOK := runtime.GOARCH == "amd64"

	switch {
	case !archOK && target.RunsWorkloadsHere():
		return Result{
			Status:  StatusFail,
			Summary: fmt.Sprintf("this machine's processor (%s) cannot run the platform's images%s", runtime.GOARCH, registrySuffix(unreachable)),
			Detail: "The platform is published for Intel and AMD processors only. On this " +
				"machine the download fails with an error about missing manifests, which never " +
				"mentions the processor — so it is worth knowing before you start.",
			Remedy: []string{
				fmt.Sprintf("Deploy to a cluster instead of this machine: dmp-lab doctor --target %s", TargetRemote),
				"Or switch the platform to its nightly build, which supports this processor.",
			},
			Observed: observed,
		}

	case !archOK:
		return Result{
			Status:   StatusInfo,
			Summary:  fmt.Sprintf("this machine's processor (%s) does not matter here%s", runtime.GOARCH, registrySuffix(unreachable)),
			Detail:   "The images run on the cluster's machines, not on this one.",
			Observed: observed,
		}

	case len(unreachable) > 0:
		return Result{
			Status: StatusWarn,
			Summary: fmt.Sprintf("cannot reach %d of the %d servers the platform downloads from: %s",
				len(unreachable), len(platformRegistries), strings.Join(unreachable, ", ")),
			Detail: "The deployment will stop part-way through, waiting for downloads that " +
				"never arrive. A company proxy or firewall is the usual cause.",
			Remedy: []string{
				"Check that this machine can reach those addresses over HTTPS, including any proxy settings.",
			},
			Observed: observed,
		}

	default:
		return Result{
			Status:   StatusPass,
			Summary:  "all downloads reachable, and this machine can run the images",
			Observed: observed,
		}
	}
}

func registrySuffix(unreachable []string) string {
	if len(unreachable) == 0 {
		return "; downloads are reachable"
	}
	return fmt.Sprintf("; also cannot reach %s", strings.Join(unreachable, ", "))
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
