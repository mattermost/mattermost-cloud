package model

const (
	// SizeAlef500 is the first definition of a cluster supporting 500 users.
	SizeAlef500 = "SizeAlef500"
	// SizeAlef1000 is the second definition of a cluster supporting 1000 users.
	SizeAlef1000 = "SizeAlef1000"
	// SizeAlef5000 is the third definition of a cluster supporting 5000 users.
	SizeAlef5000 = "SizeAlef5000"
)

// IsSupportedClusterSize returns true if the given size string is supported.
func IsSupportedClusterSize(size string) bool {
	validSuffixes := []string{"", "-HA2", "-HA3"}
	for _, suffix := range validSuffixes {
		switch size {
		case
			SizeAlef500 + suffix,
			SizeAlef1000 + suffix,
			SizeAlef5000 + suffix:
			return true
		}
	}

	return false
}
