package preflight

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"dbz-mage/ko/automation"
)

// kubeconfig is the subset of a kubeconfig file this check reads. Parsed from disk; no
// request is ever made to the API server, so a dead cluster costs nothing and cannot hang.
type kubeconfig struct {
	CurrentContext string `yaml:"current-context"`
	Contexts       []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	Clusters []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			Server string `yaml:"server"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
}

// ClusterMismatch resolves and reports the cluster a deployment would actually target.
//
// The resolution mirrors magefile.go's commandEnv exactly, including its default of
// ~/.kube/k3s-aws.yaml when CLUSTER_TYPE=k3s and K3S_LOCAL_KUBECONFIG is unset. Mirroring
// matters more than being clever here: a check that resolves differently from the code it
// describes would reassure the operator about the wrong file.
func ClusterMismatch(target Target) Check {
	return Check{
		ID:    "cluster-mismatch",
		Title: "Deployment target",
		Run: func(context.Context) Result {
			return runClusterMismatch(target)
		},
	}
}

func runClusterMismatch(target Target) Result {
	path, source := resolveKubeconfig()
	observed := map[string]string{
		"target":     string(target),
		"kubeconfig": path,
		"source":     source,
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return Result{
				Status:   StatusSkip,
				Summary:  fmt.Sprintf("could not read %s", path),
				Detail:   err.Error(),
				Observed: observed,
			}
		}
		if target.RunsWorkloadsHere() {
			return Result{
				Status:   StatusInfo,
				Summary:  "a new cluster will be created on this machine",
				Observed: observed,
			}
		}
		return Result{
			Status:  StatusFail,
			Summary: "no connection settings found, so there is nowhere to deploy",
			Detail: "Expected them at " + path + ". This is the usual result of a cluster being " +
				"rebuilt without its connection file being fetched again.",
			Remedy:   []string{"Fetch the cluster's kubeconfig again, or point K3S_LOCAL_KUBECONFIG at it."},
			Observed: observed,
		}
	}

	var cfg kubeconfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Result{
			Status:   StatusSkip,
			Summary:  fmt.Sprintf("could not parse %s", path),
			Detail:   err.Error(),
			Observed: observed,
		}
	}

	server := cfg.serverFor(cfg.CurrentContext)
	observed["currentContext"] = cfg.CurrentContext
	observed["server"] = server

	isKindContext := strings.HasPrefix(cfg.CurrentContext, "kind-")
	observed["isKindContext"] = fmt.Sprintf("%t", isKindContext)

	// On the local path this is informational, not a warning. Doctor runs before the Kind
	// cluster exists, so pointing out that the current connection is not Kind describes a
	// state that deploying immediately resolves — and the advice it used to print
	// (use-context kind-<name>) named a connection that does not exist yet, so it could not
	// be followed. What the operator actually wants to know is whether their existing setup
	// is about to be disturbed.
	if target.RunsWorkloadsHere() {
		summary := "a new cluster will be created on this machine"
		if cfg.CurrentContext != "" {
			summary += fmt.Sprintf("; your current connection (%q) stays untouched", cfg.CurrentContext)
		}
		return Result{
			Status:   StatusInfo,
			Summary:  summary,
			Observed: observed,
		}
	}

	if cfg.CurrentContext == "" {
		return Result{
			Status:   StatusFail,
			Summary:  "no cluster is selected, so there is nowhere to deploy",
			Detail:   "The connection settings in " + path + " do not say which cluster to use.",
			Remedy:   []string{"kubectl config use-context <name>", "kubectl config get-contexts   # to list them"},
			Observed: observed,
		}
	}

	if isKindContext {
		return Result{
			Status:  StatusWarn,
			Summary: fmt.Sprintf("this would deploy to %q, which is a cluster on this machine", cfg.CurrentContext),
			Detail: "You asked to deploy to a separate cluster, but the selected connection is " +
				"a local one.",
			Remedy: []string{
				"kubectl config use-context <the cluster you mean>",
				"Or run with --target local to deploy here on purpose.",
			},
			Observed: observed,
		}
	}

	return Result{
		Status:   StatusInfo,
		Summary:  fmt.Sprintf("this would deploy to %q at %s", cfg.CurrentContext, server),
		Detail:   "Check this is the cluster you mean — deploying is not easily undone.",
		Observed: observed,
	}
}

func (c kubeconfig) serverFor(contextName string) string {
	var clusterName string
	for _, ctx := range c.Contexts {
		if ctx.Name == contextName {
			clusterName = ctx.Context.Cluster
			break
		}
	}
	for _, cluster := range c.Clusters {
		if cluster.Name == clusterName {
			return cluster.Cluster.Server
		}
	}
	return "(unknown)"
}

// resolveKubeconfig mirrors magefile.go's commandEnv. Keep the two in step: if that
// function changes, this one is wrong until it is changed too.
func resolveKubeconfig() (path, source string) {
	if strings.EqualFold(automation.Env("CLUSTER_TYPE", "kind"), "k3s") {
		return automation.ExpandPath(automation.Env("K3S_LOCAL_KUBECONFIG", "~/.kube/k3s-aws.yaml")),
			"pinned by CLUSTER_TYPE=k3s"
	}

	raw := os.Getenv("KUBECONFIG")
	if raw == "" {
		return automation.ExpandPath("~/.kube/config"), "kubectl default"
	}

	// KUBECONFIG may list several files that kubectl merges. Parsing one file cannot
	// reproduce a merge, so take the first that exists and say which was used.
	entries := filepath.SplitList(raw)
	for _, entry := range entries {
		expanded := automation.ExpandPath(entry)
		if _, err := os.Stat(expanded); err == nil {
			if len(entries) > 1 {
				return expanded, fmt.Sprintf("first of %d entries in KUBECONFIG", len(entries))
			}
			return expanded, "KUBECONFIG"
		}
	}
	return automation.ExpandPath(entries[0]), "KUBECONFIG"
}
