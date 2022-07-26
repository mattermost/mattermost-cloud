// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ProvisionerSizePrefix is a provisioner specific Installation size prefix.
const ProvisionerSizePrefix = "provisioner"

// SizeProvisionerXL specifies custom Installation size.
const SizeProvisionerXL = "provisionerXL"

// GetInstallationSize returns Installation size based on its name.
func GetInstallationSize(size string) (v1alpha1.ClusterInstallationSize, error) {
	// We check first if it is one of Operator sizes, if not we expect custom
	// provisioner size.
	mmSize, err := v1alpha1.GetClusterSize(size)
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
func ParseProvisionerSize(size string) (v1alpha1.ClusterInstallationSize, error) {
	parts := strings.Split(size, "-")

	var resources v1alpha1.ClusterInstallationSize
	switch parts[0] {
	case SizeProvisionerXL:
		resources = SizeProvisionerXLResources
	default:
		return v1alpha1.ClusterInstallationSize{}, errors.Errorf("unrecognized installation size %q", parts[0])
	}

	if len(parts) == 1 {
		return resources, nil
	}
	if len(parts) > 2 {
		return v1alpha1.ClusterInstallationSize{}, errors.Errorf("expected at most 2 size segments found %d", len(parts))
	}
	if strings.TrimSpace(parts[1]) == "" {
		return v1alpha1.ClusterInstallationSize{}, errors.Errorf("replicas segment cannot be empty")
	}

	replicas, err := strconv.Atoi(parts[1])
	if err != nil {
		return v1alpha1.ClusterInstallationSize{}, errors.Wrap(err, "failed to parse number of replicas from custom provisioner size")
	}

	resources.App.Replicas = int32(replicas)

	return resources, nil
}

// SizeProvisionerXLResources specifies resources for Installation size.
var SizeProvisionerXLResources = v1alpha1.ClusterInstallationSize{
	App: v1alpha1.ComponentSize{
		Replicas: 4,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4000m"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
	},
	Minio: v1alpha1.ComponentSize{
		Replicas: 4,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
		},
	},
	Database: v1alpha1.ComponentSize{
		Replicas: 3,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
		},
	},
}
