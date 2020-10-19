// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

// InstallationDTO represents a Mattermost installation. DTO stands for Data Transfer Object.
type InstallationDTO struct {
	*Installation
	Annotations []*Annotation `json:"Annotations,omitempty"`
}

// InstallationDTOFromReader decodes a json-encoded installation DTO from the given io.Reader.
func InstallationDTOFromReader(reader io.Reader) (*InstallationDTO, error) {
	installationDTO := InstallationDTO{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&installationDTO)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &installationDTO, nil
}

// InstallationDTOsFromReader decodes a json-encoded list of installation DTOs from the given io.Reader.
func InstallationDTOsFromReader(reader io.Reader) ([]*InstallationDTO, error) {
	installationDTOs := []*InstallationDTO{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&installationDTOs)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return installationDTOs, nil
}
