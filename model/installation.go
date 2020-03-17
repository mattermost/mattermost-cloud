package model

import (
	"encoding/json"
	"fmt"
	"io"
)

// Installation represents a Mattermost installation.
type Installation struct {
	ID             string
	OwnerID        string
	GroupID        *string
	GroupSequence  *int64 `json:"GroupSequence,omitempty"`
	Version        string
	DNS            string
	Database       string
	Filestore      string
	License        string
	MattermostEnv  EnvVarMap
	Size           string
	Affinity       string
	State          string
	CreateAt       int64
	DeleteAt       int64
	LockAcquiredBy *string
	LockAcquiredAt int64
	GroupOverrides map[string]string `json:"GroupOverrides,omitempty"`

	// configconfigMergedWithGroup is set when the installation configuration
	// has been overridden with group configuration. This value can then be
	// checked later to determine whether the installation is safe to save or
	// not.
	configMergedWithGroup bool
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
func (i *Installation) Clone() *Installation {
	var clone Installation
	data, _ := json.Marshal(i)
	json.Unmarshal(data, &clone)

	return &clone
}

// IsInGroup returns if the installation is in a group or not.
func (i *Installation) IsInGroup() bool {
	return i.GroupID != nil
}

// ConfigMergedWithGroup returns if the installation currently has inherited
// group configuration values.
func (i *Installation) ConfigMergedWithGroup() bool {
	return i.configMergedWithGroup
}

// MergeWithGroup merges an installation's configuration with that of a group.
// An option can be provided to include a group override summary to the
// installation.
func (i *Installation) MergeWithGroup(group *Group, includeOverrides bool) {
	if i.ConfigMergedWithGroup() {
		return
	}
	if group == nil {
		return
	}

	i.configMergedWithGroup = true
	i.GroupOverrides = make(map[string]string)
	if group.MattermostEnv != nil && i.MattermostEnv == nil {
		i.MattermostEnv = make(EnvVarMap)
	}

	if i.Version != group.Version {
		i.Version = group.Version
		if includeOverrides {
			i.GroupOverrides["Installation Version"] = i.Version
			i.GroupOverrides["Group Version"] = group.Version
		}
	}
	for key, value := range group.MattermostEnv {
		if includeOverrides {
			if _, ok := i.MattermostEnv[key]; ok {
				i.GroupOverrides[fmt.Sprintf("Installation MattermostEnv[%s]", key)] = i.MattermostEnv[key].Value
				i.GroupOverrides[fmt.Sprintf("Group MattermostEnv[%s]", key)] = value.Value
			}
		}
		i.MattermostEnv[key] = value
	}
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
