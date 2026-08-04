package preflight

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

var ingressPorts = []int{80, 443}

const portProbeTimeout = 2 * time.Second

// IngressPorts reports whether the host ports the ingress needs are already taken.
//
// The probe connects rather than binds. Binding 80 or 443 requires privilege, so a
// bind-based test would report "busy" for every unprivileged user on an otherwise free
// port — a false positive on exactly the machines this runs on.
func IngressPorts(target Target) Check {
	return Check{
		ID:    "ingress-ports-busy",
		Title: "Network ports",
		Run: func(ctx context.Context) Result {
			return runIngressPorts(ctx, target)
		},
	}
}

func runIngressPorts(ctx context.Context, target Target) Result {
	if !target.RunsWorkloadsHere() {
		return Result{
			Status:  StatusInfo,
			Summary: "not a limit here — the platform is not served from this machine",
		}
	}

	observed := map[string]string{}
	var busy []string

	for _, port := range ingressPorts {
		if portInUse(ctx, port) {
			busy = append(busy, strconv.Itoa(port))
			observed[strconv.Itoa(port)] = "in use"
			continue
		}
		observed[strconv.Itoa(port)] = "free"
	}

	if len(busy) > 0 {
		return Result{
			Status:  StatusFail,
			Summary: "something else is already using port " + strings.Join(busy, " and "),
			Detail: "The platform is served on these ports. If another program holds them, " +
				"everything will deploy successfully and the web address will not open.",
			Remedy: []string{
				fmt.Sprintf("sudo lsof -iTCP:%s -sTCP:LISTEN", strings.Join(busy, ",")),
				"Stop whatever that command shows, then run this again.",
			},
			Observed: observed,
		}
	}

	return Result{
		Status:   StatusPass,
		Summary:  "ports 80 and 443 are free",
		Observed: observed,
	}
}

func portInUse(ctx context.Context, port int) bool {
	dialer := net.Dialer{Timeout: portProbeTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
