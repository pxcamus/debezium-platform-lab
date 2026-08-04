package preflight

// Catalog returns the machine checks for a target, in the order they are reported.
//
// Ordered most-fundamental first: a machine with no container runtime is not going to
// deploy anything, and that finding should be visible before the operator reads about
// registry latency. Every ID here has a matching heading anchor on the troubleshooting
// page, which the documentation build enforces.
func Catalog(target Target) []Check {
	return []Check{
		ContainerRuntime(),
		ToolingVersions(target),
		Memory(target),
		InotifyLimits(target),
		IngressPorts(target),
		ImagePull(target),
		DNSResolution(target),
		EnvShadowing(),
		ClusterMismatch(target),
	}
}
