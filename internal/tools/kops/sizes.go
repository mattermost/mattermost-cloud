package kops

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/model"
)

const haSeparator = "-HA"

// ClusterSize is sizing configuration used by kops at cluster creation.
type ClusterSize struct {
	NodeCount   string
	NodeSize    string
	MasterCount string
	MasterSize  string
}

// validSizes is a mapping of a size keyword to kops cluster configuration.
var validSizes = map[string]ClusterSize{
	model.SizeAlef500:  sizeAlef500,
	model.SizeAlef1000: sizeAlef1000,
	model.SizeAlef5000: sizeAlef5000,
}

// sizeAlef500 is a cluster sized for 500 users.
var sizeAlef500 = ClusterSize{
	NodeCount:   "2",
	NodeSize:    "t2.medium",
	MasterCount: "1",
	MasterSize:  "t2.medium",
}

// sizeAlef1000 is a cluster sized for 1000 users.
var sizeAlef1000 = ClusterSize{
	NodeCount:   "4",
	NodeSize:    "t2.medium",
	MasterCount: "1",
	MasterSize:  "t2.large",
}

// sizeAlef5000 is a cluster sized for 5000 users.
var sizeAlef5000 = ClusterSize{
	NodeCount:   "6",
	NodeSize:    "t2.large",
	MasterCount: "1",
	MasterSize:  "t2.large",
}

// GetSize takes a size keyword and returns the matching kops cluster
// configuration.
func GetSize(size string) (ClusterSize, error) {
	parsedSize, masterCount, err := parseHA(size)
	if err != nil {
		return ClusterSize{}, err
	}
	kopsClusterSize, ok := validSizes[parsedSize]
	if !ok {
		return ClusterSize{}, fmt.Errorf("unsupported size %s", size)
	}
	kopsClusterSize.MasterCount = masterCount

	return kopsClusterSize, nil
}

// parseHA parses a given size value to determine HA master requirements.
// Example:
//   Single master node for "SizeAlef1000" is "SizeAlef1000"
//   Three master nodes for "SizeAlef1000" is "SizeAlef1000-HA3"
func parseHA(size string) (string, string, error) {
	if !strings.Contains(size, haSeparator) {
		return size, "1", nil
	}

	splitSize := strings.Split(size, haSeparator)
	if len(splitSize) != 2 {
		return "", "", fmt.Errorf("incorrect HA syntax on size %s", size)
	}

	// Hardcode max master count to "2" or "3" for now.
	if splitSize[1] != "2" && splitSize[1] != "3" {
		return "", "", fmt.Errorf("invalid HA syntax on size %s", size)
	}

	return splitSize[0], splitSize[1], nil
}
