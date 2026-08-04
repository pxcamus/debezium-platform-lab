package preflight

import (
	"context"
	"fmt"
	"net"
	"strings"

	"dbz-mage/ko/automation"
)

// sslip.io resolves any hostname with an embedded address to that address, which is what
// lets ingress hostnames work without editing /etc/hosts on every machine. The loopback
// probe is deliberate: resolvers with DNS-rebinding protection refuse answers pointing at
// loopback or private ranges, and that refusal is the failure this check exists to catch.
const (
	sslipProbeHost = "127-0-0-1.sslip.io"
	sslipProbeWant = "127.0.0.1"
)

// DNSResolution reports whether the DNS mechanism the ingress hostnames depend on works
// from this machine.
func DNSResolution(target Target) Check {
	return Check{
		ID:    "dns-resolution",
		Title: "Web address",
		Run: func(ctx context.Context) Result {
			return runDNSResolution(ctx, target)
		},
	}
}

func runDNSResolution(ctx context.Context, target Target) Result {
	domain := automation.Env("DBZ_DOMAIN", "")
	observed := map[string]string{
		"target":     string(target),
		"DBZ_DOMAIN": domain,
	}

	if domain != "" {
		addrs, err := net.DefaultResolver.LookupHost(ctx, "dmp."+domain)
		if err != nil {
			observed["dmp."+domain] = "did not resolve"
			return Result{
				Status:  StatusWarn,
				Summary: fmt.Sprintf("dmp.%s does not lead anywhere yet", domain),
				Detail: "Everything would deploy successfully and you would not be able to open " +
					"the platform in a browser.",
				Remedy:   dnsRemedy(target),
				Observed: observed,
			}
		}
		observed["dmp."+domain] = strings.Join(addrs, ", ")

		// Resolving is not the same as resolving somewhere useful. A domain left over from
		// another environment resolves perfectly well and sends every ingress hostname to a
		// machine that is not the one being deployed to, so checking only that the name works
		// puts a green tick on a broken configuration.
		pointsHere := anyLocal(addrs)

		if target.RunsWorkloadsHere() && !pointsHere {
			return Result{
				Status: StatusWarn,
				Summary: fmt.Sprintf("dmp.%s leads to %s, which is not this machine",
					domain, strings.Join(addrs, ", ")),
				Detail: "You are deploying here, so the address has to lead here. This one looks " +
					"left over from a different environment: the deployment would succeed and " +
					"the address would take you somewhere else.",
				Remedy: []string{
					"Unset DBZ_DOMAIN to have a working address generated for you.",
					"Or set it to a name that leads to this machine.",
				},
				Observed: observed,
			}
		}

		if !target.RunsWorkloadsHere() && pointsHere {
			return Result{
				Status: StatusWarn,
				Summary: fmt.Sprintf("dmp.%s leads back to this machine, not to the cluster",
					domain),
				Detail: "You are deploying to a separate cluster, so the address has to lead " +
					"there rather than here.",
				Remedy:   []string{"Set DBZ_DOMAIN to a name that leads to the cluster."},
				Observed: observed,
			}
		}

		return Result{
			Status:   StatusPass,
			Summary:  fmt.Sprintf("dmp.%s leads to %s", domain, strings.Join(addrs, ", ")),
			Observed: observed,
		}
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, sslipProbeHost)
	switch {
	case err != nil:
		observed[sslipProbeHost] = "did not resolve"
		return Result{
			Status:  StatusWarn,
			Summary: "no web address set, and the automatic option does not work on this network",
			Detail: "The platform is normally reached at a name provided by sslip.io. Many " +
				"company networks block those names as a security measure, so you will need to " +
				"set an address by hand.",
			Remedy:   dnsRemedy(target),
			Observed: observed,
		}

	case !contains(addrs, sslipProbeWant):
		observed[sslipProbeHost] = strings.Join(addrs, ", ")
		return Result{
			Status:  StatusWarn,
			Summary: "no web address set, and the automatic option is being redirected on this network",
			Detail: fmt.Sprintf("A test name that should lead to %s led to %s instead — "+
				"something on this network is rewriting the answer.",
				sslipProbeWant, strings.Join(addrs, ", ")),
			Remedy:   dnsRemedy(target),
			Observed: observed,
		}

	default:
		observed[sslipProbeHost] = strings.Join(addrs, ", ")
		return Result{
			Status:  StatusInfo,
			Summary: "no web address set yet — one can be generated for you",
			Detail: "Set DBZ_DOMAIN and the platform gets a working address with no further " +
				"setup on your machine.",
			Observed: observed,
		}
	}
}

func dnsRemedy(target Target) []string {
	if target.RunsWorkloadsHere() {
		return []string{
			"Add the platform's hostnames to /etc/hosts pointing at 127.0.0.1.",
		}
	}
	return []string{
		"Set DBZ_DOMAIN to a name that leads to the cluster.",
	}
}

// anyLocal reports whether any resolved address belongs to this machine — loopback, or an
// address configured on one of its interfaces. Both count: Kind publishes its ingress on
// loopback, but an operator may point a name at the machine's LAN address instead.
func anyLocal(addrs []string) bool {
	local := localAddresses()
	for _, a := range addrs {
		ip := net.ParseIP(a)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || local[ip.String()] {
			return true
		}
	}
	return false
}

func localAddresses() map[string]bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	local := make(map[string]bool, len(addrs))
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			local[ipNet.IP.String()] = true
		}
	}
	return local
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
