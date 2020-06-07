package clusterdictionary

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	// SizeAlefDev is the definition of a cluster supporting dev purposes.
	SizeAlefDev = "SizeAlefDev"
	// SizeAlef500 is the key representing a cluster supporting 500 users.
	SizeAlef500 = "SizeAlef500"
	// SizeAlef1000 is the key representing a cluster supporting 1000 users.
	SizeAlef1000 = "SizeAlef1000"
	// SizeAlef5000 is the key representing a cluster supporting 5000 users.
	SizeAlef5000 = "SizeAlef5000"
	// SizeAlef10000 is the key representing a cluster supporting 10000 users.
	SizeAlef10000 = "SizeAlef10000"
)

type size struct {
	MasterInstanceType string
	MasterCount        int64
	NodeInstanceType   string
	NodeMinCount       int64
	NodeMaxCount       int64
}

// validSizes is a mapping of a size keyword to kops cluster configuration.
var validSizes = map[string]size{
	SizeAlefDev:   sizeAlefDev,
	SizeAlef500:   sizeAlef500,
	SizeAlef1000:  sizeAlef1000,
	SizeAlef5000:  sizeAlef5000,
	SizeAlef10000: sizeAlef10000,
}

// sizeAlefDev is a cluster sized for development and testing.
var sizeAlefDev = size{
	MasterInstanceType: "t3.medium",
	MasterCount:        1,
	NodeInstanceType:   "t3.medium",
	NodeMinCount:       2,
	NodeMaxCount:       2,
}

// sizeAlef500 is a cluster sized for 500 users.
var sizeAlef500 = size{
	MasterInstanceType: "t3.medium",
	MasterCount:        1,
	NodeInstanceType:   "m5.large",
	NodeMinCount:       2,
	NodeMaxCount:       2,
}

// sizeAlef1000 is a cluster sized for 1000 users.
var sizeAlef1000 = size{
	MasterInstanceType: "t3.large",
	MasterCount:        1,
	NodeInstanceType:   "m5.large",
	NodeMinCount:       4,
	NodeMaxCount:       4,
}

// sizeAlef5000 is a cluster sized for 5000 users.
var sizeAlef5000 = size{
	MasterInstanceType: "t3.large",
	MasterCount:        1,
	NodeInstanceType:   "m5.large",
	NodeMinCount:       6,
	NodeMaxCount:       6,
}

// sizeAlef10000 is a cluster sized for 10000 users.
var sizeAlef10000 = size{
	MasterInstanceType: "t3.large",
	MasterCount:        3,
	NodeInstanceType:   "m5.large",
	NodeMinCount:       10,
	NodeMaxCount:       10,
}

// IsValidClusterSize returns true if the given size string is supported.
func IsValidClusterSize(size string) bool {
	_, ok := validSizes[size]
	return ok
}

// ApplyToCreateClusterRequest takes a size keyword and applies the corresponding
// cluster values to a CreateClusterRequest.
func ApplyToCreateClusterRequest(size string, request *model.CreateClusterRequest) error {
	if len(size) == 0 {
		return nil
	}

	if !IsValidClusterSize(size) {
		return errors.Errorf("%s is not a valid size", size)
	}

	values := validSizes[size]
	request.MasterInstanceType = values.MasterInstanceType
	request.MasterCount = values.MasterCount
	request.NodeInstanceType = values.NodeInstanceType
	request.NodeMinCount = values.NodeMinCount
	request.NodeMaxCount = values.NodeMaxCount

	return nil
}

// ApplyToPatchClusterSizeRequest takes a size keyword and applies the
// corresponding cluster values to a PatchClusterSizeRequest.
func ApplyToPatchClusterSizeRequest(size string, request *model.PatchClusterSizeRequest) error {
	if len(size) == 0 {
		return nil
	}

	if !IsValidClusterSize(size) {
		return errors.Errorf("%s is not a valid size", size)
	}

	values := validSizes[size]
	request.NodeInstanceType = &values.NodeInstanceType
	request.NodeMinCount = &values.NodeMinCount
	request.NodeMaxCount = &values.NodeMaxCount

	return nil
}
