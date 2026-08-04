package preflight

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestProbeRegistryTreats401AsReachable guards the calibration finding from the first
// bare-machine baseline: three of the four registries the platform pulls from answer an
// unauthenticated /v2/ request with 401, which is the correct healthy response. A probe
// that accepts only 200 reports them all as unreachable on a working machine.
func TestProbeRegistryTreats401AsReachable(t *testing.T) {
	cases := []struct {
		name      string
		code      int
		reachable bool
	}{
		{"anonymous pull allowed", http.StatusOK, true},
		{"auth required", http.StatusUnauthorized, true},
		{"auth rejected", http.StatusForbidden, true},
		{"registry broken", http.StatusInternalServerError, false},
		{"not a registry", http.StatusNotFound, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
			}))
			defer srv.Close()

			host := mustHost(t, srv.URL)

			got := probeRegistry(context.Background(), srv.Client(), host)

			if got.reachable != tc.reachable {
				t.Errorf("HTTP %d: reachable = %v, want %v", tc.code, got.reachable, tc.reachable)
			}
			if got.status != tc.code {
				t.Errorf("status = %d, want %d", got.status, tc.code)
			}
		})
	}
}

func TestProbeRegistryUnreachableHostIsNotReachable(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	host := mustHost(t, srv.URL)
	srv.Close()

	got := probeRegistry(context.Background(), srv.Client(), host)

	if got.reachable {
		t.Error("closed server reported as reachable")
	}
	if got.err == nil {
		t.Error("closed server produced no error")
	}
}

// mustHost strips the scheme from an httptest URL. probeRegistry builds an https:// URL
// from a bare host, and the test server's own client is configured to reach it.
func mustHost(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parsing test server URL: %v", err)
	}
	return u.Host
}
