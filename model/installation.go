package model

import (
	"encoding/json"
	"io"
)

// Installation represents a Mattermost installation.
type Installation struct {
	ID             string
	OwnerID        string
	Version        string
	DNS            string
	Database       string
	Filestore      string
	License        string
	Size           string
	Affinity       string
	GroupID        *string
	State          string
	CreateAt       int64
	DeleteAt       int64
	LockAcquiredBy *string
	LockAcquiredAt int64
}

// InstallationFilter describes the parameters used to constrain a set of installations.
type InstallationFilter struct {
	OwnerID        string
	GroupID        string
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// Clone returns a deep copy the installation.
func (c *Installation) Clone() *Installation {
	var clone Installation
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &clone)

	return &clone
}

// InstallationFromReader decodes a json-encoded installation from the given io.Reader.
func InstallationFromReader(reader io.Reader) (*Installation, error) {
	installation := Installation{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&installation)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &installation, nil
}

// InstallationsFromReader decodes a json-encoded list of installations from the given io.Reader.
func InstallationsFromReader(reader io.Reader) ([]*Installation, error) {
	installations := []*Installation{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&installations)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return installations, nil
}
