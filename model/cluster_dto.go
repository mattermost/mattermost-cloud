// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

// ClusterDTO represents cluster entity with connected data. DTO stands for Data Transfer Object.
type ClusterDTO struct {
	*Cluster
	Annotations []*Annotation `json:"Annotations,omitempty"`
}
