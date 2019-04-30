package model

const (
	// SizeAlef500 is the first definition of a cluster supporting 500 users.
	SizeAlef500 = "SizeAlef500"
	// SizeAlef1000 is the second definition of a cluster supporting 1000 users.
	SizeAlef1000 = "SizeAlef1000"
)

// IsSupportedSize returns true if the given size string is supported.
func IsSupportedSize(size string) bool {
	switch size {
	case SizeAlef500, SizeAlef1000:
		return true
	}

	return false
}
