// Package preflight inspects a machine for its ability to run or deploy the Debezium
// Platform. Checks return structured results and never write to the terminal, so the same
// catalog can back a plain CLI renderer, a JSON contract for CI, and a TUI.
package preflight

import (
	"context"
	"encoding/json"
	"fmt"
)

// DocBaseURL is the published troubleshooting page. Every check ID is an explicit heading
// anchor there, and the documentation build fails under --strict if one stops resolving,
// so a renamed check cannot silently orphan its link.
const DocBaseURL = "https://lab.1int.io/troubleshooting/"

// Status is the outcome of a single check.
type Status int

const (
	// StatusPass means the machine satisfies the check.
	StatusPass Status = iota
	// StatusInfo reports a resolved value worth confirming rather than a problem.
	StatusInfo
	// StatusWarn means the deployment will probably work but something looks wrong.
	StatusWarn
	// StatusFail means the deployment is expected to break or silently do nothing.
	StatusFail
	// StatusSkip means the check could not run and reached no conclusion. It is never a
	// substitute for StatusPass: a check that could not look is not a check that passed.
	StatusSkip
)

// String returns the lowercase name used in rendered output and in the JSON contract.
func (s Status) String() string {
	switch s {
	case StatusPass:
		return "pass"
	case StatusInfo:
		return "info"
	case StatusWarn:
		return "warn"
	case StatusFail:
		return "fail"
	case StatusSkip:
		return "skip"
	default:
		return "unknown"
	}
}

// MarshalJSON encodes the status as its name so the JSON contract stays readable and
// stable if the underlying constants are ever reordered.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Result is what a check observed. Checks build one and return it; they never print.
type Result struct {
	Status Status
	// Summary is a single line stating what was found, not what should be done.
	Summary string
	// Detail is optional context, such as the exact error a probed command produced.
	Detail string
	// Remedy lists commands the operator can run verbatim. Preflight never runs them
	// itself: a check that silently repairs the machine hides the fact that it was wrong,
	// and the gap resurfaces on someone else's machine instead.
	Remedy []string
	// Observed carries the raw values behind the summary, for the JSON contract and for
	// bug reports where the conclusion is less useful than the measurement.
	Observed map[string]string
}

// Check is one machine inspection. ID must match a heading anchor on the troubleshooting
// page; see DocBaseURL.
type Check struct {
	ID    string
	Title string
	Run   func(context.Context) Result
}

// Finding pairs a check with its result and is the unit every renderer consumes.
type Finding struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Status   Status            `json:"status"`
	Summary  string            `json:"summary"`
	Detail   string            `json:"detail,omitempty"`
	Remedy   []string          `json:"remedy,omitempty"`
	Observed map[string]string `json:"observed,omitempty"`
	DocURL   string            `json:"docUrl"`
}

// Report is the complete outcome of a run.
type Report struct {
	Findings []Finding `json:"findings"`
}

// Counts tallies findings by status, keyed by Status.String.
func (r Report) Counts() map[string]int {
	counts := make(map[string]int, 5)
	for _, f := range r.Findings {
		counts[f.Status.String()]++
	}
	return counts
}

// OK reports whether the run found nothing fatal. Warnings do not make it false: they are
// judgement calls for the operator, and a preflight that blocks on them becomes a preflight
// people learn to bypass.
func (r Report) OK() bool {
	for _, f := range r.Findings {
		if f.Status == StatusFail {
			return false
		}
	}
	return true
}

// Run executes every check in order and collects the findings. Checks are expected to be
// cheap and side-effect free; ctx carries the deadline for those that touch the network.
func Run(ctx context.Context, checks []Check) Report {
	findings := make([]Finding, 0, len(checks))
	for _, c := range checks {
		res := c.Run(ctx)
		findings = append(findings, Finding{
			ID:       c.ID,
			Title:    c.Title,
			Status:   res.Status,
			Summary:  res.Summary,
			Detail:   res.Detail,
			Remedy:   res.Remedy,
			Observed: res.Observed,
			DocURL:   fmt.Sprintf("%s#%s", DocBaseURL, c.ID),
		})
	}
	return Report{Findings: findings}
}
