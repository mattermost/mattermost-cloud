// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

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

// GroupDTOFromReader decodes a json-encoded group DTO from the given io.Reader.
func GroupDTOFromReader(reader io.Reader) (*GroupDTO, error) {
	groupDTO := GroupDTO{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&groupDTO)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &groupDTO, nil
}

// GroupDTOsFromReader decodes a json-encoded list of group DTOs from the given io.Reader.
func GroupDTOsFromReader(reader io.Reader) ([]*GroupDTO, error) {
	groupDTOs := []*GroupDTO{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&groupDTOs)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return groupDTOs, nil
}
