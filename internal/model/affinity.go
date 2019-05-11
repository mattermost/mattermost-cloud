package model

const (
	// InstallationAffinityIsolated meanes that no peer installations are allowed in the same cluster.
	InstallationAffinityIsolated = "isolated"
)

// IsSupportedAffinity returns true if the given affinity string is supported.
func IsSupportedAffinity(affinity string) bool {
	return affinity == InstallationAffinityIsolated
}
