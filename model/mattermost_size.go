// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"strconv"
	"strings"

	mmv1beta1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ProvisionerSizePrefix is a provisioner specific Installation size prefix.
const ProvisionerSizePrefix = "provisioner"

// SizeProvisionerXL specifies custom Installation size.
const SizeProvisionerXL = "provisionerXL"

// GetInstallationSize returns Installation size based on its name.
func GetInstallationSize(size string) (mmv1beta1.Size, error) {
	// We check first if it is one of Operator sizes, if not we expect custom
	// provisioner size.
	mmSize, err := mmv1beta1.GetClusterSize(size)
	if err == nil {
		return mmSize, nil
	}

	return ParseProvisionerSize(size)
}

// ParseProvisionerSize parses Provisioner specific Installation size with
// configurable replicas count.
// The size should be specified in form:
// [SIZE_NAME]-[NUMBER_OF_REPLICAS]
// If number of replicas is not specified the default value for the size will
// be used.
func ParseProvisionerSize(size string) (mmv1beta1.Size, error) {
	parts := strings.Split(size, "-")

	var resources mmv1beta1.Size
	switch parts[0] {
	case SizeProvisionerXL:
		resources = SizeProvisionerXLResources
	default:
		return mmv1beta1.Size{}, errors.Errorf("unrecognized installation size %q", parts[0])
	}

	if len(parts) == 1 {
		return resources, nil
	}
	if len(parts) > 2 {
		return mmv1beta1.Size{}, errors.Errorf("expected at most 2 size segments found %d", len(parts))
	}
	if strings.TrimSpace(parts[1]) == "" {
		return mmv1beta1.Size{}, errors.Errorf("replicas segment cannot be empty")
	}

	replicas, err := strconv.Atoi(parts[1])
	if err != nil {
		return mmv1beta1.Size{}, errors.Wrap(err, "failed to parse number of replicas from custom provisioner size")
	}

	resources.App.Replicas = int32(replicas)

	return resources, nil
}

// SizeProvisionerXLResources specifies resources for Installation size.
// Size value = 25000users with 4 Replicas
var SizeProvisionerXLResources = mmv1beta1.Size{
	App: mmv1beta1.ComponentSize{
		Replicas: 4,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4000m"),
				corev1.ResourceMemory: resource.MustParse("16Gi"),
			},
		},
	},
}
