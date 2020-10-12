// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

// ClusterDTO represents cluster entity with connected data. DTO stands for Data Transfer Object.
type ClusterDTO struct {
	*Cluster
	Annotations []*Annotation `json:"Annotations,omitempty"`
}

// ClusterDTOFromReader decodes a json-encoded cluster DTO from the given io.Reader.
func ClusterDTOFromReader(reader io.Reader) (*ClusterDTO, error) {
	clusterDTO := ClusterDTO{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&clusterDTO)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &clusterDTO, nil
}

// ClusterDTOsFromReader decodes a json-encoded list of cluster DTOs from the given io.Reader.
func ClusterDTOsFromReader(reader io.Reader) ([]*ClusterDTO, error) {
	clusterDTOs := []*ClusterDTO{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&clusterDTOs)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return clusterDTOs, nil
}
