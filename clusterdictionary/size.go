// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package clusterdictionary

import (
	"strconv"
	"strings"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gookit/goutil/arrutil"
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

// ValidSizes is a mapping of a size keyword to kops cluster configuration.
var ValidSizes = map[string]size{
	SizeAlefDev:   sizeAlefDev,
	SizeAlef500:   sizeAlef500,
	SizeAlef1000:  sizeAlef1000,
	SizeAlef5000:  sizeAlef5000,
	SizeAlef10000: sizeAlef10000,
}

// sizeAlefDev is a cluster sized for development and testing.
var sizeAlefDev = size{
	MasterInstanceType: "t3.small",
	MasterCount:        1,
	NodeInstanceType:   "t3.medium",
	NodeMinCount:       2,
	NodeMaxCount:       2,
}

// sizeAlef500 is a cluster sized for 500 users.
var sizeAlef500 = size{
	MasterInstanceType: "t3.small",
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
	_, ok := ValidSizes[size]
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

	values := ValidSizes[size]
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

	values := ValidSizes[size]
	request.NodeInstanceType = &values.NodeInstanceType
	request.NodeMinCount = &values.NodeMinCount
	request.NodeMaxCount = &values.NodeMaxCount

	return nil
}

func processCustomSize(size string) (string, int64, int64, error) {
	if len(size) == 0 {
		return "", 0, 0, nil
	}

	parts := strings.Split(size, ";")
	ngType := ec2Types.InstanceType(parts[0])

	if !arrutil.In[ec2Types.InstanceType](ngType, ngType.Values()) {
		return "", 0, 0, errors.Errorf("%s is not a valid InstanceType", ngType)
	}

	minCount := 2
	maxCount := 2

	for _, part := range parts[1:] {
		switch {
		case strings.HasPrefix(part, "min="):
			minCount, _ = strconv.Atoi(strings.TrimPrefix(part, "min="))
		case strings.HasPrefix(part, "max="):
			maxCount, _ = strconv.Atoi(strings.TrimPrefix(part, "max="))
		}
	}

	if minCount < 1 {
		minCount = 1
	}

	if minCount > maxCount {
		maxCount = minCount
	}

	return string(ngType), int64(minCount), int64(maxCount), nil
}

// AddToCreateClusterRequest takes a map of size keywords and adds the corresponding
// values to a CreateClusterRequest.
func AddToCreateClusterRequest(sizes map[string]string, request *model.CreateClusterRequest) error {
	if len(sizes) == 0 {
		return nil
	}

	if request.AdditionalNodeGroups == nil {
		request.AdditionalNodeGroups = make(map[string]model.NodeGroupMetadata)
	}

	for ng, ngSize := range sizes {
		ngType, minCount, maxCount, err := processCustomSize(ngSize)
		if err != nil {
			return errors.Wrapf(err, "invalid nodegroup size for %s", ng)
		}

		request.AdditionalNodeGroups[ng] = model.NodeGroupMetadata{
			InstanceType: ngType,
			MinCount:     minCount,
			MaxCount:     maxCount,
		}
	}

	return nil
}

// AddToCreateNodegroupsRequest takes a map of size keywords and adds the corresponding
// values to a CreateNodegroupsRequest.
func AddToCreateNodegroupsRequest(sizes map[string]string, request *model.CreateNodegroupsRequest) error {
	if len(sizes) == 0 {
		return nil
	}

	if request.Nodegroups == nil {
		request.Nodegroups = make(map[string]model.NodeGroupMetadata)
	}

	for ng, ngSize := range sizes {
		ngType, minCount, maxCount, err := processCustomSize(ngSize)
		if err != nil {
			return errors.Wrapf(err, "invalid nodegroup size for %s", ng)
		}

		request.Nodegroups[ng] = model.NodeGroupMetadata{
			InstanceType: ngType,
			MinCount:     minCount,
			MaxCount:     maxCount,
		}
	}

	return nil
}
