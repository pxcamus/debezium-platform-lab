package preflight

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// tool describes an external command the deployment shells out to.
type tool struct {
	name string
	// floor is the oldest version worth recommending, as major.minor. A floor rather than
	// an exact pin: newer is the normal case and telling someone their working tool is the
	// "wrong" version because it is newer than CI's trains them to ignore the check.
	//
	// helmfile's floor is derived from this repository — its 0.x releases take a different
	// configuration format, so 1.0 is a genuine break. helm's comes from the charts being
	// apiVersion v2, which requires Helm 3. kubectl and kind have no constraint anywhere in
	// the repo, so theirs are provisional.
	//
	// All four are placeholders pending the real answer, which is empirical: deploy against
	// progressively older versions and let the lowest one that still works become the floor.
	// Until then these say "recommended", not "required", because that is what they are.
	floor string
	// versionArgs produces something containing a version number on stdout.
	versionArgs []string
	// docs is the tool's own installation page. Each project documents its own platforms
	// and keeps that page current, which is not a responsibility this project should take
	// on: a hard-coded install command here is one more thing to go stale, and it can only
	// ever be right for the platforms someone thought of.
	docs string
	// localOnly marks tools needed only when the cluster runs on this machine.
	localOnly bool
}

var tools = []tool{
	{
		name:        "kubectl",
		floor:       "1.30",
		versionArgs: []string{"version", "--client"},
		docs:        "https://kubernetes.io/docs/tasks/tools/",
	},
	{
		name:        "helm",
		floor:       "3.0",
		versionArgs: []string{"version", "--short"},
		docs:        "https://helm.sh/docs/intro/install/",
	},
	{
		name:        "helmfile",
		floor:       "1.0",
		versionArgs: []string{"--version"},
		docs:        "https://github.com/helmfile/helmfile/releases",
	},
	{
		name:        "kind",
		floor:       "0.20",
		versionArgs: []string{"--version"},
		docs:        "https://kind.sigs.k8s.io/docs/user/quick-start/#installation",
		localOnly:   true,
	},
}

var versionPattern = regexp.MustCompile(`\d+\.\d+\.\d+`)

// ToolingVersions lists the external commands the deployment needs, where each was found,
// and warns about any older than the recommended floor.
//
// Absence is a blocker because the deployment cannot proceed without them. Being old enough
// to worry about is a warning, since upgrading is something the operator can act on. Being
// merely *different* from what CI runs is not reported at all — newer is the normal case,
// and calling a working tool wrong is how a check earns itself a reputation for crying wolf.
func ToolingVersions(target Target) Check {
	return Check{
		ID:    "tooling-versions",
		Title: "Required tools",
		Run: func(ctx context.Context) Result {
			return runToolingVersions(ctx, target)
		},
	}
}

func runToolingVersions(ctx context.Context, target Target) Result {
	var (
		required []tool
		items    []string
		missing  int
		outdated int
		observed = map[string]string{"target": string(target)}
	)

	for _, t := range tools {
		if t.localOnly && !target.RunsWorkloadsHere() {
			continue
		}
		required = append(required, t)
	}

	width := 0
	for _, t := range required {
		if len(t.name) > width {
			width = len(t.name)
		}
	}

	for _, t := range required {
		path, err := exec.LookPath(t.name)
		if err != nil {
			missing++
			observed[t.name] = "missing"
			items = append(items, fmt.Sprintf("- %-*s  missing — install from %s", width, t.name, t.docs))
			continue
		}

		version := toolVersion(ctx, path, t.versionArgs)
		observed[t.name] = version
		observed[t.name+"Path"] = path

		shown := version
		if shown == "" {
			shown = "version unknown"
		}

		// Path first, and always: with mise, asdf or several package managers on one machine
		// there are usually multiple copies installed, and which one is on PATH is the
		// question behind a version that looks wrong.
		line := fmt.Sprintf("- %s %s", shortenPath(path), shown)
		if t.floor != "" {
			line += fmt.Sprintf(" (recommended: %s+)", t.floor)
			if version != "" && olderThan(version, t.floor) {
				outdated++
			}
		}
		items = append(items, line)
	}

	switch {
	case missing > 0:
		return Result{
			Status:   StatusFail,
			Summary:  fmt.Sprintf("%d of %d missing", missing, len(required)),
			Items:    items,
			Observed: observed,
		}

	case outdated > 0:
		return Result{
			Status:  StatusWarn,
			Summary: fmt.Sprintf("all %d installed, %d older than recommended", len(required), outdated),
			Items:   items,
			Detail: "Out-of-date tooling can break the installation in ways that are hard to " +
				"trace back. Consider upgrading each to its latest version.",
			Observed: observed,
		}

	default:
		return Result{
			Status:   StatusPass,
			Summary:  fmt.Sprintf("all %d installed", len(required)),
			Items:    items,
			Observed: observed,
		}
	}
}

// olderThan compares dotted numeric versions component by component. Enough for the
// major.minor floors above, and deliberately not a semver implementation: pre-release and
// build metadata carry no meaning for "is this tool old enough to worry about".
func olderThan(version, floor string) bool {
	got := versionParts(version)
	want := versionParts(floor)

	for i := range want {
		// An absent component is zero, not missing: "1" means 1.0, and is not older than a
		// floor of "1.0".
		component := 0
		if i < len(got) {
			component = got[i]
		}
		if component != want[i] {
			return component < want[i]
		}
	}
	return false
}

func versionParts(v string) []int {
	fields := strings.Split(v, ".")
	parts := make([]int, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil {
			break
		}
		parts = append(parts, n)
	}
	return parts
}

// shortenPath abbreviates the home directory. Shorter lines, and one less reason for a
// pasted report to carry someone's username.
func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if after, found := strings.CutPrefix(path, home+string(os.PathSeparator)); found {
		return "~" + string(os.PathSeparator) + after
	}
	return path
}

func toolVersion(ctx context.Context, path string, args []string) string {
	out, err := exec.CommandContext(ctx, path, args...).CombinedOutput()
	if err != nil && len(out) == 0 {
		return ""
	}
	return versionPattern.FindString(string(out))
}
