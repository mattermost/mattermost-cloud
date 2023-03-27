// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

// GroupDTO is a group with Annotations.
type GroupDTO struct {
	*Group
	Annotations       []*Annotation `json:"annotations"`
	InstallationCount *int64        `json:"installation_count,omitempty"`
}

// GetInstallationCount retrieves the installation count dereferenced value
func (g GroupDTO) GetInstallationCount() int64 {
	return *g.InstallationCount
}
