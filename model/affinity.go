// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

const (
	// InstallationAffinityIsolated means that no peer installations are allowed in the same cluster.
	InstallationAffinityIsolated = "isolated"
	// InstallationAffinityMultiTenant means peer installations are allowed in the same cluster.
	InstallationAffinityMultiTenant = "multitenant"
)

// IsSupportedAffinity returns true if the given affinity string is supported.
func IsSupportedAffinity(affinity string) bool {
	return affinity == InstallationAffinityIsolated || affinity == InstallationAffinityMultiTenant
}
