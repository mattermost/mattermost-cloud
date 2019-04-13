package kops

import "fmt"

// ClusterSize is sizing configuration used by kops at cluster creation.
type ClusterSize struct {
	NodeCount  string
	NodeSize   string
	MasterSize string
}

// validSizes is a mapping of a size keyword to kops cluster configuration.
var validSizes = map[string]ClusterSize{
	"SizeAlef500":  sizeAlef500,
	"SizeAlef1000": sizeAlef1000,
}

// sizeAlef500 is a cluster sized for 500 users.
var sizeAlef500 = ClusterSize{
	NodeCount:  "2",
	NodeSize:   "t2.medium",
	MasterSize: "m3.medium",
}

// sizeAlef1000 is a cluster sized for 1000 users.
var sizeAlef1000 = ClusterSize{
	NodeCount:  "4",
	NodeSize:   "t2.medium",
	MasterSize: "m4.large",
}

// GetSize takes a size keyword and returns the matching kops cluster
// configuration.
func GetSize(size string) (ClusterSize, error) {
	kopsClusterSize, ok := validSizes[size]
	if !ok {
		return ClusterSize{}, fmt.Errorf("unsupported size %s", size)
	}

	return kopsClusterSize, nil
}
