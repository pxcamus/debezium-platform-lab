package cluster

import (
	"fmt"

	"dbz-mage/ko/automation"
)

func NewFromEnv() (Cluster, error) {
	switch automation.Env("CLUSTER_TYPE", "kind") {
	case "kind":
		return NewKindFromEnv(), nil
	case "k3s":
		return NewK3sFromEnv()
	default:
		return nil, fmt.Errorf("unsupported CLUSTER_TYPE %q", automation.Env("CLUSTER_TYPE", "kind"))
	}
}
