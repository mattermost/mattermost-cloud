package model

const (
	// AllPerPage signals the store to return all results, avoid pagination of any kind.
	AllPerPage = -1

	// NoInstallationsLimit signals the store to return all multitenant database instances independently
	// of the number of installations using each instance.
	NoInstallationsLimit = -1
)
