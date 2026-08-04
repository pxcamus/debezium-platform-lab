package preflight

// Catalog returns the checks that run against the machine itself, in the order they are
// reported.
//
// Order is cheapest-and-most-fundamental first: a machine with no container runtime is not
// going to deploy anything, so that finding should be visible before the operator reads
// about registry latency.
//
// The catalog is deliberately partial. The remaining identifiers on the troubleshooting
// page — tooling-versions, inotify-limits, ingress-ports-busy, dns-resolution,
// env-shadowing, cluster-mismatch — are specified but not yet implemented, and are added
// once this shape has survived contact with a real machine.
func Catalog() []Check {
	return []Check{
		ContainerRuntime(),
		Memory(),
		ImagePull(),
	}
}
